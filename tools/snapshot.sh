#!/bin/bash
# Refresh the embedded data snapshot and the test fixture from the live site.
set -e
cd "$(dirname "$0")/.."
curl -sfL -o testdata/data.json https://misterzine.fyi/releases/data.json
curl -sfL -o testdata/meta.json https://misterzine.fyi/releases/meta.json
python -c "import gzip; gzip.open('internal/snapshot/data.json.gz','wb',compresslevel=9).write(open('testdata/data.json','rb').read())"
cp testdata/meta.json internal/snapshot/meta.json
echo "snapshot refreshed: $(cat testdata/meta.json)"
