"""Exercise the switch between the free MisterZine and MisterZine Arcade on
isolated card fixtures, never the real card."""
import importlib.util
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("channel", ROOT / "deploy/channel.py")
channel = importlib.util.module_from_spec(spec)
spec.loader.exec_module(channel)
spec = importlib.util.spec_from_file_location("make_db", ROOT / "tools/make-db.py")
make_db = importlib.util.module_from_spec(spec)
spec.loader.exec_module(make_db)

FREE, BETA = channel.URLS["free"], channel.URLS["beta"]
OTHER = "[distribution_mister]\ndb_url = https://example.org/distribution.json.zip\n"


class PointTest(unittest.TestCase):
    def test_free_drop_in_is_the_released_one(self):
        released = (ROOT / "deploy/downloader_misterzine.ini").read_text(encoding="utf-8")
        self.assertEqual(channel.DROP_IN_TEXT.replace("@URL@", FREE), released)
        self.assertEqual(channel.point(released, BETA), (released.replace(FREE, BETA), True))
        self.assertEqual(channel.point(released.replace(FREE, BETA), FREE), (released, True))

    def test_only_the_misterzine_db_url_changes(self):
        text = ("[MiSTer]\r\nverbose = false\r\n\r\n" + OTHER.replace("\n", "\r\n")
                + "[MisterZine] ; the app\r\nfilter = arcade\r\nDB_URL : https://old.example/misterzine.json.zip ; pinned\r\n"
                + "[misterzine_plex]\r\ndb_url = https://example.org/plex.json.zip\r\n")
        updated, found = channel.point(text, BETA)
        self.assertTrue(found)
        self.assertEqual(updated, text.replace("DB_URL : https://old.example/misterzine.json.zip ; pinned", "db_url = " + BETA))

    def test_a_value_continued_on_later_lines_is_replaced_whole(self):
        text = "[misterzine]\ndb_url =\n    https://old.example/misterzine.json.zip\nfilter =\n[other]\ndb_url = x\n"
        self.assertEqual(channel.point(text, FREE)[0], "[misterzine]\ndb_url = " + FREE + "\nfilter =\n[other]\ndb_url = x\n")

    def test_a_section_without_db_url_gets_one(self):
        self.assertEqual(channel.point("[misterzine]\nfilter =\n", BETA)[0], "[misterzine]\ndb_url = " + BETA + "\nfilter =\n")
        self.assertEqual(channel.point(OTHER + "[misterzine]", BETA)[0], OTHER + "[misterzine]\ndb_url = " + BETA + "\n")

    def test_no_section_leaves_the_text_alone(self):
        text = OTHER + "; [misterzine]\n[misterzine_plex]\ndb_url = x\n"
        self.assertEqual(channel.point(text, BETA), (text, False))


class CardTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="mz-channel-")
        self.addCleanup(self.tmp.cleanup)
        self.card = Path(self.tmp.name).resolve()
        self.proc = self.card / "proc"
        self.proc.mkdir()
        (self.card / "Scripts").mkdir()
        self.app = self.card / "misterzine"
        self.calls = []

    def install(self, channel_name):
        self.app.mkdir(exist_ok=True)
        (self.app / "misterzine").write_text(channel_name)

    def downloader(self, args, **kwargs):
        """Downloader as far as the switch sees it: it installs whichever
        build the misterzine entry names, and the binary answers Setup."""
        self.calls.append(args)
        if args[1:] == ["launcher", "enable"]:
            return subprocess.CompletedProcess(args, 0)
        self.assertEqual(args[-2:], ["--run-only", "misterzine"])
        self.assertEqual(kwargs["env"]["DOWNLOADER_LAUNCHER_PATH"], str(self.card / "Scripts/downloader.sh"))
        self.assertEqual(kwargs["cwd"], str(self.card / "Scripts"))
        texts = [p.read_text() for p in channel.ini_files(self.card)]
        self.install("beta" if any(BETA in t for t in texts) else "free")
        return subprocess.CompletedProcess(args, 0)

    def give_downloader(self):
        (self.card / "Scripts/downloader.sh").write_text("#!/bin/bash\n")

    def test_a_card_without_misterzine_gets_the_drop_in_and_setup(self):
        self.give_downloader()
        channel.switch(self.card, "beta", run=self.downloader, proc_root=self.proc)
        drop_in = (self.card / "downloader_misterzine.ini").read_text()
        self.assertEqual(drop_in, channel.DROP_IN_TEXT.replace("@URL@", BETA))
        self.assertEqual((self.app / "misterzine").read_text(), "beta")
        self.assertEqual(self.calls[0], ["/bin/bash", str(self.card / "Scripts/downloader.sh"), "--run-only", "misterzine"])
        self.assertEqual(self.calls[1], [str(self.app / "misterzine"), "launcher", "enable"])

    def test_beta_over_free_in_a_downloader_ini_section(self):
        self.give_downloader()
        self.install("free")
        (self.app / "favorites.json").write_text("kept")
        config = self.card / "downloader.ini"
        config.write_text("[MiSTer]\nupdate_linux = false\n" + OTHER + "[misterzine]\ndb_url = " + FREE + "\nfilter = arcade\n")
        (self.card / "MisterZine.mgl").write_text("free entry")
        channel.switch(self.card, "beta", run=self.downloader, proc_root=self.proc)
        self.assertEqual(config.read_text(), "[MiSTer]\nupdate_linux = false\n" + OTHER + "[misterzine]\ndb_url = " + BETA + "\nfilter = arcade\n")
        self.assertFalse((self.card / "downloader_misterzine.ini").exists())
        self.assertEqual((self.app / "misterzine").read_text(), "beta")
        self.assertEqual((self.app / "favorites.json").read_text(), "kept")
        # the entry under its old name takes the new one
        self.assertEqual(self.entries(), ["MisterZine Arcade.mgl"])
        self.assertEqual((self.card / "MisterZine Arcade.mgl").read_text(), "free entry")
        self.assertEqual(len(self.calls), 1, "Setup ran on a card that already had MisterZine")

    def entries(self):
        return sorted(p.name for p in self.card.glob("*.mgl"))

    def test_the_old_entry_goes_beside_the_new_one_either_way(self):
        self.give_downloader()
        for build in ("free", "beta"):
            self.install(build)
            (self.card / "MisterZine Arcade.mgl").write_text("entry")
            (self.card / "misterzine.mgl").write_text("old entry")
            (self.card / "MisterZine Tools.mgl").write_text("someone else's")
            channel.switch(self.card, build, run=self.downloader, proc_root=self.proc)
            self.assertEqual(self.entries(), ["MisterZine Arcade.mgl", "MisterZine Tools.mgl"], build)
            self.assertEqual((self.card / "MisterZine Arcade.mgl").read_text(), "entry", build)

    def test_free_over_beta_everywhere_the_entry_is_and_keeps_the_arcade_entry(self):
        config = self.card / "Scripts/.config/downloader"
        config.mkdir(parents=True)
        (config / "downloader_bin").write_text("#!/bin/bash\n")
        self.install("beta")
        (self.card / "downloader.ini").write_text(OTHER + "[misterzine]\ndb_url = " + BETA + "\n")
        (self.card / "downloader").mkdir()
        (self.card / "downloader/extra.ini").write_text("[misterzine]\ndb_url = " + BETA + "\n")
        (self.card / "downloader_misterzine.ini").write_text(channel.DROP_IN_TEXT.replace("@URL@", BETA))
        (self.card / "MisterZine Arcade.mgl").write_text("beta entry")
        channel.switch(self.card, "free", run=self.downloader, proc_root=self.proc)
        self.assertEqual(self.calls, [[str(config / "downloader_bin"), "--run-only", "misterzine"]])
        for path in ("downloader.ini", "downloader/extra.ini", "downloader_misterzine.ini"):
            text = (self.card / path).read_text()
            self.assertIn("db_url = " + FREE, text, path)
            self.assertNotIn(BETA, text, path)
        self.assertEqual((self.card / "downloader_misterzine.ini").read_text(), (ROOT / "deploy/downloader_misterzine.ini").read_text())
        # both builds show MisterZine Arcade in the main menu
        self.assertEqual(self.entries(), ["MisterZine Arcade.mgl"])
        self.assertEqual((self.app / "misterzine").read_text(), "free")

    def test_rerunning_changes_nothing_more(self):
        self.give_downloader()
        channel.switch(self.card, "free", run=self.downloader, proc_root=self.proc)
        first = (self.card / "downloader_misterzine.ini").read_bytes()
        channel.switch(self.card, "free", run=self.downloader, proc_root=self.proc)
        self.assertEqual((self.card / "downloader_misterzine.ini").read_bytes(), first)
        self.assertEqual(sorted(p.name for p in self.card.glob("downloader*.ini")), ["downloader_misterzine.ini"])
        self.assertEqual([c[1:] for c in self.calls].count(["launcher", "enable"]), 1)

    def test_a_drop_in_of_that_name_without_the_section_gets_it(self):
        self.give_downloader()
        (self.card / "downloader_misterzine.ini").write_text("; notes only")
        channel.switch(self.card, "beta", run=self.downloader, proc_root=self.proc)
        self.assertEqual((self.card / "downloader_misterzine.ini").read_text(),
                         "; notes only\n[misterzine]\ndb_url = " + BETA + "\nfilter =\n")

    def test_downloader_failure_keeps_the_arcade_entry(self):
        self.give_downloader()
        self.install("beta")
        (self.card / "MisterZine Arcade.mgl").write_text("beta entry")
        with self.assertRaisesRegex(RuntimeError, "did not finish"):
            channel.switch(self.card, "free", run=lambda *a, **k: subprocess.CompletedProcess(a, 3), proc_root=self.proc)
        self.assertTrue((self.card / "MisterZine Arcade.mgl").exists())
        # the entry already names free, so the next Update All finishes the switch
        self.assertIn(FREE, (self.card / "downloader_misterzine.ini").read_text())

    def test_no_downloader_points_the_entry_and_says_update_all(self):
        with self.assertRaisesRegex(RuntimeError, "Run Update All"):
            channel.switch(self.card, "beta", run=self.downloader, proc_root=self.proc)
        self.assertIn(BETA, (self.card / "downloader_misterzine.ini").read_text())
        self.assertFalse(self.calls)

    def test_a_running_updater_is_refused_before_changes(self):
        self.give_downloader()
        busy = self.proc / "4242"
        busy.mkdir()
        (busy / "cmdline").write_bytes(b"/bin/bash\0/media/fat/Scripts/update_all.sh\0")
        with self.assertRaisesRegex(RuntimeError, "updater is running"):
            channel.switch(self.card, "beta", run=self.downloader, proc_root=self.proc)
        self.assertFalse((self.card / "downloader_misterzine.ini").exists())
        self.assertFalse(self.calls)

    def test_downloader_that_installs_nothing_is_a_failure(self):
        self.give_downloader()
        with self.assertRaisesRegex(RuntimeError, "without installing"):
            channel.switch(self.card, "beta", run=lambda *a, **k: subprocess.CompletedProcess(a, 0), proc_root=self.proc)

    @unittest.skipUnless(os.name == "posix", "symbolic links")
    def test_a_linked_ini_holding_the_entry_is_left_to_the_user(self):
        self.give_downloader()
        target = self.card / "elsewhere.ini"
        target.write_text("[misterzine]\ndb_url = " + FREE + "\n")
        (self.card / "downloader_misterzine.ini").symlink_to(target)
        with self.assertRaisesRegex(RuntimeError, "is a link"):
            channel.switch(self.card, "beta", run=self.downloader, proc_root=self.proc)
        self.assertEqual(target.read_text(), "[misterzine]\ndb_url = " + FREE + "\n")


FAKE_DOWNLOADER = r"""#!/bin/bash
# Installs the build the misterzine entry names, as Downloader would.
card=$(cd "$(dirname "$0")/.." && pwd)
echo "downloader $*" >> "$card/calls"
[ "$*" = "--run-only misterzine" ] || exit 9
mkdir -p "$card/misterzine"
if cat "$card"/downloader*.ini 2>/dev/null | grep -q distribution/beta.json.zip; then build=beta; else build=free; fi
printf '#!/bin/bash\necho "%s $*" >> "%s/calls"\n' "$build" "$card" > "$card/misterzine/misterzine"
chmod +x "$card/misterzine/misterzine"
echo "$build" > "$card/installed"
"""


@unittest.skipUnless(os.name == "posix" and shutil.which("python3"), "the scripts run under bash and python3")
class ScriptTest(unittest.TestCase):
    """The generated Scripts entries, run whole the way MiSTer runs them."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="mz-channel-scripts-")
        self.addCleanup(self.tmp.cleanup)
        self.card = Path(self.tmp.name).resolve() / "card"
        (self.card / "Scripts").mkdir(parents=True)
        (self.card / "proc").mkdir()
        downloader = self.card / "Scripts/downloader.sh"
        downloader.write_text(FAKE_DOWNLOADER)
        downloader.chmod(0o755)
        for name, which in ((make_db.INSTALL_BETA, "beta"), (make_db.SWITCH_TO_FREE, "free")):
            script = self.card / "Scripts" / name
            script.write_text(make_db.channel_script(which))
            script.chmod(0o755)

    def run_script(self, name):
        script = self.card / "Scripts" / name
        # Main's MENU_SCRIPTS_FB command: cd to Scripts, the path unquoted
        command = f"cd {script.parent}\n{script} --card {self.card} --proc {self.card / 'proc'}\n"
        return subprocess.run(["bash", "-c", command], text=True, capture_output=True, timeout=30)

    def calls(self):
        return (self.card / "calls").read_text().splitlines()

    def test_install_switch_back_and_again(self):
        result = self.run_script(make_db.INSTALL_BETA)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("MisterZine Arcade is installed", result.stdout)
        self.assertEqual((self.card / "installed").read_text(), "beta\n")
        self.assertEqual(self.calls(), ["downloader --run-only misterzine", "beta launcher enable"])
        (self.card / "misterzine/favorites.json").write_text("kept")
        (self.card / "MisterZine Arcade.mgl").write_text("beta entry")

        result = self.run_script(make_db.SWITCH_TO_FREE)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("The free MisterZine is back. Choose MisterZine Arcade in the main menu.", result.stdout)
        self.assertEqual((self.card / "installed").read_text(), "free\n")
        self.assertTrue((self.card / "MisterZine Arcade.mgl").exists())
        self.assertEqual((self.card / "downloader_misterzine.ini").read_text(),
                         (ROOT / "deploy/downloader_misterzine.ini").read_text())
        self.assertEqual((self.card / "misterzine/favorites.json").read_text(), "kept")

        for name, build in ((make_db.SWITCH_TO_FREE, "free"), (make_db.INSTALL_BETA, "beta"), (make_db.INSTALL_BETA, "beta")):
            result = self.run_script(name)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertEqual((self.card / "installed").read_text(), build + "\n")
        self.assertEqual(self.calls().count("beta launcher enable"), 1)
        self.assertEqual(sorted(p.name for p in self.card.glob("downloader*.ini")), ["downloader_misterzine.ini"])

    def test_a_failure_says_so_and_exits_nonzero(self):
        (self.card / "Scripts/downloader.sh").write_text("#!/bin/bash\nexit 4\n")
        result = self.run_script(make_db.INSTALL_BETA)
        self.assertEqual(result.returncode, 1)
        self.assertIn("MisterZine Arcade was not installed: Downloader did not finish (status 4)", result.stdout)


if __name__ == "__main__":
    unittest.main()
