#!/usr/bin/env python3
"""Move this card's MisterZine to the free release or to MisterZine Arcade,
the Patreon members' beta, through Downloader.

    python3 channel.py beta|free [--card /media/fat] [--point-only]

Both builds install as the one Downloader database, misterzine, so a switch
rewrites only that entry's db_url, wherever the card keeps it: the drop-in
downloader_misterzine.ini, another drop-in, or a [misterzine] section in
downloader.ini. Every other line stays as it was. A card with no entry gets
the drop-in the free release ships, pointed at the chosen build. Downloader
then runs for that database alone and replaces the files in place, so
favorites and settings stay and Update All keeps updating the chosen build.

Both builds show the same main-menu entry, MisterZine Arcade.mgl, which a
switch leaves in place; an entry still under its old name, MisterZine.mgl,
takes the new one. On a card that never had MisterZine the menu entry is
enabled as MisterZine-Setup would. Safe to run again.
MisterZine-Install-Beta.sh and MisterZine-Switch-To-Free.sh carry a copy of
this file, so they work on any card; the beta also installs it as
misterzine/channel.py. --point-only rewrites the entry and nothing else: the
beta runs it when something (MiSTer Companion's Install Center, a re-copied
free drop-in) has pointed its own entry back at the free database, so that
the next Update All keeps the beta.
"""
import argparse
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

DB_ID = "misterzine"
URLS = {
    "free": "https://github.com/matijaerceg/misterzine-on-device/releases/latest/download/misterzine.json.zip",
    "beta": "https://raw.githubusercontent.com/matijaerceg/misterzine-arcade-betas/main/beta.json.zip",
}
NAMES = {"free": "the free MisterZine", "beta": "MisterZine Arcade"}
DROP_IN = "downloader_misterzine.ini"
# The drop-in the free release ships (deploy/downloader_misterzine.ini),
# with the chosen build's database in it.
DROP_IN_TEXT = """; misterzine on your MiSTer: the release tracker, on the device itself.
; Copy this file to the SD card root beside downloader.ini, then run Update All
; or Downloader. Keep it there so future runs continue to update MisterZine.
[misterzine]
db_url = @URL@
; Install the whole app even when global filters select only particular cores.
filter =
"""
MGL = "MisterZine Arcade.mgl"
LEGACY_MGL = "MisterZine.mgl"  # the entry's name before the rename
UPDATERS = {
    "update.sh", "update_all.sh", "update_all.pyz", "downloader.sh",
    "downloader_bin", "downloader_latest.zip", "ua_downloader_bin",
    "ua_downloader_dd.pyz", "ua_downloader_latest.zip",
}
# Section headers and db_url keys as Downloader's ini reader takes them:
# section names without case, keys with = or :.
HEADER = re.compile(r"[ \t]*\[([^\]\r\n]+)\][ \t]*(?:[;#].*)?")
KEY = re.compile(r"([ \t]*)db_url[ \t]*[=:]", re.IGNORECASE)


def ini_files(card):
    """The files Downloader reads databases from, in its order: the base
    downloader.ini, then drop-ins in downloader/ and downloader_*.ini."""
    files = [card / "downloader.ini"]
    folder = card / "downloader"
    if folder.is_dir():
        files += sorted(p for p in folder.glob("*.ini") if not p.name.startswith("."))
    files += sorted(p for p in card.glob("downloader_*.ini") if not p.name.startswith("."))
    return [p for p in files if p.is_file()]


def ending(line):
    stripped = line.rstrip("\r\n")
    return line[len(stripped):] or "\n"


def indent(line):
    return len(line) - len(line.lstrip(" \t"))


def header(line):
    match = HEADER.fullmatch(line.rstrip("\r\n"))
    return match.group(1).strip().lower() if match else None


def point(text, url):
    """Rewrite db_url in every [misterzine] section of an ini text, adding
    one under a header that has none. Returns the text and whether the
    section was there; nothing else in the text changes."""
    lines = text.splitlines(keepends=True)
    out, found, inside, i = [], False, False, 0
    while i < len(lines):
        line = lines[i]
        name = header(line)
        if name is not None:
            inside = name == DB_ID
            if inside:
                found = True
                rest = []
                for later in lines[i + 1:]:
                    if header(later) is not None:
                        break
                    rest.append(later)
                if not any(KEY.match(later) for later in rest):
                    if not line.endswith(("\n", "\r")):
                        line += "\n"
                    out.append(line)
                    out.append("db_url = " + url + ending(line))
                    i += 1
                    continue
            out.append(line)
            i += 1
            continue
        key = KEY.match(line) if inside else None
        if key is None:
            out.append(line)
            i += 1
            continue
        out.append(key.group(1) + "db_url = " + url + ending(line))
        i += 1
        # an old value continued on further, deeper-indented lines goes too
        while i < len(lines) and lines[i].strip() and indent(lines[i]) > indent(line) and header(lines[i]) is None:
            i += 1
    return "".join(out), found


def read(path):
    with open(path, "r", encoding="utf-8", errors="surrogateescape", newline="") as f:
        return f.read()


def write(path, text):
    """Replace a file only once its new contents are on the card."""
    mode = path.stat().st_mode & 0o777 if path.exists() else 0o644
    fd, name = tempfile.mkstemp(prefix=".misterzine-", suffix=".tmp", dir=str(path.parent))
    try:
        with os.fdopen(fd, "w", encoding="utf-8", errors="surrogateescape", newline="") as f:
            f.write(text)
            f.flush()
            os.fsync(f.fileno())
        os.chmod(name, mode)
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)


def repoint(card, channel):
    """Point every misterzine entry on the card at the channel's database,
    or add the drop-in when there is none. Returns the files that hold it."""
    url = URLS[channel]
    holders = []
    for path in ini_files(card):
        text = read(path)
        updated, found = point(text, url)
        if not found:
            continue
        if path.is_symlink():
            raise RuntimeError(path.name + " is a link. Change its [misterzine] db_url by hand to " + url)
        if updated != text:
            write(path, updated)
        holders.append(path)
    if holders:
        return holders
    path = card / DROP_IN
    if path.is_symlink():
        raise RuntimeError(DROP_IN + " is a link. Replace it with a copy of the one from the release.")
    if path.exists():
        # a drop-in of that name without the section: add it at the end
        text = read(path)
        if text and not text.endswith(("\n", "\r")):
            text += "\n"
        write(path, text + "[misterzine]\ndb_url = " + url + "\nfilter =\n")
    else:
        write(path, DROP_IN_TEXT.replace("@URL@", url))
    return [path]


def busy(proc_root):
    """Refuse while Update All or Downloader is running."""
    if not proc_root.is_dir():
        return
    for proc in proc_root.iterdir():
        if not proc.name.isdigit() or int(proc.name) == os.getpid():
            continue
        try:
            args = (proc / "cmdline").read_bytes().rstrip(b"\0").split(b"\0")
        except OSError:
            continue
        if any(Path(os.fsdecode(arg)).name in UPDATERS for arg in args if arg):
            raise RuntimeError("An updater is running. Let it finish, then try again.")


def rename_entry(card):
    """Give a menu entry still under its old name the new one, the way the
    launcher does when it starts, so the menu never lists both."""
    legacy = [p for p in card.iterdir() if p.name.lower() == LEGACY_MGL.lower() and (p.is_symlink() or p.is_file())]
    if not legacy:
        return
    entry = card / MGL
    if not (entry.is_symlink() or entry.exists()):
        os.replace(legacy[0], entry)
        legacy = legacy[1:]
    for path in legacy:
        path.unlink()


def downloader(card):
    """How to run Downloader for this database alone, the way the app does:
    the card's own launcher first, since it also fixes the clock and the
    certificates, then the copy Update All keeps. None when there is none."""
    env = dict(os.environ, DOWNLOADER_LAUNCHER_PATH=str(card / "Scripts/downloader.sh"), PYTHONUTF8="1")
    for cert in (card / "Scripts/.config/downloader/cacert.pem", Path("/etc/ssl/certs/cacert.pem")):
        if cert.is_file() and cert.stat().st_size > 0:
            env["SSL_CERT_FILE"] = str(cert)
            break
    only = ["--run-only", DB_ID]
    script = card / "Scripts/downloader.sh"
    config = card / "Scripts/.config/downloader"
    if script.is_file():
        return ["/bin/bash", str(script)] + only, env
    if (config / "downloader_bin").is_file():
        return [str(config / "downloader_bin")] + only, env
    if (config / "downloader_latest.zip").is_file():
        return [sys.executable, str(config / "downloader_latest.zip")] + only, env
    return None, env


def switch(card, channel, run=subprocess.run, proc_root=Path("/proc")):
    card = Path(card)
    if not card.is_dir():
        raise RuntimeError("There is no SD card at " + str(card) + ".")
    card = card.resolve()
    busy(proc_root)
    binary = card / "misterzine/misterzine"
    fresh = not binary.exists()
    for path in repoint(card, channel):
        print("MisterZine's Downloader entry in " + path.name + " now points at " + NAMES[channel] + ".", flush=True)
    command, env = downloader(card)
    if command is None:
        raise RuntimeError("Downloader is not on this card. Run Update All: it brings " + NAMES[channel]
                           + " with everything else" + (", then run MisterZine-Setup from Scripts once." if fresh else "."))
    scripts = card / "Scripts"
    result = run(command, env=env, cwd=str(scripts if scripts.is_dir() else card))
    if result.returncode:
        raise RuntimeError("Downloader did not finish (status " + str(result.returncode)
                           + "). Run this again, or run Update All, which finishes the switch.")
    if not binary.is_file():
        raise RuntimeError("Downloader finished without installing MisterZine. Check the misterzine entry's filter.")
    rename_entry(card)
    if fresh:
        if run([str(binary), "launcher", "enable"]).returncode:
            print("Run MisterZine-Setup from Scripts once to add the main-menu entry.")
    if hasattr(os, "sync"):
        os.sync()


def point_only(card, channel):
    """Point the card's misterzine entry at the channel's database without
    running Downloader; the files already on the card stay as they are."""
    card = Path(card)
    if not card.is_dir():
        raise RuntimeError("There is no SD card at " + str(card) + ".")
    for path in repoint(card.resolve(), channel):
        print("MisterZine's Downloader entry in " + path.name + " now points at " + NAMES[channel] + ".", flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("channel", choices=sorted(URLS))
    parser.add_argument("--card", default="/media/fat")
    parser.add_argument("--point-only", action="store_true", help="rewrite the entry, do not run Downloader")
    parser.add_argument("--proc", default="/proc", help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.point_only:
        try:
            point_only(args.card, args.channel)
        except (OSError, RuntimeError) as exc:
            print("MisterZine's Downloader entry was not changed: " + str(exc))
            return 1
        return 0
    print()
    try:
        switch(args.card, args.channel, proc_root=Path(args.proc))
    except (OSError, RuntimeError) as exc:
        print("\n" + NAMES[args.channel][0].upper() + NAMES[args.channel][1:] + " was not installed: " + str(exc))
        return 1
    if args.channel == "beta":
        print("\nMisterZine Arcade is installed. Choose it in the main menu and enter the code")
        print("from the Patreon members' post. Favorites and settings came along.")
    else:
        print("\nThe free MisterZine is back. Choose MisterZine Arcade in the main menu.")
        print("Favorites and settings came along.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
