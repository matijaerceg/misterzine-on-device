#!/usr/bin/env python3
"""Build the MiSTer downloader database for a release.

    make-db.py <tag> <binary> <out.json.zip>
    make-db.py --check <json.zip>
    make-db.py --distribution <tag> <release-dir> <distribution-dir>

Stages all release assets beside the database and hashes those exact bytes.
User-created preferences, favorites and debug flags are never packaged.

The drop-in downloader_misterzine.ini is a release asset for manual use only:
downloader rejects root-level ini files inside a database ("illegal path").

A vX.Y.Z-beta.N tag builds MisterZine Arcade, the Patreon members' beta: the
same database ID, so it replaces the free files in place, plus channel.py and
MisterZine-Switch-To-Free.sh for the way back. MisterZine-Install-Beta.sh and a
drop-in pointed at the beta are release assets only. --distribution then
writes the files the distribution branch serves at a fixed address: the beta
database as beta.json.zip, catalogue.json that beta builds check for updates,
and both scripts. Stable and candidate releases are built exactly as before.

Schema: https://github.com/MiSTer-devel/Downloader_MiSTer/blob/main/docs/custom-databases.md
"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
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
    "Scripts/MisterZine-Run.sh": ("deploy/Scripts/MisterZine-Run.sh", "MisterZine-Run.sh"),
    "Scripts/MisterZine-Setup.sh": ("deploy/Scripts/MisterZine-Setup.sh", "MisterZine-Setup.sh"),
    "Scripts/MisterZine-Uninstall.sh": ("deploy/Scripts/MisterZine-Uninstall.sh", "MisterZine-Uninstall.sh"),
    "misterzine/LICENSE": ("LICENSE", "LICENSE"),
    "misterzine/LICENSE-CATALOGUE": ("LICENSE-CATALOGUE", "LICENSE-CATALOGUE"),
    "misterzine/SPLEEN-LICENSE": ("internal/fonts/SPLEEN-LICENSE", "SPLEEN-LICENSE"),
    "misterzine/SCIENTIFICA-LICENSE": ("internal/fonts/SCIENTIFICA-LICENSE", "SCIENTIFICA-LICENSE"),
    "misterzine/THIRD-PARTY-NOTICES.txt": ("deploy/THIRD-PARTY-NOTICES.txt", "THIRD-PARTY-NOTICES.txt"),
}
ROOT = Path(__file__).resolve().parent.parent
BETA_TAG = re.compile(r"v([0-9]+)\.([0-9]+)\.([0-9]+)-beta\.([0-9]+)")
# The beta's way back to free, which the app runs as misterzine/channel.py.
BETA_ASSETS = {"misterzine/channel.py": ("deploy/channel.py", "channel.py")}
INSTALL_BETA = "MisterZine-Install-Beta.sh"
SWITCH_TO_FREE = "MisterZine-Switch-To-Free.sh"
SCRIPT_END = "MISTERZINE_CHANNEL"
SCRIPT_NOTES = {
    "beta": ("Installs MisterZine Arcade, the Patreon members' beta, or moves this card's",
             "MisterZine over to it. Put it in Scripts and run it; running it again is",
             "safe. Favorites and settings stay, and Update All keeps the beta current.",
             "MisterZine-Switch-To-Free goes back to the free version."),
    "free": ("Puts the free MisterZine back in place of MisterZine Arcade, the members'",
             "beta. Favorites and settings stay, and Update All follows the free releases",
             "again. Running it again is safe."),
}
DISTRIBUTION_README = """# MisterZine Arcade distribution

Files served at fixed addresses for MisterZine Arcade, the Patreon members'
beta of [MisterZine](https://github.com/matijaerceg/misterzine-on-device).
The release workflow writes this branch for every vX.Y.Z-beta.N tag; do not
edit it by hand.

- `beta.json.zip`: the Downloader database of the newest beta
- `catalogue.json`: the newest beta, which beta builds check for updates
- `MisterZine-Install-Beta.sh`: installs the beta, or moves a card to it
- `MisterZine-Switch-To-Free.sh`: goes back to the free version
"""


def channel_module():
    spec = importlib.util.spec_from_file_location("channel", ROOT / "deploy/channel.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def is_beta(tag):
    return BETA_TAG.fullmatch(tag) is not None


def channel_script(channel):
    """A Scripts entry that carries deploy/channel.py whole, so it works on
    any card; exec hands over to Python, so Downloader may replace the
    script while it runs."""
    source = (ROOT / "deploy/channel.py").read_text(encoding="utf-8")
    if SCRIPT_END in source.splitlines():
        raise ValueError("channel.py contains the script's end marker")
    lines = ["#!/bin/bash"] + ["# " + line for line in SCRIPT_NOTES[channel]] + [
        'command -v python3 >/dev/null || { echo "This needs Python 3, supplied with current MiSTer Linux."; exit 1; }',
        "exec python3 - " + channel + ' "$@" <<\'' + SCRIPT_END + "'",
    ]
    return "\n".join(lines) + "\n" + source + SCRIPT_END + "\n"


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
    beta = is_beta(tag)
    for path, (source, asset) in dict(ASSETS, **(BETA_ASSETS if beta else {})).items():
        target = os.path.join(staging, asset)
        shutil.copyfile(os.path.join(root, source), target)
        files[path] = entry(target, base + quote(asset))
        asset_names.append(asset)
    if beta:
        for name, channel in ((SWITCH_TO_FREE, "free"), (INSTALL_BETA, "beta")):
            with open(os.path.join(staging, name), "w", encoding="utf-8", newline="\n") as script:
                script.write(channel_script(channel))
            asset_names.append(name)
        target = os.path.join(staging, SWITCH_TO_FREE)
        files["Scripts/" + SWITCH_TO_FREE] = entry(target, base + quote(SWITCH_TO_FREE))
        switcher = channel_module()
        with open(os.path.join(staging, "downloader_misterzine.ini"), "w", encoding="utf-8", newline="\n") as ini:
            ini.write(switcher.DROP_IN_TEXT.replace("@URL@", switcher.URLS["beta"]))
    else:
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
    print(f"{out}: {len(files)} files for {tag}" + (" (MisterZine Arcade beta)" if beta else ""))
    for p, e in files.items():
        print(f"  {p}  {e['size']} bytes  {e['hash']}")


def beta_order(tag):
    match = BETA_TAG.fullmatch(tag or "")
    if match is None:
        raise ValueError("Not a beta tag: " + str(tag))
    return tuple(int(part) for part in match.groups())


def distribution(tag, release, out):
    """Write the distribution branch's files for a published beta release,
    from its verified assets. Keeps other channels in the catalogue, does
    nothing for the beta it already serves and refuses an older one, so a
    rerun of an old tag cannot take members back."""
    sys.path.insert(0, str(ROOT / "tools"))
    import beta_batch
    import verify_package
    release, out = Path(release), Path(out)
    beta_order(tag)
    verify_package.verify(release, tag)
    batch = beta_batch.load()[0]
    out.mkdir(parents=True, exist_ok=True)
    path = out / "catalogue.json"
    catalogue = {"schema": 1, "releases": {}}
    if path.exists():
        catalogue = json.loads(path.read_text(encoding="utf-8"))
        if catalogue.get("schema") != 1 or not isinstance(catalogue.get("releases"), dict):
            raise ValueError("Unsupported catalogue.json on the distribution branch")
        current = (catalogue["releases"].get("beta") or {}).get("version")
        if current == tag:
            print("The distribution branch already serves " + tag)
            return False
        if current and beta_order(current) > beta_order(tag):
            raise ValueError("The distribution branch serves " + current + ", which is newer than " + tag)
    db_url = channel_module().URLS["beta"]
    shutil.copyfile(release / "misterzine.json.zip", out / db_url.rsplit("/", 1)[1])
    for name in (INSTALL_BETA, SWITCH_TO_FREE):
        shutil.copyfile(release / name, out / name)
    catalogue["releases"]["beta"] = {"version": tag, "batch": batch, "db_url": db_url}
    path.write_text(json.dumps(catalogue, indent=2, sort_keys=True) + "\n", encoding="utf-8", newline="\n")
    (out / "README.md").write_text(DISTRIBUTION_README, encoding="utf-8", newline="\n")
    print("Distribution files for " + tag + " (batch " + batch + ") in " + str(out))
    return True


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
    elif len(sys.argv) == 5 and sys.argv[1] == "--distribution":
        distribution(sys.argv[2], sys.argv[3], sys.argv[4])
    elif len(sys.argv) == 4:
        build(sys.argv[1], sys.argv[2], sys.argv[3])
    else:
        print(__doc__)
        sys.exit(2)
