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
	if len(os.Args) == 5 && os.Args[1] == "update-worker" {
		os.Exit(Worker(os.Args[2], os.Args[3], os.Args[4]))
	}
	if len(os.Args) == 4 && os.Args[1] == "test-start" {
		s, err := Start(os.Args[2], os.Args[3])
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
	s, err := Start(root, card)
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
	s, err := Start(root, card)
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, root, func(s State) bool { return len(s.Lines) > 0 })
	other, err := Start(root, card)
	if err != nil || other.ID != s.ID {
		t.Fatalf("duplicate run: %v %+v", err, other)
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
	root, card := fakeCard(t, "echo 'Linux will be updated from distribution_mister:'\nsleep 2\necho 'Linux has been updated!'\nsleep 8\necho 'Success! Log saved.'\n")
	s, err := Start(root, card)
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
	if _, err := Start(root, card); err != nil {
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
	s, err := Start(root, card)
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
			s, err := Start(root, card)
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
	if _, err := Start(root, card); err != nil {
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
