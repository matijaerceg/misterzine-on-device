//go:build linux

package updater

import (
	"fmt"
	"strings"
	"testing"
)

func TestSafetyWarningsAcrossOutputBoundary(t *testing.T) {
	for _, tc := range []struct {
		marker string
		start  State
		check  func(State) bool
	}{
		{"Linux will be updated", State{}, func(s State) bool { return s.Protected && s.Reboot }},
		{"installing analogue pocket firmware", State{}, func(s State) bool { return s.Protected }},
		{"Linux has been updated", State{Protected: true}, func(s State) bool { return !s.Protected && s.Reboot }},
		{"There were some errors in the Updaters", State{}, func(s State) bool { return s.HadErrors }},
	} {
		for split := 1; split < len(tc.marker); split++ {
			t.Run(fmt.Sprintf("%s/%d", tc.marker, split), func(t *testing.T) {
				o := &outputSink{s: tc.start}
				o.Write([]byte(strings.Repeat("x", 4096-split) + tc.marker[:split]))
				o.snapshot() // a heartbeat between pipe reads must keep the prefix
				o.Write([]byte(tc.marker[split:]))
				if got := o.snapshot(); !tc.check(got) {
					t.Fatalf("missed split warning: protected=%v reboot=%v errors=%v", got.Protected, got.Reboot, got.HadErrors)
				}
				o.Write([]byte("\n"))
				if got := o.snapshot(); !tc.check(got) {
					t.Fatal("finishing the line lost the warning")
				}
			})
		}
	}
}

func TestOutputWarningOrderAndTerminalEscapes(t *testing.T) {
	o := &outputSink{}
	write := func(s string) {
		for _, c := range []byte(s) {
			o.Write([]byte{c})
		}
	}
	// OSC window-title contents must not be mistaken for updater output.
	write("\x1b]0;Linux will be updated\x1b\\")
	if o.snapshot().Protected {
		t.Fatal("terminal title was treated as a warning")
	}
	write(strings.Repeat("x", 600) + "Linux will be up\x1b[32mdated\x1b[0m")
	if got := o.snapshot(); !got.Protected || !got.Reboot {
		t.Fatal("long, split, coloured warning was missed")
	}
	write(strings.Repeat("x", 600) + "Linux has been updated")
	for i := 0; i < 3; i++ {
		if o.snapshot().Protected {
			t.Fatal("an old start warning overrode the later completion")
		}
	}
	write("\n")
	if o.snapshot().Protected {
		t.Fatal("logging the finished line replayed the old start warning")
	}
	write("Linux will be up\ndated\rLinux will be up\rdated\n")
	if o.snapshot().Protected {
		t.Fatal("warning fragments crossed a real line boundary")
	}
	for _, line := range o.snapshot().Lines {
		if len([]rune(line)) > 512 {
			t.Fatal("display line limit was lost")
		}
	}
}

func TestOutputSuccessNeedsRealLineStart(t *testing.T) {
	o := &outputSink{}
	// A display-buffer flush is not a new logical line. Success quoted in
	// the middle of a long line must not turn a failed updater into success.
	o.Write([]byte(strings.Repeat("x", 4096) + "Success! Log saved.\n"))
	if o.snapshot().SawSuccess {
		t.Fatal("a display flush manufactured a success announcement")
	}
	o.Write([]byte("   \x1b[32mSuccess! More details at: log.txt\x1b[0m\r"))
	if !o.snapshot().SawSuccess {
		t.Fatal("a real, indented success line was missed")
	}
}
