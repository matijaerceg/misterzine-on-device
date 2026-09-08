#!/usr/bin/env python3
"""Build the MiSTer downloader database for a release.

    make-db.py <tag> <binary> <out.json.zip>
    make-db.py --check <json.zip>

The database lists three files, all served from the GitHub release of <tag>,
so the md5 in the database is the md5 of exactly the bytes those URLs serve:
a mismatch would make downloader fetch the file again on every run.

    Scripts/misterzine.sh         from deploy/Scripts/misterzine.sh
    downloader_misterzine.ini     from deploy/downloader_misterzine.ini
    misterzine/misterzine         the cross-built binary

Schema: https://github.com/MiSTer-devel/Downloader_MiSTer/blob/main/docs/custom-databases.md
"""
import hashlib
import json
import os
import sys
import time
import zipfile

DB_ID = "misterzine"
REPO = "matijaerceg/misterzine-on-device"
NAME = "misterzine.json"


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
    files = {
        "Scripts/misterzine.sh": entry(os.path.join(root, "deploy", "Scripts", "misterzine.sh"), base + "misterzine.sh"),
        "downloader_misterzine.ini": entry(os.path.join(root, "deploy", "downloader_misterzine.ini"), base + "downloader_misterzine.ini"),
        "misterzine/misterzine": entry(binary, base + "misterzine"),
    }
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
