"""Install, upgrade, remove and reinstall a package using an installed Downloader.

Usage: python3 tools/test_downloader_integration.py DOWNLOADER.zip PACKAGE_DIR
Only HTTP transport is replaced with byte fixtures. Downloader's URL validation,
hash checks, installation, local store and removal run against a temporary card.
"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import zipfile

RUNNER = r'''
import contextlib
import io
from pathlib import Path
import runpy
import sys
from urllib.parse import unquote

archive, fixtures = sys.argv[1:3]
sys.path.insert(0, archive)
from downloader.http_gateway import HttpGateway

class Response(io.BytesIO):
    status = 200
    def getheader(self, name, default=None):
        return str(len(self.getvalue())) if name.lower() == "content-length" else default

@contextlib.contextmanager
def fixture_http(self, url, *args, **kwargs):
    prefix = "https://example.org/misterzine-fixture/"
    if not url.startswith(prefix):
        raise RuntimeError("Unexpected network request in fixture: " + url)
    name = unquote(url[len(prefix):])
    if Path(name).name != name:
        raise RuntimeError("Invalid fixture asset: " + name)
    with Response((Path(fixtures) / name).read_bytes()) as response:
        yield url, response

HttpGateway.open = fixture_http
sys.argv = [archive] + sys.argv[3:]
runpy.run_path(archive, run_name="__main__")
'''


def exercise(archive, package):
    archive, package = Path(archive).resolve(), Path(package).resolve()
    with tempfile.TemporaryDirectory(prefix="mz-downloader-") as tmp:
        root = Path(tmp).resolve()
        server_dir, card, proc = root / "server", root / "card", root / "proc"
        for path in (server_dir, card, proc):
            path.mkdir()
        for asset in package.iterdir():
            if asset.is_file():
                shutil.copyfile(asset, server_dir / asset.name)

        runner = root / "runner.py"
        runner.write_text(RUNNER)
        base = "https://example.org/misterzine-fixture/"
        installed_archive = card / "Scripts/.config/downloader/downloader_latest.zip"
        installed_archive.parent.mkdir(parents=True)
        shutil.copyfile(archive, installed_archive)
        config = card / "downloader.ini"
        config.write_text("[MiSTer]\nupdate_linux=false\nallow_reboot=0\nstorage_priority=off\n"
            "[misterzine]\ndb_url=" + base + "fixture.json.zip\n"
            "[unrelated]\ndb_url=" + base + "unrelated.json.zip\n")
        env = dict(os.environ, FORCED_BASE_PATH=str(card), DEFAULT_BASE_PATH=str(card),
                   DOWNLOADER_INI_PATH=str(config), DOWNLOADER_LAUNCHER_PATH=str(card / "Scripts/update.sh"),
                   EXTRA_DROP_IN_DATABASE_FILES="", UPDATE_LINUX="false", ALLOW_REBOOT="0",
                   LOGFILE=str(card / "downloader.log"), SKIP_FREE_SPACE_CHECKS="true", DEBUG="true",
                   CURL_SSL="", SSL_CERT_FILE="", FAIL_ON_FILE_ERROR="true")
        env.pop("PC_LAUNCHER", None)

        def run(args, **kwargs):
            # Every invocation is forced into this fixture, including removal.
            command = [sys.executable, str(runner), args[1], str(server_dir), *args[2:]]
            result = subprocess.run(command, env=env, cwd=card, text=True,
                                    capture_output=True, timeout=120)
            if result.returncode:
                print(result.stdout, result.stderr)
                raise RuntimeError("Fixture Downloader failed: " + str(result.returncode))
            return result

        def update():
            result = run([sys.executable, str(installed_archive), "--run-only", "misterzine"])
            if not (card / "misterzine/misterzine").exists():
                raise RuntimeError("Downloader did not install the fixture:\n" + result.stdout)

        def publish(db):
            with zipfile.ZipFile(server_dir / "fixture.json.zip", "w") as z:
                z.writestr("misterzine.json", json.dumps(db))

        (server_dir / "old-binary").write_bytes(b"old release fixture")
        (server_dir / "old-launch.sh").write_bytes(b"#!/bin/bash\n")
        old = {"v": 1, "db_id": "misterzine", "timestamp": 1,
               "folders": {"Scripts": {}, "misterzine": {}}, "files": {}}
        for path, name in (("misterzine/misterzine", "old-binary"), ("Scripts/misterzine.sh", "old-launch.sh")):
            data = (server_dir / name).read_bytes()
            old["files"][path] = {"url": base + name, "size": len(data), "hash": hashlib.md5(data).hexdigest()}
        publish(old)
        update()
        app = card / "misterzine"
        assert (app / "misterzine").read_bytes() == b"old release fixture"
        saved = {"favorites.json": b"favorite fixture", "settings.json": b"preference fixture",
                 "state.json": b"filter and visit fixture"}
        for name, data in saved.items():
            (app / name).write_bytes(data)

        with zipfile.ZipFile(package / "misterzine.json.zip") as z:
            current = json.loads(z.read("misterzine.json"))
        for entry in current["files"].values():
            entry["url"] = base + entry["url"].rsplit("/", 1)[-1]
        publish(current)
        update()
        assert not (card / "Scripts/misterzine.sh").exists(), "old launcher was not removed"
        for path, entry in current["files"].items():
            assert hashlib.md5((card / path).read_bytes()).hexdigest() == entry["hash"], path
        for name, data in saved.items():
            assert (app / name).read_bytes() == data
        assert not (app / "debug.flag").exists()

        spec = importlib.util.spec_from_file_location("fixture_maintenance", package / "maintenance.py")
        maintenance = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(maintenance)
        startup = card / "linux/user-startup.sh"
        startup.parent.mkdir()
        original = b"#!/bin/sh\necho another service\n"
        startup.write_bytes(original + b"# misterzine\n" + maintenance.STARTUP_LINE.encode() + b"\n")
        maintenance.uninstall(card, True, run=run, proc_root=proc)
        assert startup.read_bytes() == original
        assert "[misterzine]" not in config.read_text()
        assert "[unrelated]" in config.read_text()
        for name, data in saved.items():
            assert (app / name).read_bytes() == data

        # Reinstall the same version; Downloader must have forgotten its old records.
        with config.open("a") as f:
            f.write("\n[misterzine]\ndb_url=" + base + "fixture.json.zip\n")
        update()
        assert (app / "launch.sh").exists()
        assert (app / "favorites.json").read_bytes() == saved["favorites.json"]
        maintenance.uninstall(card, False, run=run, proc_root=proc)
        assert not app.exists()
        assert "[unrelated]" in config.read_text()
        print("PASS: packaged install, legacy upgrade, keep-data removal, same-version reinstall, full removal")


if __name__ == "__main__":
    exercise(sys.argv[1], sys.argv[2])
