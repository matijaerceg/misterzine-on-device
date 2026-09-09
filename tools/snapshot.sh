#!/bin/bash
# Refresh only the embedded snapshot. sort_golden.js owns the coupled test fixtures.
set -euo pipefail
cd "$(dirname "$0")/.."
staging=$(mktemp -d)
trap 'rm -f "$staging/data.json" "$staging/meta.json" "$staging/data.json.gz"; rmdir "$staging"' EXIT
curl -sfL -o "$staging/data.json" https://misterzine.fyi/releases/data.json
curl -sfL -o "$staging/meta.json" https://misterzine.fyi/releases/meta.json
python3 - "$staging" <<'PY'
import gzip, json, pathlib, sys
stage = pathlib.Path(sys.argv[1])
raw = (stage / "data.json").read_bytes()
rows = json.loads(raw)
json.loads((stage / "meta.json").read_bytes())
if not isinstance(rows, list) or not rows:
    raise SystemExit("snapshot: empty or invalid catalogue")
(stage / "data.json.gz").write_bytes(gzip.compress(raw, compresslevel=9, mtime=0))
PY
cp "$staging/data.json.gz" internal/snapshot/data.json.gz
cp "$staging/meta.json" internal/snapshot/meta.json
echo "snapshot refreshed: $(cat "$staging/meta.json")"
