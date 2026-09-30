import hashlib
import importlib.util
from pathlib import Path
import re
import tempfile
import unittest
import zipfile
import json
import subprocess
import os
from unittest import mock
from urllib.parse import unquote

import beta_batch
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
            self.assertIn("misterzine/LICENSE-CATALOGUE", paths)
            self.assertIn("misterzine/SPLEEN-LICENSE", paths)
            self.assertIn("misterzine/SCIENTIFICA-LICENSE", paths)
            self.assertFalse(any(Path(p).name in ("debug.flag", "settings.json", "favorites.json", "state.json") for p in paths))
            (directory / "launch.sh").write_bytes(b"damaged download")
            with self.assertRaisesRegex(ValueError, "does not match"):
                verify(directory, "v1.0.0-rc.1")


class BetaPackageTest(unittest.TestCase):
    """MisterZine Arcade, the Patreon members' beta, as a release."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="mz-beta-package-")
        self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name)
        self.binary = self.dir / "input"
        self.binary.write_bytes(b"beta binary fixture")
        batch = self.dir / "beta-batch.json"
        batch.write_text(json.dumps({"batch": "arcade-one", "code_sha256": "ab" * 32}))
        patch = mock.patch.object(beta_batch, "BATCH_FILE", batch)
        patch.start()
        self.addCleanup(patch.stop)

    def release(self, tag):
        out = self.dir / tag
        make_db.build(tag, str(self.binary), str(out / "misterzine.json.zip"))
        return out

    def test_beta_ships_the_way_back_and_points_at_the_beta(self):
        release = self.release("v1.2.0-beta.1")
        verify(release, "v1.2.0-beta.1")
        with zipfile.ZipFile(release / "misterzine.json.zip") as z:
            db = json.loads(z.read("misterzine.json"))
        self.assertEqual(db["db_id"], "misterzine")
        self.assertIn("misterzine/channel.py", db["files"])
        self.assertIn("Scripts/MisterZine-Switch-To-Free.sh", db["files"])
        self.assertNotIn("Scripts/MisterZine-Install-Beta.sh", db["files"])
        channel = (Path(make_db.ROOT) / "deploy/channel.py").read_text(encoding="utf-8")
        self.assertEqual((release / "channel.py").read_text(encoding="utf-8"), channel)
        for name in ("MisterZine-Install-Beta.sh", "MisterZine-Switch-To-Free.sh"):
            script = (release / name).read_text(encoding="utf-8")
            self.assertIn(channel, script)
            self.assertTrue(script.startswith("#!/bin/bash\n"))
        self.assertIn("exec python3 - beta", (release / "MisterZine-Install-Beta.sh").read_text(encoding="utf-8"))
        self.assertIn("exec python3 - free", (release / "MisterZine-Switch-To-Free.sh").read_text(encoding="utf-8"))
        ini = (release / "downloader_misterzine.ini").read_text(encoding="utf-8")
        self.assertIn("db_url = https://raw.githubusercontent.com/matijaerceg/misterzine-arcade-betas/main/beta.json.zip\n", ini)

    def test_the_members_licence_ships_only_with_a_beta_built_with_it(self):
        notice = self.dir / "MEMBERS-LICENSE.txt"
        notice.write_text("The members' extras are proprietary.\n")
        with mock.patch.object(make_db, "MEMBERS_LICENSE", (str(notice), "MEMBERS-LICENSE.txt")):
            beta = self.release("v1.2.0-beta.3")
            stable = self.release("v1.2.3")
        # a tree without the file (this repository's own) ships none
        with mock.patch.object(make_db, "MEMBERS_LICENSE", (str(self.dir / "absent.txt"), "MEMBERS-LICENSE.txt")):
            without = self.release("v1.2.0-beta.4")
        verify(beta, "v1.2.0-beta.3")
        verify(stable, "v1.2.3")
        for release, shipped in ((beta, True), (stable, False), (without, False)):
            with zipfile.ZipFile(release / "misterzine.json.zip") as z:
                files = json.loads(z.read("misterzine.json"))["files"]
            self.assertEqual("misterzine/MEMBERS-LICENSE.txt" in files, shipped, release.name)
        self.assertEqual((beta / "MEMBERS-LICENSE.txt").read_text(), notice.read_text())

    def test_stable_and_candidates_are_built_as_before(self):
        for tag in ("v1.2.0", "v1.2.0-rc.2"):
            release = self.release(tag)
            verify(release, tag)
            self.assertEqual((release / "downloader_misterzine.ini").read_bytes(),
                             (Path(make_db.ROOT) / "deploy/downloader_misterzine.ini").read_bytes())
            for name in ("channel.py", "MisterZine-Install-Beta.sh", "MisterZine-Switch-To-Free.sh"):
                self.assertFalse((release / name).exists(), tag + " ships " + name)

    def test_a_package_of_the_wrong_kind_is_refused(self):
        # published on the other build's repository
        with mock.patch.object(make_db, "is_beta", lambda tag: False):
            release = self.release("v1.2.0-beta.1")
        with self.assertRaisesRegex(ValueError, "Wrong release URL"):
            verify(release, "v1.2.0-beta.1")
        with mock.patch.object(make_db, "is_beta", lambda tag: True):
            release = self.release("v1.2.0")
        with self.assertRaisesRegex(ValueError, "Wrong release URL"):
            verify(release, "v1.2.0")
        # at the right addresses, with the other build's files
        with mock.patch.object(make_db, "is_beta", lambda tag: False), mock.patch.object(make_db, "REPO", make_db.BETA_REPO):
            release = self.release("v1.2.0-beta.2")
        with self.assertRaisesRegex(ValueError, "beta files"):
            verify(release, "v1.2.0-beta.2")
        with mock.patch.object(make_db, "is_beta", lambda tag: True), mock.patch.object(make_db, "BETA_REPO", make_db.REPO):
            release = self.release("v1.2.1")
        with self.assertRaisesRegex(ValueError, "beta files"):
            verify(release, "v1.2.1")

    def test_the_betas_repository_serves_the_newest_beta_only(self):
        out = self.dir / "betas"
        first = self.release("v1.2.0-beta.2")
        self.assertTrue(make_db.distribution("v1.2.0-beta.2", first, out))
        self.assertEqual((out / "beta.json.zip").read_bytes(), (first / "misterzine.json.zip").read_bytes())
        self.assertEqual(json.loads((out / "catalogue.json").read_text()), {"schema": 1, "releases": {"beta": {
            "version": "v1.2.0-beta.2", "batch": "arcade-one",
            "db_url": "https://raw.githubusercontent.com/matijaerceg/misterzine-arcade-betas/main/beta.json.zip"}}})
        for name in ("MisterZine-Install-Beta.sh", "MisterZine-Switch-To-Free.sh"):
            self.assertTrue((out / name).is_file(), name)
        # the README there is the members' guide, which the members' release writes
        self.assertFalse((out / "README.md").exists())
        self.assertNotIn("ab" * 32, "".join(p.read_text(errors="replace") for p in out.iterdir()))
        # a rerun of the same tag changes nothing; an older one is refused
        self.assertFalse(make_db.distribution("v1.2.0-beta.2", first, out))
        with self.assertRaisesRegex(ValueError, "newer than v1.2.0-beta.1"):
            make_db.distribution("v1.2.0-beta.1", self.release("v1.2.0-beta.1"), out)
        with self.assertRaisesRegex(ValueError, "Not a beta tag"):
            make_db.distribution("v1.2.0", self.release("v1.2.0"), out)
        # another channel in the catalogue stays; a newer beta replaces this one
        catalogue = json.loads((out / "catalogue.json").read_text())
        catalogue["releases"]["other"] = {"version": "v9.9.9"}
        (out / "catalogue.json").write_text(json.dumps(catalogue))
        newer = self.release("v1.2.1-beta.1")
        self.assertTrue(make_db.distribution("v1.2.1-beta.1", newer, out))
        catalogue = json.loads((out / "catalogue.json").read_text())
        self.assertEqual(catalogue["releases"]["beta"]["version"], "v1.2.1-beta.1")
        self.assertEqual(catalogue["releases"]["other"], {"version": "v9.9.9"})
        self.assertEqual((out / "beta.json.zip").read_bytes(), (newer / "misterzine.json.zip").read_bytes())
        with self.assertRaisesRegex(ValueError, "No beta batch"):
            with mock.patch.object(beta_batch, "BATCH_FILE", self.dir / "missing.json"):
                make_db.distribution("v1.2.2-beta.1", self.release("v1.2.2-beta.1"), out)


class BetaBatchTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="mz-beta-batch-")
        self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name)
        self.batch = self.dir / "beta-batch.json"

    def write(self, data):
        self.batch.write_text(json.dumps(data))

    def test_ldflags_carry_the_channel_batch_and_hash(self):
        self.write({"batch": "arcade-one", "code_sha256": "0f" * 32})
        module = "github.com/matijaerceg/misterzine-on-device/internal/beta"
        self.assertEqual(beta_batch.ldflags(self.batch),
                         "-X " + module + ".Channel=beta -X " + module + ".Batch=arcade-one -X " + module + ".CodeSHA256=" + "0f" * 32)

    def test_a_missing_or_partial_batch_is_refused(self):
        for data, message in (({"batch": "", "code_sha256": "0f" * 32}, "no batch"),
                              ({"batch": "arcade-one", "code_sha256": ""}, "no code hash"),
                              ({"batch": "arcade-one"}, "exactly"),
                              ({"batch": "arcade-one", "code_sha256": "0f" * 32, "code": "123456"}, "exactly"),
                              ({"batch": "Arcade One", "code_sha256": "0f" * 32}, "batch name"),
                              ({"batch": "arcade-one", "code_sha256": "123456"}, "not a lowercase SHA-256")):
            with self.subTest(data=data):
                self.write(data)
                with self.assertRaisesRegex(ValueError, message):
                    beta_batch.ldflags(self.batch)
        self.batch.unlink()
        with self.assertRaisesRegex(ValueError, "No beta batch"):
            beta_batch.ldflags(self.batch)

    def test_a_new_batch_keeps_its_code_private(self):
        private = self.dir / "private"
        secret = beta_batch.new("arcade-one", private, self.batch)
        code = secret.read_text()
        self.assertRegex(code, r"^[0-9]{6}\n$")
        committed = self.batch.read_text()
        self.assertNotIn(code.strip(), committed)
        self.assertEqual(json.loads(committed), {"batch": "arcade-one",
                                                 "code_sha256": hashlib.sha256(code.strip().encode()).hexdigest()})
        with self.assertRaisesRegex(ValueError, "already arcade-one"):
            beta_batch.new("arcade-one", self.dir / "elsewhere", self.batch)
        self.batch.unlink()
        with self.assertRaises(FileExistsError):
            beta_batch.new("arcade-one", private, self.batch)
        self.assertEqual(secret.read_text(), code, "a code was overwritten")
        beta_batch.new("arcade-two", private, self.batch)
        self.assertEqual(beta_batch.load(self.batch)[0], "arcade-two")

    def test_verify_needs_both_in_the_binary(self):
        self.write({"batch": "arcade-one", "code_sha256": "0f" * 32})
        binary = self.dir / "misterzine"
        binary.write_bytes(b"\0arcade-one\0" + b"0f" * 32 + b"\0")
        beta_batch.verify(binary, self.batch)
        binary.write_bytes(b"\0arcade-one\0")
        with self.assertRaisesRegex(ValueError, "does not carry"):
            beta_batch.verify(binary, self.batch)

    def test_no_plain_code_in_the_workflow_or_the_batch(self):
        root = Path(make_db.ROOT)
        for path in (root / ".github/workflows/release.yml", root / "deploy/beta-batch.json"):
            if path.exists():
                self.assertIsNone(re.search(r"(?<![0-9A-Za-z])[0-9]{6}(?![0-9A-Za-z])", path.read_text()), path)


if __name__ == "__main__":
    unittest.main()
