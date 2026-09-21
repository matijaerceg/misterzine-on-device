//go:build linux

package updater

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func bootID() string {
	b, _ := os.ReadFile("/proc/sys/kernel/random/boot_id")
	return strings.TrimSpace(string(b))
}
func restarted(s State) bool { return s.Boot != "" && s.Boot != bootID() }
func workerAlive(s State) bool {
	if s.PID <= 0 || s.Boot != bootID() {
		return false
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", s.PID))
	return err == nil && strings.Contains(string(b), "update-worker\x00") && strings.Contains(string(b), s.ID+"\x00")
}

// OtherScript guards the updater's shared temporary paths.
func OtherScript() bool {
	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		b, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(strings.TrimRight(string(b), "\x00"), "\x00")
		for _, a := range args {
			base := filepath.Base(a)
			if base == "update_all.sh" || base == "update_all.pyz" || base == "ua_downloader_bin" || base == "ua_downloader_dd.pyz" || base == "ua_downloader_latest.zip" || base == "downloader_bin" || base == "downloader.sh" || base == "update.sh" {
				return true
			}
		}
	}
	return false
}

func lock(root string) (*os.File, error) {
	if err := os.MkdirAll(stateDir(root), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(stateDir(root), "lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("an update is already running")
	}
	return f, nil
}

// scriptFor is what a mode runs: Update All's script, or Downloader's
// launcher told to run the misterzine database alone (it passes its
// arguments through to Downloader, which reads the card's downloader.ini
// and keeps its usual store, so nothing else on the card is touched).
func scriptFor(card, mode string) (string, []string, error) {
	if mode == ModeApp {
		s := filepath.Join(card, "Scripts", "downloader.sh")
		if _, err := os.Stat(s); err != nil {
			return "", nil, fmt.Errorf("Downloader is not installed in Scripts")
		}
		return s, []string{"--run-only", "misterzine"}, nil
	}
	s := filepath.Join(card, "Scripts", "update_all.sh")
	if _, err := os.Stat(s); err != nil {
		return "", nil, fmt.Errorf("Update All is not installed in Scripts")
	}
	return s, nil, nil
}

// Start copies the supervisor so a package update cannot replace the executable
// from under this run. FD 3 transfers ownership of the run lock to that process.
func Start(root, card, mode string) (State, error) {
	if s, err := Read(root); err == nil && s.Active() {
		return s, nil
	}
	f, err := lock(root)
	if err != nil {
		return State{}, err
	}
	defer f.Close()
	if OtherScript() {
		return State{}, fmt.Errorf("another updater is running; let it finish first")
	}
	if _, _, err := scriptFor(card, mode); err != nil {
		return State{}, err
	}
	id := strconv.FormatInt(time.Now().UnixNano(), 36)
	tmp, err := os.MkdirTemp("", "misterzine-update-")
	if err != nil {
		return State{}, err
	}
	worker := filepath.Join(tmp, "worker")
	started := false
	defer func() {
		if !started {
			os.Remove(worker)
			os.Remove(tmp)
		}
	}()
	from, err := os.Open("/proc/self/exe")
	if err != nil {
		return State{}, err
	}
	to, err := os.OpenFile(worker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		from.Close()
		return State{}, err
	}
	_, err = io.Copy(to, from)
	from.Close()
	closeErr := to.Close()
	if err != nil {
		return State{}, err
	}
	if closeErr != nil {
		return State{}, closeErr
	}
	cmd := exec.Command(worker, "update-worker", root, card, id, mode)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.ExtraFiles = []*os.File{f}
	if err = cmd.Start(); err != nil {
		os.Remove(worker)
		os.Remove(tmp)
		return State{}, err
	}
	started = true
	go cmd.Wait()
	for i := 0; i < 60; i++ {
		time.Sleep(50 * time.Millisecond)
		if s, err := Read(root); err == nil && s.ID == id {
			return s, nil
		}
	}
	s := State{ID: id, Mode: mode, PID: cmd.Process.Pid, Boot: bootID(), Status: "starting", Message: "Supervisor started; waiting for status", Started: time.Now()}
	s.Label = "Starting " + s.Name()
	if workerAlive(s) {
		return s, nil
	}
	return State{}, fmt.Errorf("updater supervisor exited before publishing status; check the update log and card")
}

func Cancel(root, id string) error {
	s, err := Read(root)
	if err != nil {
		return err
	}
	if !s.Active() || s.ID != id {
		return fmt.Errorf("that update is no longer running")
	}
	return os.WriteFile(filepath.Join(stateDir(root), "cancel-"+id), []byte("cancel\n"), 0600)
}

// outputSink keeps draining even when the UI disappears. Carriage-return
// progress and partial output are flushed at the worker's next heartbeat.
type outputSink struct {
	mu          sync.Mutex
	s           State
	parser      outputParser
	pending     []byte
	log         *os.File
	logPending  []byte
	logBytes    int
	writerGuard bool
}

func (o *outputSink) line() {
	if len(o.pending) == 0 {
		return
	}
	line := o.s.recordOutput(string(o.pending), time.Now())
	o.pending = o.pending[:0]
	if line == "" || o.log == nil {
		return
	}
	// Keep pipe draining independent of card latency. The retained log already
	// rotates at 2 MB; cap queued output too if the card stalls for a long time.
	if len(o.logPending)+len(line)+1 > 2<<20 {
		o.logPending = append([]byte("[older buffered output omitted during card stall]\n"), o.logPending[len(o.logPending)/2:]...)
	}
	o.logPending = append(o.logPending, line...)
	o.logPending = append(o.logPending, '\n')
}

// Called only by the supervisor loop, never while holding the output mutex.
func (o *outputSink) flushLog() {
	o.mu.Lock()
	pending := o.logPending
	o.logPending = nil
	o.mu.Unlock()
	if len(pending) == 0 || o.log == nil {
		return
	}
	if o.logBytes+len(pending) > 2<<20 {
		o.log.Truncate(0)
		o.log.Seek(0, 0)
		o.logBytes = 0
	}
	n, _ := o.log.Write(pending)
	o.logBytes += n
}

func (o *outputSink) Write(b []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(b) > 0 {
		o.s.LastOutput = time.Now()
	}
	o.parser.write(&o.s, b)
	for _, c := range b {
		if c == '\n' || c == '\r' {
			o.line()
		} else {
			o.pending = append(o.pending, c)
			if len(o.pending) >= 4096 {
				o.line()
			}
		}
	}
	return len(b), nil
}

// Preview an unfinished line without discarding it: the next pipe read may
// complete a stage announcement or an ANSI sequence split across writes.
func (o *outputSink) snapshot() State {
	line := cleanLine(string(o.pending))
	s := o.s
	s.Protected = s.Protected || o.writerGuard
	if line != "" {
		s.Lines = append(append([]string(nil), s.Lines...), line)
		if len(s.Lines) > TailLines {
			s.Lines = s.Lines[len(s.Lines)-TailLines:]
		}
	}
	return s
}

// systemWriter checks the actual command group too, so cancellation is not
// based solely on how quickly a stage's log output reached us.
func systemWriter(group int) bool {
	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		pg, err := syscall.Getpgid(pid)
		if err != nil || pg != group {
			continue
		}
		b, _ := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		args := strings.Split(strings.TrimRight(string(b), "\x00"), "\x00")
		for _, a := range args {
			base := filepath.Base(a)
			if base == "updateboot" || base == "update_linux.sh" || base == "update_linux" || base == "dd" || base == "flashcp" {
				return true
			}
		}
	}
	return false
}

// signalGroup uses the same protection for the initial TERM and later KILL.
// Check known writers before stopping the group, so a deferred cancellation
// does not repeatedly pause a writer. Recheck with the group stopped before
// signalling, allowing queued phase announcements to reach the output sink.
func (o *outputSink) signalGroup(group int, signal syscall.Signal) bool {
	writer := systemWriter(group)
	o.mu.Lock()
	o.writerGuard = writer
	protected := o.snapshot().Protected
	o.mu.Unlock()
	if protected {
		return false
	}
	if err := syscall.Kill(-group, syscall.SIGSTOP); err != nil {
		return false
	}
	defer syscall.Kill(-group, syscall.SIGCONT)
	time.Sleep(75 * time.Millisecond)
	writer = systemWriter(group)
	o.mu.Lock()
	defer o.mu.Unlock()
	o.writerGuard = writer
	if o.snapshot().Protected {
		return false
	}
	return syscall.Kill(-group, signal) == nil
}

// Worker runs in a detached copy of the binary, never on the script console.
// The entry point owns inherited lock FD 3 until this entire run is finished.
func Worker(root, card, id, mode string) int {
	lockFile := os.NewFile(3, "update-lock")
	defer lockFile.Close()
	syscall.CloseOnExec(3)
	exe, _ := os.Executable()
	defer os.Remove(filepath.Dir(exe))
	defer os.Remove(exe)
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, 5)
	s, args, err := scriptFor(card, mode)
	if err != nil {
		return 1
	}
	return runWorker(root, card, id, mode, s, args)
}

func runWorker(root, card, id, mode, script string, args []string) int {
	now := time.Now()
	o := &outputSink{s: State{ID: id, Mode: mode, PID: os.Getpid(), Boot: bootID(), Status: "starting", Started: now, Heartbeat: now, LastOutput: now}}
	o.s.Label = "Preparing " + o.s.Name()
	var err error
	o.log, err = os.OpenFile(LogPath(root), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return 1
	}
	defer o.log.Close()
	if err = saveCard(StatePath(root), o.s); err != nil {
		return 1
	}
	if err = save(livePath(root), o.s); err != nil {
		return 1
	}
	cmd := exec.Command("/bin/bash", append([]string{script}, args...)...)
	cmd.Dir = filepath.Join(card, "Scripts")
	cmd.Env = append(os.Environ(), "UPDATE_ALL_NON_INTERACTIVE=true", "PYTHONUNBUFFERED=1", "TERM=dumb", "COLUMNS=80", "LINES=24")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout, cmd.Stderr = o, o
	if err = cmd.Start(); err != nil {
		o.s.Status = "failed"
		o.s.Message = "Could not start " + o.s.Name() + ": " + err.Error()
		o.s.Finished = time.Now()
		saveCard(StatePath(root), o.s)
		save(livePath(root), o.s)
		return 1
	}
	o.mu.Lock()
	o.s.Status = "running"
	o.mu.Unlock()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	cancelPath := filepath.Join(stateDir(root), "cancel-"+id)
	defer os.Remove(cancelPath)
	var terminateAt time.Time
	checkpoint := time.Now()
	var checkpointReboot, checkpointSuccess, checkpointErrors bool
	for {
		select {
		case err = <-done:
			o.mu.Lock()
			o.line()
			o.s.Finished = time.Now()
			o.s.Elapsed = int(time.Since(now).Seconds())
			o.s.Heartbeat = time.Now()
			switch {
			case !terminateAt.IsZero():
				o.s.Status = "cancelled"
				o.s.Label = "Update cancelled"
				o.s.Message = "Update cancelled. Some files may have changed."
			case o.s.HadErrors:
				o.s.Status = "errors"
				o.s.Label = "Finished with errors"
				o.s.Message = "MiSTer was not fully updated. Review the log."
			case err != nil:
				o.s.Status = "failed"
				o.s.Label = "Update failed"
				o.s.Message = o.s.Name() + " failed. Review the output below."
			case !o.s.SawSuccess && o.s.Mode != ModeApp:
				// Update All announces its success in the output; Downloader
				// on its own does not, and says so with its exit code.
				o.s.Status = "failed"
				o.s.Label = "Update stopped"
				o.s.Message = "Updater exited without confirming success."
			default:
				o.s.Status = "completed"
				o.s.Stage = len(Stages)
				o.s.Label = "Update complete"
				if o.s.Reboot {
					o.s.Message = "Update complete; restart required"
				}
			}
			o.s.Protected = false
			s := o.s
			s.Lines = append([]string(nil), s.Lines...)
			o.mu.Unlock()
			saveCard(StatePath(root), s)
			save(livePath(root), s)
			o.flushLog()
			o.log.Sync()
			return 0
		case <-tick.C:
			o.mu.Lock()
			for _, marker := range []string{"/tmp/MiSTer_downloader_needs_reboot", "/tmp/downloader_needs_reboot_after_linux_update"} {
				if _, e := os.Stat(marker); e == nil {
					o.s.Reboot = true
				}
			}
			if _, e := os.Stat(cancelPath); e == nil {
				o.s.CancelRequested = true
			}
			cancel := o.s.CancelRequested
			o.mu.Unlock()
			if cancel && terminateAt.IsZero() && o.signalGroup(cmd.Process.Pid, syscall.SIGTERM) {
				terminateAt = time.Now()
				o.mu.Lock()
				o.s.Status = "cancelling"
				o.s.Message = "Cancelling " + o.s.Name()
				o.mu.Unlock()
			}
			if !terminateAt.IsZero() && time.Since(terminateAt) > 3*time.Second {
				o.signalGroup(cmd.Process.Pid, syscall.SIGKILL)
			}
			o.mu.Lock()
			o.s.Heartbeat = time.Now()
			o.s.Elapsed = int(time.Since(now).Seconds())
			s := o.snapshot()
			s.Lines = append([]string(nil), s.Lines...)
			o.mu.Unlock()
			// A slow SD flush must not hold the output lock. The copied state
			// remains stable while the pipe reader continues accepting output.
			save(livePath(root), s)
			o.flushLog()
			if time.Since(checkpoint) > 5*time.Second || s.Reboot != checkpointReboot || s.SawSuccess != checkpointSuccess || s.HadErrors != checkpointErrors {
				saveCard(StatePath(root), s)
				checkpointReboot, checkpointSuccess, checkpointErrors = s.Reboot, s.SawSuccess, s.HadErrors
				checkpoint = time.Now()
			}
		}
	}
}
