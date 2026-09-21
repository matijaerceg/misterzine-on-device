//go:build linux

package updater

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 6 && os.Args[1] == "update-worker" {
		if delay, err := time.ParseDuration(os.Getenv("MZ_TEST_WORKER_DELAY")); err == nil {
			time.Sleep(delay)
		}
		os.Exit(Worker(os.Args[2], os.Args[3], os.Args[4], os.Args[5]))
	}
	if len(os.Args) == 4 && os.Args[1] == "test-start" {
		s, err := Start(os.Args[2], os.Args[3], ModeAll)
		if err != nil {
			panic(err)
		}
		json.NewEncoder(os.Stdout).Encode(s)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func fakeCard(t *testing.T, script string) (string, string) {
	t.Helper()
	card := t.TempDir()
	root := filepath.Join(card, "misterzine")
	os.MkdirAll(filepath.Join(card, "Scripts"), 0700)
	if err := os.WriteFile(filepath.Join(card, "Scripts", "update_all.sh"), []byte("#!/bin/bash\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(livePath(root)) })
	return root, card
}

func waitState(t *testing.T, root string, want func(State) bool) State {
	t.Helper()
	until := time.Now().Add(10 * time.Second)
	var s State
	var err error
	for time.Now().Before(until) {
		s, err = Read(root)
		if err == nil && want(s) {
			return s
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout: %+v err %v", s, err)
	return s
}

func TestRunnerStreamsAndFinishes(t *testing.T) {
	root, card := fakeCard(t, "echo 'Running MiSTer Downloader'\nsleep .2\nprintf 'Downloading 1\\rDownloading 2\\n'\nsleep .2\necho 'Running Arcade Organizer'\necho 'Success! Log saved.'\n")
	s, err := Start(root, card, ModeAll)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID == "" {
		t.Fatal("missing run id")
	}
	s = waitState(t, root, func(s State) bool { return !s.Active() })
	if s.Status != "completed" || s.Stage != len(Stages) {
		t.Fatalf("%+v", s)
	}
	b, _ := os.ReadFile(LogPath(root))
	if !strings.Contains(string(b), "Downloading 2") {
		t.Fatal("lost output")
	}
	// The live copy disappears at reboot. The card checkpoint must preserve
	// the final result and log tail independently of that temporary copy.
	if err := os.Remove(livePath(root)); err != nil {
		t.Fatal(err)
	}
	recovered, err := Read(root)
	if err != nil || recovered.ID != s.ID || recovered.Status != "completed" || !recovered.SawSuccess || !strings.Contains(strings.Join(recovered.Lines, "\n"), "Downloading 2") {
		t.Fatalf("card-only result recovery failed: %+v, %v", recovered, err)
	}
}

func TestCancelAndDuplicateRun(t *testing.T) {
	root, card := fakeCard(t, "echo 'Running MiSTer Downloader'\ntrap '' TERM\nwhile true; do echo working; sleep .2; done\n")
	s, err := Start(root, card, ModeAll)
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, root, func(s State) bool { return len(s.Lines) > 0 })
	other, err := Start(root, card, ModeAll)
	if err != nil || other.ID != s.ID {
		t.Fatalf("duplicate run: %v %+v", err, other)
	}
	if err := Acknowledge(root, s.ID); err == nil {
		t.Fatal("a live worker was acknowledged as a dismissed recovery warning")
	}
	if err := Cancel(root, "wrong-run"); err == nil {
		t.Fatal("cancel accepted wrong id")
	}
	if err := Cancel(root, s.ID); err != nil {
		t.Fatal(err)
	}
	end := waitState(t, root, func(s State) bool { return !s.Active() })
	if end.Status != "cancelled" {
		t.Fatalf("%+v", end)
	}
}

func TestCancelWaitsForSystemWrite(t *testing.T) {
	// The warning starts past the display limit, crosses the 4096-byte
	// buffer flush, and finishes in a later pipe read before cancellation.
	root, card := fakeCard(t, "printf '%4090sLinux will be up' ''\nsleep .3\nprintf 'dated from distribution_mister:\\n'\nsleep 2\necho 'Linux has been updated!'\nsleep 8\necho 'Success! Log saved.'\n")
	s, err := Start(root, card, ModeAll)
	if err != nil {
		t.Fatal(err)
	}
	s = waitState(t, root, func(s State) bool { return s.Protected })
	if !s.Reboot {
		t.Fatal("no early restart flag")
	}
	if err := Cancel(root, s.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	s, err = Read(root)
	if err != nil || !s.Active() || !s.CancelRequested || !s.Protected {
		t.Fatalf("system write interrupted: %+v %v", s, err)
	}
	s = waitState(t, root, func(s State) bool { return !s.Active() })
	if s.Status != "cancelled" || !strings.Contains(strings.Join(s.Lines, "\n"), "Linux has been updated") {
		t.Fatalf("%+v", s)
	}
}

func TestUnexpectedExitNotSuccess(t *testing.T) {
	root, card := fakeCard(t, "echo something\nexit 0\n")
	if _, err := Start(root, card, ModeAll); err != nil {
		t.Fatal(err)
	}
	s := waitState(t, root, func(s State) bool { return !s.Active() })
	if s.Status != "failed" {
		t.Fatalf("%+v", s)
	}
}

func TestStaleWorkerIsInterrupted(t *testing.T) {
	root, _ := fakeCard(t, "")
	os.MkdirAll(stateDir(root), 0700)
	if err := saveCard(StatePath(root), State{ID: "old", Status: "running", PID: 123, Boot: "previous-boot"}); err != nil {
		t.Fatal(err)
	}
	s, err := Read(root)
	if err != nil || s.Status != "interrupted" {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestWorkerSurvivesStartingProcess(t *testing.T) {
	root, card := fakeCard(t, "echo 'Running MiSTer Downloader'\nsleep 1\nprintf 'Linux will be up'\nsleep .6\nprintf 'dated from distribution_mister:\\n'\nsleep 1\necho 'Linux has been updated!'\necho 'Success! Log saved.'\n")
	exe, _ := os.Executable()
	b, err := exec.Command(exe, "test-start", root, card).Output()
	if err != nil {
		t.Fatal(err)
	}
	var started State
	if err = json.Unmarshal(b, &started); err != nil {
		t.Fatal(err)
	}
	s := waitState(t, root, func(s State) bool { return s.Reboot })
	if s.ID != started.ID {
		t.Fatal("lost detached run")
	}
	s = waitState(t, root, func(s State) bool { return !s.Active() })
	if s.Status != "completed" {
		t.Fatalf("%+v", s)
	}
}

func TestSystemWriterWithoutStageOutput(t *testing.T) {
	root, card := fakeCard(t, "echo start\nbash ./dd\nsleep 8\necho 'Success! Log saved.'\n")
	if err := os.WriteFile(filepath.Join(card, "Scripts", "dd"), []byte("#!/bin/bash\ntrap 'echo writer-resumed' CONT\necho writing\nsleep 2\necho write-finished\n"), 0700); err != nil {
		t.Fatal(err)
	}
	s, err := Start(root, card, ModeAll)
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, root, func(s State) bool { return strings.Contains(strings.Join(s.Lines, "\n"), "writing") })
	if err = Cancel(root, s.ID); err != nil {
		t.Fatal(err)
	}
	s = waitState(t, root, func(s State) bool { return !s.Active() })
	if s.Status != "cancelled" || !strings.Contains(strings.Join(s.Lines, "\n"), "write-finished") {
		t.Fatalf("%+v", s)
	}
	if strings.Contains(strings.Join(s.Lines, "\n"), "writer-resumed") {
		t.Fatal("cancellation paused a writer that was already detectable")
	}
}

// These are ordinary shell fixtures in a temporary card directory. The fake
// writer starts only after TERM, so it exercises the later force-stop guard;
// it never invokes dd or writes firmware.
func TestCancelEscalationWaitsForLateSystemWrite(t *testing.T) {
	for _, kind := range []string{"announcement", "process"} {
		t.Run(kind, func(t *testing.T) {
			write := "echo 'Linux will be updated from distribution_mister:'\nsleep 5\necho 'Linux has been updated!'\necho write-finished\n"
			if kind == "process" {
				write = "bash ./dd\n"
			}
			root, card := fakeCard(t, "start_write=0\ntrap 'start_write=1' TERM\necho ready\nwhile [ \"$start_write\" = 0 ]; do sleep .1; done\ntrap '' TERM\n"+write+"sleep 8\necho 'Success! Log saved.'\n")
			if kind == "process" {
				if err := os.WriteFile(filepath.Join(card, "Scripts", "dd"), []byte("#!/bin/bash\necho writer-started\nsleep 5\necho write-finished\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			s, err := Start(root, card, ModeAll)
			if err != nil {
				t.Fatal(err)
			}
			waitState(t, root, func(s State) bool { return strings.Contains(strings.Join(s.Lines, "\n"), "ready") })
			if err := Cancel(root, s.ID); err != nil {
				t.Fatal(err)
			}
			waitState(t, root, func(s State) bool {
				lines := strings.Join(s.Lines, "\n")
				return strings.Contains(lines, "Linux will be updated") || strings.Contains(lines, "writer-started")
			})
			// Go past the ordinary three-second escalation deadline, but stay
			// within the fixture's five-second protected operation.
			time.Sleep(3500 * time.Millisecond)
			s, err = Read(root)
			if err != nil || !s.Active() || !s.CancelRequested || !s.Protected {
				t.Errorf("late system write was not protected from escalation: %+v, %v", s, err)
			}
			if !strings.Contains(s.Summary(), "finishing system update") {
				t.Errorf("UI does not explain why cancellation is waiting: %q", s.Summary())
			}
			s = waitState(t, root, func(s State) bool { return !s.Active() })
			if s.Status != "cancelled" || !strings.Contains(strings.Join(s.Lines, "\n"), "write-finished") {
				t.Fatalf("cancellation must wait for the write, then finish: %+v", s)
			}
		})
	}
}

func TestReportedPartialFailure(t *testing.T) {
	root, card := fakeCard(t, "echo 'There were some errors in the Updaters.'\nexit 0\n")
	if _, err := Start(root, card, ModeAll); err != nil {
		t.Fatal(err)
	}
	s := waitState(t, root, func(s State) bool { return !s.Active() })
	if s.Status != "errors" {
		t.Fatalf("%+v", s)
	}
}

func TestRestartRecovery(t *testing.T) {
	root, _ := fakeCard(t, "")
	os.MkdirAll(stateDir(root), 0700)
	if err := saveCard(StatePath(root), State{ID: "old", Status: "running", PID: 123, Boot: "previous-boot", SawSuccess: true, Reboot: true}); err != nil {
		t.Fatal(err)
	}
	s, err := Read(root)
	if err != nil || s.Status != "restarted" || s.Reboot {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestSlowWorkerStartupRemainsActive(t *testing.T) {
	t.Setenv("MZ_TEST_WORKER_DELAY", "4s")
	root, card := fakeCard(t, "echo 'Success! Log saved.'\n")
	s, err := Start(root, card, ModeAll)
	if err != nil || !s.Active() || s.ID == "" || s.PID == 0 {
		t.Fatalf("slow live supervisor mislabeled: %+v, %v", s, err)
	}
	if !ReconcileWorker(s).Active() {
		t.Fatal("live supervisor treated as interrupted")
	}
	done := waitState(t, root, func(s State) bool { return s.Status == "completed" })
	if done.ID != s.ID {
		t.Fatal("lost run identity while reconnecting")
	}
}

func TestWorkerExitBeforeStatusIsFailure(t *testing.T) {
	root, card := fakeCard(t, "echo 'Success! Log saved.'\n")
	if err := os.MkdirAll(LogPath(root), 0700); err != nil {
		t.Fatal(err)
	}
	s, err := Start(root, card, ModeAll)
	if err == nil || s.Active() {
		t.Fatalf("dead supervisor treated as live: %+v, %v", s, err)
	}
}

func TestSlowLogFlushDoesNotBlockOutputState(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	defer reader.Close()
	o := &outputSink{log: writer, logPending: make([]byte, 1<<20)}
	flushed := make(chan struct{})
	go func() { o.flushLog(); close(flushed) }()
	time.Sleep(20 * time.Millisecond)
	written := make(chan struct{})
	go func() { o.Write([]byte("latest progress\n")); close(written) }()
	select {
	case <-written:
	case <-time.After(time.Second):
		t.Fatal("blocked log also blocked output parser")
	}
	o.mu.Lock()
	line := o.s.Lines[len(o.s.Lines)-1]
	o.mu.Unlock()
	if line != "latest progress" {
		t.Fatal("live output not updated during card stall")
	}
	reader.Close()
	select {
	case <-flushed:
	case <-time.After(time.Second):
		t.Fatal("flush did not finish after pipe closed")
	}
}

// ModeApp runs Downloader's launcher for the misterzine database alone,
// under the same supervisor: the run is named for what it does, and a
// card without the launcher refuses to start rather than fail later.
func TestAppModeRunsDownloaderForMisterzineOnly(t *testing.T) {
	root, card := fakeCard(t, "echo 'Running Update All'\n")
	if CanUpdateApp(card) {
		t.Fatal("a card with no Downloader claims it can update the app")
	}
	if _, err := Start(root, card, ModeApp); err == nil || !strings.Contains(err.Error(), "Downloader is not installed") {
		t.Fatalf("started without Downloader: %v", err)
	}
	// Downloader on its own prints no success line: its summary ends with
	// the version and the installed files, and the exit code carries the
	// result. The fake says exactly that much.
	downloader := filepath.Join(card, "Scripts", "downloader.sh")
	finishes := "#!/bin/bash\necho \"Running MiSTer Downloader args: $*\"\necho 'Downloading misterzine/misterzine'\necho 'Downloader 2.7 (abc) by theypsilon. Run time: 3s'\necho 'Installed:'\necho ' •misterzine/misterzine'\n"
	if err := os.WriteFile(downloader, []byte(finishes), 0700); err != nil {
		t.Fatal(err)
	}
	s, err := Start(root, card, ModeApp)
	if err != nil {
		t.Fatal(err)
	}
	// a run this short can already have finished by the time Start returns,
	// so only what does not change with it is asserted here
	if s.Mode != ModeApp || s.Name() != "MisterZine update" || s.ID == "" {
		t.Fatalf("started as %+v", s)
	}
	s = waitState(t, root, func(s State) bool { return !s.Active() })
	if s.Status != "completed" || s.Mode != ModeApp || !strings.Contains(strings.Join(s.Lines, "\n"), "args: --run-only misterzine") {
		t.Fatalf("%+v", s)
	}
	if s.Summary() != "MisterZine update finished successfully" {
		t.Fatalf("summary %q", s.Summary())
	}
	// a Downloader that fails still fails, exit code and all
	if err := os.WriteFile(downloader, []byte("#!/bin/bash\necho 'Running MiSTer Downloader'\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(root, card, ModeApp); err != nil {
		t.Fatal(err)
	}
	if s = waitState(t, root, func(s State) bool { return !s.Active() }); s.Status != "failed" {
		t.Fatalf("a failing Downloader reported %+v", s)
	}
}

// Update All does not install Scripts/downloader.sh; it keeps Downloader
// under Scripts/.config/downloader, which is where that launcher runs it
// from anyway. An ordinary Update All card has only the second, so the run
// goes straight to that binary, with the environment the launcher exports.
func TestAppModeFallsBackToUpdateAllsDownloader(t *testing.T) {
	root, card := fakeCard(t, "echo 'Running Update All'\n")
	config := filepath.Join(card, "Scripts", ".config", "downloader")
	if err := os.MkdirAll(config, 0700); err != nil {
		t.Fatal(err)
	}
	report := "#!/bin/bash\necho \"args: $*\"\necho \"ini: $DOWNLOADER_LAUNCHER_PATH\"\necho \"certs: $SSL_CERT_FILE\"\necho 'Downloader 2.7 (abc) by theypsilon. Run time: 3s'\n"
	if err := os.WriteFile(filepath.Join(config, "downloader_bin"), []byte(report), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "cacert.pem"), []byte("-----BEGIN CERTIFICATE-----\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if !CanUpdateApp(card) {
		t.Fatal("a card with Update All's Downloader cannot update the app")
	}
	if _, err := Start(root, card, ModeApp); err != nil {
		t.Fatal(err)
	}
	s := waitState(t, root, func(s State) bool { return !s.Active() })
	out := strings.Join(s.Lines, "\n")
	if s.Status != "completed" {
		t.Fatalf("%+v", s)
	}
	for _, want := range []string{
		"args: --run-only misterzine",
		"ini: " + filepath.Join(card, "Scripts", "downloader.sh"),
		"certs: " + filepath.Join(config, "cacert.pem"),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// the card's own launcher wins when it is there, since it also fixes
	// the clock and the certificates before running Downloader
	script := filepath.Join(card, "Scripts", "downloader.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho 'launcher ran'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	e, err := engineFor(card, ModeApp)
	if err != nil || e.path != "/bin/bash" || e.args[0] != script {
		t.Fatalf("engine %+v, %v", e, err)
	}
}
