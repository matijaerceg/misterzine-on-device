"""Exact deterministic render regression gate; --write deliberately updates baseline."""
import argparse
import hashlib
import json
from pathlib import Path
p=argparse.ArgumentParser()
p.add_argument("root",type=Path)
p.add_argument("manifest",type=Path)
p.add_argument("--write",action="store_true")
a=p.parse_args()
actual={str(f.relative_to(a.root)).replace("\\","/"):hashlib.sha256(f.read_bytes()).hexdigest() for f in sorted(a.root.rglob("*.png"))}
if not actual: raise SystemExit("No renders found")
if a.write:
    a.manifest.write_text(json.dumps(actual,indent=2)+"\n")
else:
    expected=json.loads(a.manifest.read_text())
    changed=[k for k in sorted(actual.keys() | expected.keys()) if actual.get(k)!=expected.get(k)]
    if changed: raise SystemExit("Render regression (inspect before updating golden): " + ", ".join(changed))
    print(f"{len(actual)} golden renders match")
