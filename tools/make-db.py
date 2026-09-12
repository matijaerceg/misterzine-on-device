#!/usr/bin/env python3
"""Build the MiSTer downloader database for a release.

    make-db.py <tag> <binary> <out.json.zip>
    make-db.py --check <json.zip>

Stages all release assets beside the database and hashes those exact bytes.
User-created preferences, favorites and debug flags are never packaged.

The drop-in downloader_misterzine.ini is a release asset for manual use only:
downloader rejects root-level ini files inside a database ("illegal path").

Schema: https://github.com/MiSTer-devel/Downloader_MiSTer/blob/main/docs/custom-databases.md
"""
import hashlib
import json
import os
import shutil
import sys
import time
import zipfile
from urllib.parse import quote

DB_ID = "misterzine"
REPO = "matijaerceg/misterzine-on-device"
NAME = "misterzine.json"
ASSETS = {
    "misterzine/launch.sh": ("deploy/launch.sh", "launch.sh"),
    "misterzine/maintenance.py": ("deploy/maintenance.py", "maintenance.py"),
    "Scripts/MisterZine-Open.sh": ("deploy/Scripts/MisterZine-Open.sh", "MisterZine-Open.sh"),
    "Scripts/MisterZine-Setup.sh": ("deploy/Scripts/MisterZine-Setup.sh", "MisterZine-Setup.sh"),
    "Scripts/MisterZine-Uninstall.sh": ("deploy/Scripts/MisterZine-Uninstall.sh", "MisterZine-Uninstall.sh"),
    "MisterZine.mgl": ("deploy/MisterZine.mgl", "MisterZine.mgl"),
    "misterzine/LICENSE": ("LICENSE", "LICENSE"),
    "misterzine/SPLEEN-LICENSE": ("internal/fonts/SPLEEN-LICENSE", "SPLEEN-LICENSE"),
    "misterzine/SCIENTIFICA-LICENSE": ("internal/fonts/SCIENTIFICA-LICENSE", "SCIENTIFICA-LICENSE"),
    "misterzine/THIRD-PARTY-NOTICES.txt": ("deploy/THIRD-PARTY-NOTICES.txt", "THIRD-PARTY-NOTICES.txt"),
}


def md5(path):
    h = hashlib.md5()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def entry(path, url):
    return {"hash": md5(path), "size": os.path.getsize(path), "url": url}


def build(tag, binary, out):
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    base = f"https://github.com/{REPO}/releases/download/{tag}/"
    staging = os.path.dirname(os.path.abspath(out))
    os.makedirs(staging, exist_ok=True)
    binary_asset = os.path.join(staging, "misterzine")
    if os.path.abspath(binary) != binary_asset:
        shutil.copyfile(binary, binary_asset)
    files = {"misterzine/misterzine": entry(binary_asset, base + "misterzine")}
    asset_names = ["misterzine"]
    for path, (source, asset) in ASSETS.items():
        target = os.path.join(staging, asset)
        shutil.copyfile(os.path.join(root, source), target)
        files[path] = entry(target, base + quote(asset))
        asset_names.append(asset)
    shutil.copyfile(os.path.join(root, "deploy/downloader_misterzine.ini"),
                    os.path.join(staging, "downloader_misterzine.ini"))
    asset_names += ["downloader_misterzine.ini", os.path.basename(out)]
    db = {
        "v": 1,
        "db_id": DB_ID,
        "timestamp": int(time.time()),
        "files": files,
        "folders": {"Scripts": {}, "misterzine": {}},
    }
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr(NAME, json.dumps(db, indent=1, sort_keys=True))
    check(out)
    with open(os.path.join(staging, "SHA256SUMS"), "w", newline="\n") as sums:
        for name in sorted(asset_names):
            with open(os.path.join(staging, name), "rb") as asset:
                sums.write(hashlib.sha256(asset.read()).hexdigest() + "  " + name + "\n")
    print(f"{out}: {len(files)} files for {tag}")
    for p, e in files.items():
        print(f"  {p}  {e['size']} bytes  {e['hash']}")


def check(path):
    with zipfile.ZipFile(path) as z:
        names = z.namelist()
        assert names == [NAME], f"zip must contain exactly {NAME}, got {names}"
        db = json.loads(z.read(NAME))
    assert db.get("v") == 1, "v must be 1"
    assert db.get("db_id") == DB_ID, f"db_id must be {DB_ID}"
    assert isinstance(db.get("timestamp"), int), "timestamp must be an int"
    assert db.get("files"), "no files"
    for p, e in db["files"].items():
        assert not p.startswith("/") and ".." not in p, f"bad path {p}"
        for k in ("hash", "size", "url"):
            assert k in e, f"{p} lacks {k}"
        assert len(e["hash"]) == 32, f"{p} hash is not md5"
        assert e["url"].startswith("https://"), f"{p} url must be https"
    for p in db.get("folders", {}):
        assert not p.startswith("/") and ".." not in p, f"bad folder {p}"
    print(f"{path}: ok ({len(db['files'])} files)")


if __name__ == "__main__":
    if len(sys.argv) == 3 and sys.argv[1] == "--check":
        check(sys.argv[2])
    elif len(sys.argv) == 4:
        build(sys.argv[1], sys.argv[2], sys.argv[3])
    else:
        print(__doc__)
        sys.exit(2)
