"""Verify staged or downloaded release assets before publication."""
import configparser
import hashlib
import json
from pathlib import Path
import re
import sys
from urllib.parse import unquote, urlparse
import zipfile

# Where each build's drop-in points (deploy/channel.py has the same two).
DB_URLS = {
    False: "https://github.com/matijaerceg/misterzine-on-device/releases/latest/download/misterzine.json.zip",
    True: "https://raw.githubusercontent.com/matijaerceg/misterzine-on-device/distribution/beta.json.zip",
}
# Files only MisterZine Arcade, the members' beta, ships: its way back to free.
BETA_FILES = {"misterzine/channel.py", "Scripts/MisterZine-Switch-To-Free.sh"}
BETA_ASSETS = {"MisterZine-Install-Beta.sh", "MisterZine-Switch-To-Free.sh", "channel.py"}


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
    names = set()
    for line in (directory / "SHA256SUMS").read_text().splitlines():
        expected, name = line.split("  ", 1)
        if Path(name).name != name or hashlib.sha256((directory / name).read_bytes()).hexdigest() != expected:
            raise ValueError("SHA256 mismatch: " + name)
        names.add(name)
    # A beta database must never reach the free channel, nor the free one the beta.
    beta = re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+-beta\.[0-9]+", tag) is not None
    shipped = BETA_FILES & set(db["files"])
    if beta and shipped != BETA_FILES or not beta and shipped:
        raise ValueError("The database's beta files do not match a " + ("beta" if beta else "free") + " release")
    if beta and not BETA_ASSETS <= names or not beta and BETA_ASSETS & names:
        raise ValueError("The release's beta scripts do not match a " + ("beta" if beta else "free") + " release")
    ini = configparser.ConfigParser(inline_comment_prefixes=(";", "#"))
    ini.read(directory / "downloader_misterzine.ini")
    if ini.get("misterzine", "db_url", fallback="") != DB_URLS[beta]:
        raise ValueError("downloader_misterzine.ini points at the wrong database for " + tag)
    print("Verified", len(db["files"]), "installed files and release checksums" + (" (beta)" if beta else ""))


if __name__ == "__main__":
    verify(sys.argv[1], sys.argv[2])
