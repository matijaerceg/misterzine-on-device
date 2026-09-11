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
     "space*3; home; shot alphabetical; tab; pagedown*4; right; shot arcade; end; right; shot controls-players"],
    ["-rot", "left", "-logical", "-out", "out/filters-t", "-script",
     "space*3; home; shot alphabetical; tab; pagedown*4; right; shot arcade; end; right; shot controls-players"],
    ["-update-state", "testdata/update-running.json", "-out", "out/update-h",
     "-script", "shot running; hold back 2200; shot cancel"],
    ["-update-state", "testdata/update-running.json", "-rot", "left", "-logical",
     "-out", "out/update-t", "-script", "shot running; hold back 2200; shot cancel"],
    ["-out", "out/search-h", "-script", "type 1943; shot matches; type xyz; shot empty"],
    ["-rot", "left", "-logical", "-out", "out/search-t",
     "-script", "type 1943; shot matches; type xyz; shot empty"],
    ["-out", "out/saver-h", "-script",
     "back; down*12; shot option; enter; wait 12000; shot preview; back; up*12; up; shot wrap; down; shot top"],
    ["-rot", "left", "-logical", "-out", "out/saver-t", "-script",
     "back; down*12; shot option; enter; wait 12000; shot preview; back; up*12; up; shot wrap; down; shot top"],
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
                          "back; down*16; shot remember-on; left; shot remember-off; back; "
                          "space*3; home; pagedown; shot letter-jump; enter; enter; start; shot launch-failure; "
                          "back; back; wait 3200; end; pageup; pagedown; shot last-letter; "
                          "space*2; home; pagedown; shot updated-month; wait 2200; shot updated-month-settled; "
                          "space; home; pagedown; shot debut-month; wait 2200; shot debut-month-settled; "
                          "space; home; shot year-top; pagedown; shot year-jump; wait 2200; shot year-settled"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-app-update", "v1.0.6",
                          "-out", f"out/new-modes-{orientation}-{inset}", "-script",
                          "shot update-notice; back; down; shot update-option; back; "
                          "enter; space; back; space*4; wait 2200; shot favorites; tab; shot counts"])
        scenarios.append([*rotation, "-inset", str(inset), "-scan-result",
                          "-out", f"out/card-scan-{orientation}-{inset}", "-script", "shot result"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-unchanged",
                          "-out", f"out/rotation-only-{orientation}-{inset}", "-script",
                          "shot unchanged; back; down*6; shot follow-on; left; shot follow-off; "
                          "right; back; tab; pagedown*2; right; down; shot only-legend; space; shot only-selected; space; shot only-restored; space; back; shot filtered-unchanged"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    scenarios.append([*rotation, "-inset", "40",
                      "-out", f"out/rotation-filter-{orientation}", "-script",
                      "back; down*8; shot off; right; shot on; back; shot list; "
                      "tab; shot filters; back; back; down*8; left; back; shot restored"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset),
                          "-out", f"out/year-filter-{orientation}-{inset}", "-script",
                          "tab; pagedown*4; right; down*2; shot decades; tab; shot expanded; "
                          "down*2; enter; shot mixed; space; shot only-year; "
                          "space; shot all-years; tab; shot collapsed"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    scenarios.append([*rotation, "-inset", "40",
                      "-out", f"out/center-scroll-{orientation}", "-script",
                      "space*3; home; down*12; shot middle; end; up*4; shot bottom; "
                      "home; shot top; tab; pagedown*4; right; shot section; left; shot closed; "
                      "right; down*2; right; shot years; left; left; shot nested-closed; "
                      "pagedown; shot next-section; pageup; shot previous-section"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        # narrow titles with a beta sign, the normal font, list shots and
        # date formats in Options, a scrolling alternative name, and A on
        # a Filters heading
        scenarios.append([*rotation, "-inset", str(inset),
                          "-out", f"out/list-polish-{orientation}-{inset}", "-script",
                          "type gradius; shot beta-tall; back; back; down*9; shot title-font; left; back; "
                          "type gradius; shot beta-narrow; back; back; down*9; left; back; "
                          "type gradius; shot beta-normal; back; back; down*9; right; right; "
                          "down; shot list-shots; right; back; shot title-shot; back; down*10; left; "
                          "down; right; shot date-dd-mm; back; shot list-dd-mm; back; down*11; right; shot date-mon-d; back; shot list-mon-d; "
                          "back; down*11; right; shot date-d-mon; back; shot list-d-mon; back; down*11; right; shot date-yymmdd; back; shot list-yymmdd; "
                          "back; down*11; left*4; back; "
                          "type galaga; down*2; enter; right*2; shot alt-start; wait 1200; shot alt-mid; wait 1200; shot alt-end; "
                          "down; shot info-page; wait 400; shot info-sliding; wait 2000; shot info-paged; "
                          "back; tab; shot heading; enter; shot heading-open; enter; shot heading-closed; "
                          "pagedown*2; right; down; right; shot arcade-open; down*2; enter; shot beta-off; up; space; shot stable-only"])
for args in scenarios:
    subprocess.run([go, "run", "./cmd/mzharness", "-images", "", *args], check=True)
