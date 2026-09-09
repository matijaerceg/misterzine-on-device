#!/usr/bin/env python3
"""Remove MisterZine through Downloader, then clean up its local data."""
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import zipfile

STARTUP_LINE = "[[ -e /media/fat/misterzine/misterzine ]] && /media/fat/misterzine/misterzine launcher start"
SAVED_FILES = ("favorites.json", "settings.json", "state.json")
UPDATERS = {
    "update.sh", "update_all.sh", "update_all.pyz", "downloader.sh",
    "downloader_bin", "downloader_latest.zip", "ua_downloader_bin",
    "ua_downloader_dd.pyz", "ua_downloader_latest.zip",
}


def checked_app_directory(card):
    card = Path(card).resolve(strict=True)
    app = card / "misterzine"
    if app.is_symlink() or app.resolve() != app or not app.is_dir():
        raise RuntimeError("The MisterZine folder is missing or redirects elsewhere.")
    return card, app


def downloader_archive(card):
    archive = card / "Scripts/.config/downloader/downloader_latest.zip"
    try:
        with zipfile.ZipFile(archive) as z:
            source = z.read("downloader/main.py")
    except (OSError, KeyError, zipfile.BadZipFile) as exc:
        raise RuntimeError("Run a current Downloader once before uninstalling MisterZine.") from exc
    if b"'--uninstall'" not in source and b'"--uninstall"' not in source:
        raise RuntimeError("This Downloader is too old to uninstall. Update Downloader first.")
    return archive


def cmdline(proc):
    try:
        return (proc / "cmdline").read_bytes().rstrip(b"\0").split(b"\0")
    except FileNotFoundError:
        return []


def idle_watchers(app, proc_root=Path("/proc")):
    watchers = []
    for proc in proc_root.iterdir():
        if not proc.name.isdigit() or int(proc.name) == os.getpid():
            continue
        args = cmdline(proc)
        if not args or not args[0]:
            continue
        words = [os.fsdecode(arg) for arg in args]
        if words[0] == str(app / "misterzine"):
            if words[1:] == ["launcher", "watch"]:
                watchers.append((proc, args))
            else:
                raise RuntimeError("Quit MisterZine and wait for any update to finish, then try again.")
        if any(Path(word).name in UPDATERS for word in words):
            raise RuntimeError("An updater is running. Let it finish, then try again.")
    return watchers


def remove_startup_hook(card):
    path = card / "linux/user-startup.sh"
    if path.is_symlink():
        raise RuntimeError("The startup script is a link; remove the MisterZine hook manually.")
    if not path.exists():
        return
    original = path.read_bytes()
    kept = []
    for line in original.splitlines(keepends=True):
        stripped = line.strip()
        code = stripped.split(b"#", 1)[0].strip()
        if stripped == b"# misterzine" or code == STARTUP_LINE.encode():
            continue
        kept.append(line)
    replacement = b"".join(kept)
    if replacement == original:
        return
    fd, name = tempfile.mkstemp(prefix=".misterzine-startup-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(replacement)
            f.flush()
            os.fsync(f.fileno())
        os.chmod(name, path.stat().st_mode & 0o777)
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)


def stop_watchers(watchers):
    for proc, expected in watchers:
        if cmdline(proc) != expected:
            continue
        try:
            os.kill(int(proc.name), signal.SIGTERM)
        except ProcessLookupError:
            continue
        deadline = time.monotonic() + 3
        while cmdline(proc) == expected:
            if time.monotonic() >= deadline:
                raise RuntimeError("The menu launcher has not stopped. Restart MiSTer and retry.")
            time.sleep(0.05)


def clean_local_files(card, keep):
    # Recheck after Downloader has run; never follow a replaced directory.
    if not (card / "misterzine").exists():
        return
    card, app = checked_app_directory(card)
    for child in app.iterdir():
        preserve = keep and any(child.name == name or child.name.startswith(name + ".") for name in SAVED_FILES)
        if preserve:
            continue
        if child.is_symlink() or not child.is_dir():
            child.unlink()
        else:
            shutil.rmtree(child)
    if not any(app.iterdir()):
        app.rmdir()


def uninstall(card, keep, run=subprocess.run, proc_root=Path("/proc")):
    card, app = checked_app_directory(card)
    archive = downloader_archive(card)
    watchers = idle_watchers(app, proc_root)
    remove_startup_hook(card)
    stop_watchers(watchers)
    # The installed zip handles --uninstall locally. Running it directly avoids
    # older shell launchers' unrelated clock/certificate setup.
    env = dict(os.environ, DOWNLOADER_LAUNCHER_PATH=str(card / "Scripts/update.sh"), PYTHONUTF8="1")
    result = run([sys.executable, str(archive), "--uninstall", "misterzine"], env=env)
    if result.returncode:
        raise RuntimeError("Downloader did not finish removal. Local saved data was kept. Fix the reported problem and retry; Setup can restore the menu entry.")
    clean_local_files(card, keep)
    # Also remove pre-release/manual launcher entries, which might not be in
    # Downloader's store. These exact paths belong to MisterZine.
    for relative in ("MisterZine.mgl", "misterzine.mgl", "Scripts/misterzine.sh",
                     "Scripts/MisterZine-Setup.sh", "Scripts/MisterZine-Uninstall.sh",
                     "Scripts/MisterZine Setup.sh", "Scripts/MisterZine Uninstall.sh"):
        path = card / relative
        if path.is_symlink() or path.is_file():
            path.unlink()
    if hasattr(os, "sync"):
        os.sync()


def read_key():
    import select
    key = os.read(sys.stdin.fileno(), 1)
    if key == b"\x1b":
        if select.select([sys.stdin], [], [], 0.15)[0]:
            second = os.read(sys.stdin.fileno(), 1)
            if second == b"[" and select.select([sys.stdin], [], [], 0.15)[0]:
                return {b"A": "up", b"B": "down"}.get(os.read(sys.stdin.fileno(), 1), "")
        return "back"
    return {b"\r": "enter", b"\n": "enter", b"1": "up", b"2": "down", b"": "back"}.get(key, "")


def choose_removal():
    import termios
    import tty
    fd = sys.stdin.fileno()
    saved = termios.tcgetattr(fd)
    choice = 0
    try:
        tty.setcbreak(fd)
        while True:
            print("\033[2J\033[HRemove MisterZine\n")
            for i, label in enumerate(("Keep favorites and preferences", "Remove everything")):
                print(("> " if choice == i else "  ") + label)
            print("\nUp/Down choose   A/Enter select   B/Esc cancel", flush=True)
            key = read_key()
            if key == "back":
                return None
            if key in ("up", "down"):
                choice = 0 if key == "up" else 1
            if key == "enter":
                print("\nRemove MisterZine?" if choice == 0 else "\nRemove MisterZine AND all favorites, preferences and pictures?")
                print("A/Enter confirms   B/Esc cancels", flush=True)
                while True:
                    key = read_key()
                    if key == "enter":
                        return choice == 0
                    if key == "back":
                        return None
    finally:
        termios.tcsetattr(fd, termios.TCSADRAIN, saved)


def main():
    if not sys.stdin.isatty():
        print("Open MisterZine-Uninstall from MiSTer's Scripts menu.")
        return 2
    keep = choose_removal()
    if keep is None:
        print("\nCancelled.")
        return 0
    try:
        uninstall("/media/fat", keep)
    except (OSError, RuntimeError, zipfile.BadZipFile) as exc:
        print("\nMisterZine removal stopped:", exc)
        return 1
    print("\nMisterZine removed." + (" Favorites and preferences remain in /media/fat/misterzine/." if keep else ""))
    return 0


if __name__ == "__main__":
    sys.exit(main())
