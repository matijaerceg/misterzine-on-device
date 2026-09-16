"""Render the fixed UI scenarios used by CI and visual review."""
import os
from pathlib import Path
import subprocess

os.chdir(Path(__file__).resolve().parents[1])
go = os.environ.get("GO", "go")
scenarios = [
    ["-update-state", "testdata/update-completed.json", "-update-restart", "-out", "out/update-restart-h", "-script", "shot restart"],
    ["-update-state", "testdata/update-completed.json", "-update-restart", "-rot", "left", "-logical", "-out", "out/update-restart-t", "-script", "shot restart"],
    ["-update-state", "testdata/update-completed.json", "-out", "out/update-completed-h", "-script", "shot completed"],
    ["-update-state", "testdata/update-completed.json", "-rot", "left", "-logical", "-out", "out/update-completed-t", "-script", "shot completed"],
    ["-data", "testdata/rom-warning.json", "-rom-issue", "Missing game ROM: jpark.zip", "-out", "out/missing-rom-h", "-script",
     "enter; shot details; start; shot blocked"],
    ["-data", "testdata/rom-warning.json", "-rom-issue", "Missing game ROM: jpark.zip", "-rot", "left", "-logical", "-out", "out/missing-rom-t", "-script",
     "enter; shot details; start; shot blocked"],
    ["-out", "out/saver-dim-h", "-script",
     "back; home; pagedown*2; right; down*3; enter; down*2; right*2; down; shot brightness-33; enter; wait 1500; shot preview-33; back; right; shot brightness-66; enter; wait 1500; shot preview-66"],
    ["-rot", "left", "-logical", "-out", "out/saver-dim-t", "-script",
     "back; home; pagedown*2; right; down*3; enter; down*2; right*2; down; shot brightness-33; enter; wait 1500; shot preview-33; back; right; shot brightness-66; enter; wait 1500; shot preview-66"],
    ["-support-report", "testdata/support-mapped-start.json", "-out", "out/support-mapped-h", "-script",
     "back; end; up*2; enter; down*3; enter; shot result; right; shot mapping"],
    ["-support-report", "testdata/support-mapped-start.json", "-inset", "40", "-rot", "left", "-logical", "-out", "out/support-mapped-t", "-script",
     "back; end; up*2; enter; down*3; enter; shot result; right; shot mapping"],
    ["-rot", "none", "-out", "out/h"],
    ["-rot", "left", "-out", "out/t", "-logical"],
    ["-out", "out/filters-h", "-script",
     "space*3; home; shot alphabetical; tab; pagedown*4; right; shot arcade; end; right; shot controls-players"],
    ["-rot", "left", "-logical", "-out", "out/filters-t", "-script",
     "space*3; home; shot alphabetical; tab; pagedown*4; right; shot arcade; end; right; shot controls-players"],
    ["-update-state", "testdata/update-running.json", "-out", "out/update-h",
     "-script", "shot running; press back; wait 1000; shot cancel-hold; release back; hold back 2200; shot cancel"],
    ["-update-state", "testdata/update-running.json", "-rot", "left", "-logical",
     "-out", "out/update-t", "-script", "shot running; press back; wait 1000; shot cancel-hold; release back; hold back 2200; shot cancel"],
    ["-out", "out/search-h", "-script", "type 1943; shot matches; type xyz; shot empty"],
    ["-rot", "left", "-logical", "-out", "out/search-t",
     "-script", "type 1943; shot matches; type xyz; shot empty"],
    ["-out", "out/saver-h", "-script",
     "back; home; pagedown*2; right; down*3; enter; down; shot option; enter; wait 12000; shot preview; back; wait 300; back; home; up; shot wrap; down; shot top"],
    ["-rot", "left", "-logical", "-out", "out/saver-t", "-script",
     "back; home; pagedown*2; right; down*3; enter; down; shot option; enter; wait 12000; shot preview; back; wait 300; back; home; up; shot wrap; down; shot top"],
    # Options -> Screensaver style and the brightness row it brings; the preview
    # with no pictures at all falls back to the lettering
    ["-out", "out/saver-shots-h", "-script",
     "back; home; pagedown*2; right; down*3; enter; down*2; shot style; right; shot style-shots; down; shot bright; right; shot bright-full; enter; wait 1500; shot preview; back; wait 300; left; up; left; shot style-word"],
    ["-rot", "left", "-logical", "-out", "out/saver-shots-t", "-script",
     "back; home; pagedown*2; right; down*3; enter; down*2; shot style; right; shot style-shots; down; shot bright; right; shot bright-full; enter; wait 1500; shot preview; back; wait 300; left; up; left; shot style-word"],
    ["-support-report", "testdata/support-controller.json", "-out", "out/support-h", "-script",
     "back; end; up*2; enter; shot menu; enter; shot ready; wait 3500; shot capture; wait 6000; shot result; right; shot evidence; back; down*2; enter; shot launch; enter; shot launch-result"],
    ["-support-report", "testdata/support-controller.json", "-rot", "left", "-logical", "-out", "out/support-t", "-script",
     "back; end; up*2; enter; shot menu; enter; shot ready; wait 3500; shot capture; wait 6000; shot result; right; shot evidence; back; down*2; enter; shot launch; enter; shot launch-result"],
    ["-support-report", "testdata/support-no-input.json", "-inset", "40", "-out", "out/support-small-h", "-script",
     "back; end; up*2; enter; down*3; enter; shot result; right; shot evidence"],
    ["-support-report", "testdata/support-no-input.json", "-inset", "40", "-rot", "left", "-logical", "-out", "out/support-small-t", "-script",
     "back; end; up*2; enter; down*3; enter; shot result; right; shot evidence"],
]
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-status", "missing",
                          "-out", f"out/browsing-{orientation}-{inset}", "-script",
                          "back; home; pagedown; right; down*6; shot remember-on; left; shot remember-off; back; "
                          "space*3; home; pagedown; shot letter-jump; enter; enter; start; shot launch-failure; "
                          "back; back; wait 3200; end; pageup; pagedown; shot last-letter; "
                          "space*3; home; pagedown; shot updated-month; wait 2200; shot updated-month-settled; "
                          "space; home; pagedown; shot debut-month; wait 2200; shot debut-month-settled; "
                          "space; home; shot year-top; pagedown; shot year-jump; wait 2200; shot year-settled"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-app-update", "v1.0.6",
                          "-out", f"out/new-modes-{orientation}-{inset}", "-script",
                          "shot update-notice; back; down*2; shot update-option; back; "
                          "enter; space; back; space*5; wait 2200; shot favorites; tab; shot counts"])
        scenarios.append([*rotation, "-inset", str(inset), "-scan-result",
                          "-out", f"out/card-scan-{orientation}-{inset}", "-script", "shot result"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-unchanged",
                          "-out", f"out/rotation-only-{orientation}-{inset}", "-script",
                          "shot unchanged; back; home; pagedown*2; right; down; shot follow-on; left; shot follow-off; "
                          "right; back; tab; pagedown*2; right; down; shot only-legend; space; shot only-selected; space; shot only-restored; space; back; shot filtered-unchanged"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    scenarios.append([*rotation, "-inset", "40",
                      "-out", f"out/rotation-filter-{orientation}", "-script",
                      "back; home; pagedown; right; down*4; shot off; right; shot on; back; shot list; "
                      "tab; shot filters; back; back; home; pagedown; right; down*4; left; back; shot restored"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset),
                          "-out", f"out/year-filter-{orientation}-{inset}", "-script",
                          "tab; pagedown*4; right; down*2; shot decades; right; shot expanded; "
                          "down*2; enter; shot mixed; space; shot only-year; "
                          "space; shot all-years; left; shot collapsed"])
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
                          "type gradius; shot beta-tall; back; back; home; pagedown; right; down*7; shot title-font; left; back; "
                          "type gradius; shot beta-narrow; back; back; home; pagedown; right; down*7; left; back; "
                          "type gradius; shot beta-normal; back; back; home; pagedown; right; down*7; right; right; "
                          "down; shot list-shots; right; back; shot title-shot; back; home; pagedown; right; down*8; left; "
                          "down; right; shot date-dd-mm; back; shot list-dd-mm; back; home; pagedown; right; down*9; right; shot date-mon-d; back; shot list-mon-d; "
                          "back; home; pagedown; right; down*9; right; shot date-d-mon; back; shot list-d-mon; back; home; pagedown; right; down*9; right; shot date-yymmdd; back; shot list-yymmdd; "
                          "back; home; pagedown; right; down*9; left*4; back; "
                          "type galaga; down*2; enter; right*2; shot alt-start; wait 1200; shot alt-mid; wait 1200; shot alt-end; "
                          "down; shot info-page; wait 400; shot info-sliding; wait 2000; shot info-paged; "
                          "back; tab; shot heading; enter; shot heading-open; enter; shot heading-closed; "
                          "pagedown*2; right; down; shot arcade-open; down; enter; shot beta-off; up; space; shot stable-only"])
scenarios.append(["-out", "out/list-polish-h-15", "-remember-alt", "2", "-script",
                  "type galaga; down*2; shot alt-remembered-list; enter; shot alt-remembered"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    # Options -> Sources: installed only, with the MiSTer and Jotego databases
    scenarios.append([*rotation, "-installed", "distribution_mister,jtcores",
                      "-out", f"out/installed-sources-{orientation}", "-script",
                      "shot list; tab; down*3; right; shot filters; back; back; home; pagedown; right; down*2; shot option; left; shot option-all; back; shot list-all"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    # the Recents view with nine launches: the view, a month jump, the
    # Views row and page with it on, and the fallback when it goes off
    # from the Views page while the list is in it
    scenarios.append([*rotation, "-recents", "9",
                      "-out", f"out/recents-{orientation}", "-script",
                      "space*6; shot recents; home; pagedown; shot month-jump; wait 2200; "
                      "back; home; pagedown; right; down*5; shot option; enter; end; shot views-on; enter; back; back; shot off"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        # Options -> Layout: split and picture, a vertical shot in each
        scenarios.append([*rotation, "-inset", str(inset), "-layout", "split",
                          "-out", f"out/list-layout-{orientation}-{inset}", "-script",
                          "down*2; shot split; type 1942; shot split-vertical; back; back; home; pagedown; right; down*10; shot option; right; back; "
                          "shot picture; type 1942; shot picture-vertical"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    # Options -> Button labels: PlayStation symbols on the list, Details and
    # Filters legends, the option row, then numbers on the list and a search
    # miss, and the Xbox set's swapped letters
    scenarios.append([*rotation, "-button-labels", "playstation",
                      "-out", f"out/button-labels-{orientation}", "-script",
                      "shot list; enter; shot details; back; tab; shot filters; back; "
                      "back; home; pagedown*3; right; down; shot option; right; shot option-numbers; back; shot list-numbers; "
                      "type xyz; shot empty-numbers; back; back; home; pagedown*3; right; down; left*3; shot option-mister; left; back; shot list-mister"])
    scenarios.append([*rotation, "-button-labels", "xbox",
                      "-out", f"out/button-labels-xbox-{orientation}", "-script", "shot list; enter; shot details"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    # Troubleshooting -> Test pad buttons: the pads as read, then scripted
    # presses (no pad name, so they show as script input), then hold B out
    scenarios.append([*rotation, "-support-report", "testdata/support-controller.json",
                      "-out", f"out/pad-test-{orientation}", "-script",
                      "back; end; up*2; enter; down; shot menu; enter; shot empty; enter; space; tab; start; shot presses; "
                      "press back; wait 1000; shot leave-hold; release back; hold back 2200; shot left"])
    # Options -> OK button with the defined pad (MENU OK on B) as the current
    # pad: auto, the A override, the B override, and back to auto
    scenarios.append([*rotation, "-support-report", "testdata/support-controller.json",
                      "-out", f"out/ok-button-{orientation}", "-script",
                      "back; home; pagedown*3; right; down*2; shot option; right; shot option-a; right; shot option-b; left*2; shot option-auto"])
for canvas in ("360x270", "400x300"):
    for rotation in ([], ["-rot", "left", "-logical"]):
        orientation = "t" if rotation else "h"
        # Options -> HDMI picture: the fit-display sizes for 1080p and 1600x900 or
        # 800x600 on every main screen and in the split and picture layouts
        scenarios.append([*rotation, "-canvas", canvas,
                          "-out", f"out/fit-{canvas}-{orientation}", "-script",
                          "shot list; enter; shot details; back; tab; shot filter; back; back; shot options; "
                          "home; pagedown; right; down*10; right; back; shot split; back; right; back; shot picture"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    # Select held on the list: the chord legend, Y cycling the layout, X the
    # list shot, then the ordinary legend and the sort once it is released
    scenarios.append([*rotation,
                      "-out", f"out/quick-{orientation}", "-script",
                      "press select; shot chord-legend; space; shot layout-split; space; shot layout-picture; "
                      "tab; shot shots-title; enter; shot favorite-added; release select; shot released; space; shot sorted"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    # the pad's Menu button: Options from the list, closed again, Options
    # the Maker order with its header lines, a maker jump, and the Views page
    scenarios.append([*rotation,
                      "-out", f"out/maker-{orientation}", "-script",
                      "space*4; shot maker; pagedown*3; shot maker-jump; down*2; shot maker-rows; home; down*48; shot maker-pinned; "
                      "back; home; pagedown; right; down*5; shot options-views; enter; shot views; down*4; enter; shot views-off; "
                      "up*4; enter; down; enter; down; enter; down; enter; down*2; enter; shot views-last; "
                      "enter; back; back"])
    # Options -> Credits: the row, the page from its top, its end and the way back
    scenarios.append([*rotation,
                      "-out", f"out/credits-{orientation}", "-script",
                      "back; end; up; shot options-credits; enter; shot credits; end; shot credits-end; back; shot options-back"])
    # the letter and year headers: a jump under its header, a group name
    # pinned on the top line while its rows run past it, the unknown years
    scenarios.append([*rotation,
                      "-out", f"out/headers-{orientation}", "-script",
                      "space*3; home; down*30; shot letter-pinned; pagedown*2; shot letter-jump; "
                      "space*5; home; down*30; shot year-pinned; end; up*2; shot year-unknown"])
    # over Details, the Menu button option row itself, closing Options back
    # onto Details and Filters, and the held-Menu hint over Filters (the
    # screen stays put once the hold engages) with its progress line
    scenarios.append([*rotation,
                      "-out", f"out/menu-{orientation}", "-script",
                      "menu; shot options; menu; shot list; enter; menu; shot options-from-details; "
                      "home; pagedown*3; right; down*3; shot option-row; right; shot option-leave; left; back; shot details-back; "
                      "back; tab; down*2; menu; shot options-over-filters; back; shot filters-back; "
                      "press menu; wait 700; shot menu-hint; wait 900; shot menu-hold; release menu; shot menu-released; back; back"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    scenarios.append([*rotation, "-out", f"out/layout-previews-{orientation}", "-script",
                      "back; home; pagedown; right; down*10; shot list; right; shot split; right; shot picture; down; shot dismissed"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-out", f"out/option-samples-{orientation}-{inset}", "-script",
                          "back; home; pagedown; right; down*7; shot font; left; shot font-narrow; left; shot font-normal; "
                          "home; pagedown*3; right; down*4; shot speed; wait 200; shot speed-moving; down; shot delay; "
                          "wait 250; shot delay-short; wait 400; shot delay-reverse; down; shot dismissed"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-out", f"out/default-view-{orientation}-{inset}", "-script",
                          "back; home; pagedown; right; down*6; shot remember-on; left; shot remember-off; down; shot default; right*3; shot alphabetical; up; right; shot hidden"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    scenarios.append([*rotation, "-show-non-arcade", "-out", f"out/deprecated-{orientation}", "-script",
                      "back; home; pagedown; right; down*3; shot hidden; right; shot shown; back; type genesis; shot found"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-out", f"out/transitions-{orientation}-{inset}", "-script",
                          "back; home; pagedown*2; right; down*6; shot on; left; shot off"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-arcade-intro",
                          "-out", f"out/arcade-first-{orientation}-{inset}", "-script",
                          "shot intro; press enter; wait 1000; shot intro-hold; release enter; shot intro-cancelled; hold enter 2000; shot arcade; back; home; pagedown; right; down; shot preference-off; right; shot preference-on; "
                          "back; shot mixed; tab; pagedown*2; right; shot mixed-types; back; back; left; back; "
                          "tab; pagedown*2; right; shot arcade-types; down; space; shot stable-only; space; shot restored; "
                          "home; shot clear; back; shot arcade-restored"])
for rotation in ([], ["-rot", "left", "-logical"]):
    orientation = "t" if rotation else "h"
    for inset in (15, 40):
        scenarios.append([*rotation, "-inset", str(inset), "-out", f"out/options-sections-{orientation}-{inset}", "-script",
                          "back; shot default; enter; shot collapsed; pagedown; shot list-closed; right; shot list-open; "
                          "pagedown; enter; shot display-open; left; shot display-closed; back; back; shot remembered; "
                          "end; shot bottom; up*2; enter; shot troubleshooting; back; shot returned"])
for rotation in ([], ["-rot", "left", "-logical"], ["-rot", "right", "-logical"]):
    orientation = rotation[1] if rotation else "horizontal"
    for layout in ("list", "split", "picture"):
        scenarios.append([*rotation, "-canvas", "480x270", "-layout", layout,
                          "-out", f"out/full-display-{orientation}-{layout}", "-script",
                          "shot list; enter; shot details; back; tab; shot filters; back; back; shot options; home; pagedown*2; right; down*5; right*2; shot full-option; home; pagedown*2; right; down*3; enter; down; enter; wait 12000; shot saver"])
for args in scenarios:
    subprocess.run([go, "run", "./cmd/mzharness", "-images", "", *args], check=True)
