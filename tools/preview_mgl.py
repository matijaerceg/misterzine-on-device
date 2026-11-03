"""Run the isolated MGL importer and render its five-game example in the UI.

No card/device writes or real game launches. Requires Go and Python 3.
"""
import json
import os
from pathlib import Path
import subprocess


def main():
    root = Path(__file__).resolve().parents[1]
    os.chdir(root)
    out = root / "dist" / "mgl-prototype"
    out.mkdir(parents=True, exist_ok=True)
    go = os.environ.get("GO", "go")
    probe = [go, "run", "./cmd/mzmglprobe", "-card", "testdata/mgl/card",
             "-arcade-list", "testdata/mgl/arcade-pilot.json"]
    report = json.loads(subprocess.check_output(probe, text=True))
    (out / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    rows = out / "rows.json"
    rows.write_text(json.dumps(report["rows"], indent=2) + "\n", encoding="utf-8")
    counts = {}
    for entry in report["entries"]:
        decision = entry["decision"]
        counts[decision] = counts.get(decision, 0) + 1
    print("MGL import:", counts, flush=True)
    for orientation in ("horizontal", "vertical"):
        command = [go, "run", "./cmd/mzharness", "-data", str(rows),
                   "-status", "unknown", "-out", str(out / orientation),
                   "-script", "shot list; enter; shot details; back; start"]
        if orientation == "vertical":
            command += ["-rot", "left", "-logical"]
        subprocess.run(command, check=True)
    print("Preview and report:", out)


if __name__ == "__main__":
    main()
