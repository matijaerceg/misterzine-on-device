"""Render the fixed UI scenarios used by CI and visual review."""
import os
from pathlib import Path
import subprocess

os.chdir(Path(__file__).resolve().parents[1])
go = os.environ.get("GO", "go")
scenarios = [
    ["-support-report", "testdata/support-mapped-start.json", "-out", "out/support-mapped-h", "-script",
     "back; end; up; enter; down*2; enter; shot result; right; shot mapping"],
    ["-support-report", "testdata/support-mapped-start.json", "-inset", "40", "-rot", "left", "-logical", "-out", "out/support-mapped-t", "-script",
     "back; end; up; enter; down*2; enter; shot result; right; shot mapping"],
    ["-rot", "none", "-out", "out/h"],
    ["-rot", "left", "-out", "out/t", "-logical"],
    ["-out", "out/filters-h", "-script",
     "space*2; home; shot alphabetical; tab; down*30; shot arcade; end; shot controls-players"],
    ["-rot", "left", "-logical", "-out", "out/filters-t", "-script",
     "space*2; home; shot alphabetical; tab; down*30; shot arcade; end; shot controls-players"],
    ["-update-state", "testdata/update-running.json", "-out", "out/update-h",
     "-script", "shot running; hold back 2200; shot cancel"],
    ["-update-state", "testdata/update-running.json", "-rot", "left", "-logical",
     "-out", "out/update-t", "-script", "shot running; hold back 2200; shot cancel"],
    ["-out", "out/search-h", "-script", "type 1943; shot matches; type xyz; shot empty"],
    ["-rot", "left", "-logical", "-out", "out/search-t",
     "-script", "type 1943; shot matches; type xyz; shot empty"],
    ["-out", "out/saver-h", "-script",
     "back; down*9; shot option; enter; wait 12000; shot preview; back; up*9; up; shot wrap; down; shot top"],
    ["-rot", "left", "-logical", "-out", "out/saver-t", "-script",
     "back; down*9; shot option; enter; wait 12000; shot preview; back; up*9; up; shot wrap; down; shot top"],
    ["-support-report", "testdata/support-controller.json", "-out", "out/support-h", "-script",
     "back; end; up; enter; shot menu; enter; shot ready; wait 3500; shot capture; wait 6000; shot result; right; shot evidence; back; down; enter; shot launch; enter; shot launch-result"],
    ["-support-report", "testdata/support-controller.json", "-rot", "left", "-logical", "-out", "out/support-t", "-script",
     "back; end; up; enter; shot menu; enter; shot ready; wait 3500; shot capture; wait 6000; shot result; right; shot evidence; back; down; enter; shot launch; enter; shot launch-result"],
    ["-support-report", "testdata/support-no-input.json", "-inset", "40", "-out", "out/support-small-h", "-script",
     "back; end; up; enter; down*2; enter; shot result; right; shot evidence"],
    ["-support-report", "testdata/support-no-input.json", "-inset", "40", "-rot", "left", "-logical", "-out", "out/support-small-t", "-script",
     "back; end; up; enter; down*2; enter; shot result; right; shot evidence"],
]
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-status", "missing",
                          "-out", f"out/browsing-{orientation}-{inset}", "-script",
                          "back; down*8; shot remember-on; left; shot remember-off; back; "
                          "space*2; home; pagedown; shot letter-jump; enter; enter; start; shot launch-failure; "
                          "back; back; wait 3200; end; pageup; pagedown; shot last-letter; "
                          "space*2; home; pagedown; shot updated-month; wait 2200; shot updated-month-settled; "
                          "space; home; pagedown; shot debut-month; wait 2200; shot debut-month-settled"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-app-update", "v1.0.6",
                          "-out", f"out/new-modes-{orientation}-{inset}", "-script",
                          "shot update-notice; back; down; shot update-option; back; "
                          "enter; space; back; space*3; wait 2200; shot favorites; tab; shot counts"])
        scenarios.append([*rotation, "-inset", str(inset), "-scan-result",
                          "-out", f"out/card-scan-{orientation}-{inset}", "-script", "shot result"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-unchanged",
                          "-out", f"out/rotation-only-{orientation}-{inset}", "-script",
                          "shot unchanged; back; down*5; shot follow-on; left; shot follow-off; "
                          "right; back; tab; down*14; shot only-legend; space; shot only-selected; space; shot only-restored; space; back; shot filtered-unchanged"])
for args in scenarios:
    subprocess.run([go, "run", "./cmd/mzharness", "-images", "", *args], check=True)
