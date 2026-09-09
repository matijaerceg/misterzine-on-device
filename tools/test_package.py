import importlib.util
from pathlib import Path
import tempfile
import unittest
import zipfile
import json

from verify_package import verify

spec = importlib.util.spec_from_file_location("make_db", Path(__file__).with_name("make-db.py"))
make_db = importlib.util.module_from_spec(spec)
spec.loader.exec_module(make_db)


class PackageTest(unittest.TestCase):
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
            self.assertIn("Scripts/MisterZine Setup.sh", paths)
            self.assertIn("Scripts/MisterZine Uninstall.sh", paths)
            self.assertIn("misterzine/SPLEEN-LICENSE", paths)
            self.assertFalse(any(Path(p).name in ("debug.flag", "settings.json", "favorites.json", "state.json") for p in paths))
            (directory / "launch.sh").write_bytes(b"damaged download")
            with self.assertRaisesRegex(ValueError, "does not match"):
                verify(directory, "v1.0.0-rc.1")


if __name__ == "__main__":
    unittest.main()
