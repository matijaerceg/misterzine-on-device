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
     "back; down*7; shot option; enter; wait 12000; shot preview; back; up*7; up; shot wrap; down; shot top"],
    ["-rot", "left", "-logical", "-out", "out/saver-t", "-script",
     "back; down*7; shot option; enter; wait 12000; shot preview; back; up*7; up; shot wrap; down; shot top"],
    ["-support-report", "testdata/support-controller.json", "-out", "out/support-h", "-script",
     "back; end; up; enter; shot menu; enter; shot ready; wait 3500; shot capture; wait 6000; shot result; right; shot evidence; back; down; enter; shot launch; enter; shot launch-result"],
    ["-support-report", "testdata/support-controller.json", "-rot", "left", "-logical", "-out", "out/support-t", "-script",
     "back; end; up; enter; shot menu; enter; shot ready; wait 3500; shot capture; wait 6000; shot result; right; shot evidence; back; down; enter; shot launch; enter; shot launch-result"],
    ["-support-report", "testdata/support-no-input.json", "-inset", "40", "-out", "out/support-small-h", "-script",
     "back; end; up; enter; down*2; enter; shot result; right; shot evidence"],
    ["-support-report", "testdata/support-no-input.json", "-inset", "40", "-rot", "left", "-logical", "-out", "out/support-small-t", "-script",
     "back; end; up; enter; down*2; enter; shot result; right; shot evidence"],
]
for args in scenarios:
    subprocess.run([go, "run", "./cmd/mzharness", "-images", "", *args], check=True)
