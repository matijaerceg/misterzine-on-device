"""Exercise the pinned installer with a synthetic card and local download."""
import hashlib
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
import zipfile


@unittest.skipUnless(os.name == "posix", "requires Linux shell tools")
class InstallerTest(unittest.TestCase):
    def test_install_repair_and_rejected_download(self):
        source = (Path(__file__).resolve().parents[1] / "deploy/MisterZine-NeoGeo-Install-test.3-r2.sh").read_text()
        self.assertIn("--proto '=https'", source)
        self.assertIn("/releases/download/v1.2.24-neogeo-test.3/", source)
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            card = base / "card"
            (card / "Scripts").mkdir(parents=True)
            normal = card / "misterzine"
            normal.mkdir()
            (normal / "settings.json").write_text("normal settings")
            archive = base / "test.zip"
            with zipfile.ZipFile(archive, "w") as z:
                for name in ("mz-neogeo", "mzmglprobe", "launch.sh"):
                    z.writestr("misterzine-neogeo-test/" + name, "#!/bin/sh\necho test-version\n")
                z.writestr("Scripts/MisterZine-NeoGeo-TEST.sh", "#!/bin/sh\necho launcher\n")
            digest = hashlib.sha256(archive.read_bytes()).hexdigest()
            script = source.replace("CARD=/media/fat", "CARD=" + str(card))
            script = script.replace(next(s for s in script.splitlines() if s.startswith("URL=")), "URL=file://" + str(archive))
            script = script.replace(next(s for s in script.splitlines() if s.startswith("SHA=")), "SHA=" + digest)
            script = script.replace("--proto '=https'", "--proto '=file'")
            def run(text):
                return subprocess.run(["bash", "-c", text], input="", text=True, capture_output=True)
            result = run(script)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            root = card / "misterzine-neogeo-test"
            (root / "settings.json").write_text("test settings")
            (root / "mz-neogeo").write_text("broken")
            result = run(script)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual((root / "settings.json").read_text(), "test settings")
            self.assertIn("test-version", (root / "mz-neogeo").read_text())
            before = {str(p.relative_to(card)): p.read_bytes() for p in card.rglob("*") if p.is_file()}
            result = run(script.replace("SHA=" + digest, "SHA=" + "0" * 64))
            self.assertNotEqual(result.returncode, 0)
            after = {str(p.relative_to(card)): p.read_bytes() for p in card.rglob("*") if p.is_file()}
            self.assertEqual(before, after)
            self.assertEqual((normal / "settings.json").read_text(), "normal settings")
            self.assertFalse(list(card.glob(".mz-neogeo-install.*")))


if __name__ == "__main__":
    unittest.main()
