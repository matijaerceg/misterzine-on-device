"""Exercise removal against isolated card fixtures, never the real card."""
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
import zipfile

spec = importlib.util.spec_from_file_location("maintenance", Path(__file__).resolve().parents[1] / "deploy/maintenance.py")
maintenance = importlib.util.module_from_spec(spec)
spec.loader.exec_module(maintenance)


class RemovalTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="mz-removal-")
        self.addCleanup(self.tmp.cleanup)
        self.card = Path(self.tmp.name).resolve()
        self.app = self.card / "misterzine"
        self.app.mkdir()
        self.proc = self.card / "proc"
        self.proc.mkdir()
        self.archive = self.card / "Scripts/.config/downloader/downloader_latest.zip"
        self.archive.parent.mkdir(parents=True)
        with zipfile.ZipFile(self.archive, "w") as z:
            z.writestr("downloader/main.py", "parser.add_argument('--uninstall')")
        self.startup = self.card / "linux/user-startup.sh"
        self.startup.parent.mkdir()
        self.other_hook = b"#!/bin/sh\n# another service\n/usr/bin/fan start\n"
        self.startup.write_bytes(self.other_hook + b"# misterzine\n" + maintenance.STARTUP_LINE.encode() + b"\n")
        for name in ("favorites.json", "favorites.json.bad", "settings.json", "state.json"):
            (self.app / name).write_bytes(("original " + name).encode())
        (self.app / "shots").mkdir()
        (self.app / "shots/a.png").write_bytes(b"cached picture")
        (self.app / "misterzine").write_bytes(b"binary")
        (self.card / "unrelated.txt").write_bytes(b"keep")
        self.calls = []

    def downloader(self, args, **kwargs):
        self.calls.append(args)
        self.assertEqual(args[-2:], ["--uninstall", "misterzine"])
        self.assertEqual(Path(args[1]), self.archive)
        (self.app / "misterzine").unlink()
        return subprocess.CompletedProcess(args, 0)

    def test_keep_preserves_originals_and_recovery_files(self):
        maintenance.uninstall(self.card, True, run=self.downloader, proc_root=self.proc)
        self.assertEqual(sorted(p.name for p in self.app.iterdir()),
                         ["favorites.json", "favorites.json.bad", "settings.json", "state.json"])
        self.assertEqual((self.app / "favorites.json").read_bytes(), b"original favorites.json")
        self.assertEqual((self.app / "state.json").read_bytes(), b"original state.json")
        self.assertEqual(self.startup.read_bytes(), self.other_hook)
        self.assertEqual((self.card / "unrelated.txt").read_bytes(), b"keep")
        self.assertEqual(len(self.calls), 1)

    def test_remove_everything_stays_inside_app_directory(self):
        maintenance.uninstall(self.card, False, run=self.downloader, proc_root=self.proc)
        self.assertFalse(self.app.exists())
        self.assertTrue(self.archive.exists())
        self.assertEqual((self.card / "unrelated.txt").read_bytes(), b"keep")

    def test_downloader_failure_preserves_all_local_data(self):
        with self.assertRaisesRegex(RuntimeError, "did not finish"):
            maintenance.uninstall(self.card, False,
                run=lambda *a, **k: subprocess.CompletedProcess(a, 22), proc_root=self.proc)
        self.assertTrue((self.app / "favorites.json").exists())
        self.assertTrue((self.app / "shots/a.png").exists())

    def test_old_downloader_refused_before_changes(self):
        with zipfile.ZipFile(self.archive, "w") as z:
            z.writestr("downloader/main.py", "old version")
        original = self.startup.read_bytes()
        with self.assertRaisesRegex(RuntimeError, "too old"):
            maintenance.uninstall(self.card, False, run=self.downloader, proc_root=self.proc)
        self.assertEqual(self.startup.read_bytes(), original)
        self.assertFalse(self.calls)

    def test_active_app_or_updater_refused_before_changes(self):
        for args in ([str(self.app / "misterzine")], [str(self.app / "misterzine"), "update-worker"],
                     ["/bin/bash", "/media/fat/Scripts/update_all.sh"]):
            with self.subTest(args=args):
                proc = self.proc / "987654"
                proc.mkdir(exist_ok=True)
                (proc / "cmdline").write_bytes(b"\0".join(s.encode() for s in args) + b"\0")
                original = self.startup.read_bytes()
                with self.assertRaises(RuntimeError):
                    maintenance.uninstall(self.card, False, run=self.downloader, proc_root=self.proc)
                self.assertEqual(self.startup.read_bytes(), original)
                self.assertFalse(self.calls)

    def test_symlinked_install_refused(self):
        target = self.card / "elsewhere"
        self.app.rename(target)
        try:
            self.app.symlink_to(target, target_is_directory=True)
        except OSError:
            self.skipTest("symlinks unavailable")
        with self.assertRaisesRegex(RuntimeError, "redirects"):
            maintenance.uninstall(self.card, False, run=self.downloader, proc_root=self.proc)
        self.assertTrue((target / "favorites.json").exists())


if __name__ == "__main__":
    unittest.main()
