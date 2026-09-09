"""Verify staged or downloaded release assets before publication."""
import hashlib
import json
from pathlib import Path
import sys
from urllib.parse import unquote, urlparse
import zipfile


def verify(directory, tag):
    directory = Path(directory)
    prefix = "https://github.com/matijaerceg/misterzine-on-device/releases/download/" + tag + "/"
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
    for line in (directory / "SHA256SUMS").read_text().splitlines():
        expected, name = line.split("  ", 1)
        if Path(name).name != name or hashlib.sha256((directory / name).read_bytes()).hexdigest() != expected:
            raise ValueError("SHA256 mismatch: " + name)
    print("Verified", len(db["files"]), "installed files and release checksums")


if __name__ == "__main__":
    verify(sys.argv[1], sys.argv[2])
