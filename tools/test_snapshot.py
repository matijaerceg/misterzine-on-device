"""Exercise snapshot refresh with local download fixtures, never the live site."""
import gzip
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

class SnapshotTest(unittest.TestCase):
    def test_refresh_preserves_golden_fixtures_and_rejects_empty_data(self):
        with tempfile.TemporaryDirectory(prefix="mz-snapshot-") as tmp:
            root = Path(tmp)
            for name in ("tools", "testdata", "internal/snapshot", "bin", "downloads"):
                (root / name).mkdir(parents=True)
            shutil.copyfile(Path(__file__).with_name("snapshot.sh"), root / "tools/snapshot.sh")
            fixture = root / "testdata/data.json"
            fixture.write_text("untouched golden fixture")
            curl = root / "bin/curl"
            curl.write_text("#!/usr/bin/env python3\nimport os,sys,shutil\nfrom pathlib import Path\nshutil.copyfile(Path(os.environ['MOCK_DOWNLOADS'])/sys.argv[-1].split('/')[-1],sys.argv[sys.argv.index('-o')+1])\n")
            curl.chmod(0o700)
            raw = b'[{"k":"new"}]'
            (root / "downloads/data.json").write_bytes(raw)
            (root / "downloads/meta.json").write_text('{"hash":"fixture"}')
            env = dict(os.environ, PATH=str(root / "bin") + os.pathsep + os.environ["PATH"], MOCK_DOWNLOADS=str(root / "downloads"))
            subprocess.run(["bash", str(root / "tools/snapshot.sh")], env=env, check=True, capture_output=True)
            snapshot = root / "internal/snapshot/data.json.gz"
            before = snapshot.read_bytes()
            self.assertEqual(gzip.decompress(before), raw)
            self.assertEqual(fixture.read_text(), "untouched golden fixture")
            (root / "downloads/data.json").write_text("[]")
            result = subprocess.run(["bash", str(root / "tools/snapshot.sh")], env=env, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(snapshot.read_bytes(), before)
            self.assertEqual(fixture.read_text(), "untouched golden fixture")

if __name__ == "__main__":
    unittest.main()
