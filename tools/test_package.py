import importlib.util
from pathlib import Path
import tempfile
import unittest
import zipfile
import json
import subprocess
import os
from urllib.parse import unquote

from verify_package import verify

spec = importlib.util.spec_from_file_location("make_db", Path(__file__).with_name("make-db.py"))
make_db = importlib.util.module_from_spec(spec)
spec.loader.exec_module(make_db)


class PackageTest(unittest.TestCase):
    @unittest.skipUnless(os.name == "posix", "requires MiSTer's POSIX shell behavior")
    def test_scripts_launch_with_mister_shell_command(self):
        with tempfile.TemporaryDirectory(prefix="mz-scripts-") as tmp:
            root = Path(tmp)
            card, package = root / "card", root / "package"
            app = card / "misterzine"
            app.mkdir(parents=True)
            binary = app / "misterzine"
            binary.write_text('#!/bin/bash\nprintf "%s\\n" "$*" > "' + str(app / "setup-called") + '"\n')
            binary.chmod(0o700)
            (app / "maintenance.py").write_text(
                "from pathlib import Path\nPath(" + repr(str(app / "uninstall-called")) + ").touch()\n"
            )
            database = package / "misterzine.json.zip"
            make_db.build("v1.0.2-test", str(binary), str(database))
            with zipfile.ZipFile(database) as z:
                files = json.loads(z.read("misterzine.json"))["files"]
            for path, entry in files.items():
                if not path.startswith("Scripts/"):
                    continue
                target = card / path
                target.parent.mkdir(exist_ok=True)
                asset = unquote(entry["url"].rsplit("/", 1)[1])
                target.write_text((package / asset).read_text().replace("/media/fat", str(card)))
                target.chmod(0o700)
                # Main's MENU_SCRIPTS_FB command does not quote the script path.
                # Its final echo can mask a failed invocation with exit status 0.
                command = f"cd $(dirname {target})\n{target}\necho 'Press any key to continue'\n"
                result = subprocess.run(["bash", "-c", command], text=True, capture_output=True, timeout=5)
                if "Run" in path:
                    # No launch.sh on this card: the entry must say so, not hang or fail silently.
                    self.assertIn("MisterZine is missing", result.stdout, path + ": " + result.stderr)
                    continue
                marker = app / ("setup-called" if "Setup" in path else "uninstall-called")
                self.assertTrue(marker.exists(), path + ": " + result.stderr)
            self.assertEqual((app / "setup-called").read_text(), "launcher enable\n")

    def test_exact_assets_and_no_user_state(self):
        with tempfile.TemporaryDirectory(prefix="mz-package-") as tmp:
            directory = Path(tmp)
            binary = directory / "input"
            binary.write_bytes(b"device binary fixture")
            database = directory / "misterzine.json.zip"
            make_db.build("v1.0.0-rc.1", str(binary), str(database))
            verify(directory, "v1.0.0-rc.1")
            with zipfile.ZipFile(database) as z:
                paths = set(json.loads(z.read("misterzine.json"))["files"])
            self.assertIn("Scripts/MisterZine-Run.sh", paths)
            self.assertIn("Scripts/MisterZine-Setup.sh", paths)
            self.assertIn("Scripts/MisterZine-Uninstall.sh", paths)
            self.assertIn("misterzine/SPLEEN-LICENSE", paths)
            self.assertIn("misterzine/SCIENTIFICA-LICENSE", paths)
            self.assertFalse(any(Path(p).name in ("debug.flag", "settings.json", "favorites.json", "state.json") for p in paths))
            (directory / "launch.sh").write_bytes(b"damaged download")
            with self.assertRaisesRegex(ValueError, "does not match"):
                verify(directory, "v1.0.0-rc.1")


if __name__ == "__main__":
    unittest.main()
