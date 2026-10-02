"""Verify staged or downloaded release assets before publication."""
import configparser
import hashlib
import json
from pathlib import Path
import re
import sys
from urllib.parse import unquote, urlparse
import zipfile

DB_URL = "https://github.com/matijaerceg/misterzine-on-device/releases/latest/download/misterzine.json.zip"
RELEASES = "https://github.com/matijaerceg/misterzine-on-device/releases/download/"


def verify(directory, tag):
    directory = Path(directory)
    if "-beta" in tag:
        raise ValueError("Beta releases have been retired")
    prefix = RELEASES + tag + "/"
    with zipfile.ZipFile(directory / "misterzine.json.zip") as z:
        db = json.loads(z.read("misterzine.json"))
    if db.get("db_id") != "misterzine":
        raise ValueError("Downloader ID changed")
    for path, entry in db["files"].items():
        if not entry["url"].startswith(prefix):
            raise ValueError("Wrong release URL for " + path)
        name = unquote(urlparse(entry["url"]).path.rsplit("/", 1)[-1])
        if Path(name).name != name:
            raise ValueError("Invalid asset name")
        data = (directory / name).read_bytes()
        if hashlib.md5(data).hexdigest() != entry["hash"] or len(data) != entry["size"]:
            raise ValueError("Asset does not match database: " + path)
    names = set()
    for line in (directory / "SHA256SUMS").read_text().splitlines():
        expected, name = line.split("  ", 1)
        if Path(name).name != name or hashlib.sha256((directory / name).read_bytes()).hexdigest() != expected:
            raise ValueError("SHA256 mismatch: " + name)
        names.add(name)
    if {"MisterZine-Install-Beta.sh", "MisterZine-Switch-To-Stable.sh", "channel.py"} & names:
        raise ValueError("Retired channel scripts in unified package")
    ini = configparser.ConfigParser(inline_comment_prefixes=(";", "#"))
    ini.read(directory / "downloader_misterzine.ini")
    if ini.get("misterzine", "db_url", fallback="") != DB_URL:
        raise ValueError("downloader_misterzine.ini points at the wrong database for " + tag)
    print("Verified", len(db["files"]), "installed files and release checksums")


if __name__ == "__main__":
    verify(sys.argv[1], sys.argv[2])
