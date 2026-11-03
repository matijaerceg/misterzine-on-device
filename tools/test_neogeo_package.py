"""Check the private package and MiSTer Main's unquoted Scripts invocation."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
import zipfile


class NeoGeoPackageTest(unittest.TestCase):
    def test_clean_package_and_main_script_path(self):
        repo = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory(prefix="mz-neogeo-") as tmp:
            root = Path(tmp)
            binary = root / "binary"
            binary.write_bytes(b"test binary")
            archive = root / "test.zip"
            subprocess.run([sys.executable, str(repo / "tools/package_neogeo.py"),
                            "--binary", str(binary), "--probe", str(binary),
                            "--output", str(archive)], check=True, capture_output=True)
            with zipfile.ZipFile(archive) as z:
                self.assertIsNone(z.testzip())
                scripts = [x for x in z.namelist() if x.startswith("Scripts/")]
                self.assertEqual(scripts, ["Scripts/MisterZine-NeoGeo-TEST.sh"])
                self.assertFalse(any(x.endswith(("settings.json", "debug.flag", "debug-token")) for x in z.namelist()))
                wrapper = z.read(scripts[0])
                self.assertNotIn(b"\r", wrapper)
                self.assertEqual(wrapper, z.read("misterzine-neogeo-test/launch.sh"))
                self.assertTrue((z.getinfo(scripts[0]).external_attr >> 16) & 0o111)
                z.extractall(root / "card")
            if os.name != "posix":
                return
            entry = root / "card" / scripts[0]
            # Main writes both dirname and execution without shell quotes.
            # Run that exact calling convention with a stand-in payload.
            entry.write_text("#!/bin/sh\nprintf 'script-started\\n'\n")
            entry.chmod(0o755)
            result = subprocess.run(["bash", "-c", f"set -e\ncd $(dirname {entry})\n{entry}\n"], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, "script-started\n")


if __name__ == "__main__":
    unittest.main()
