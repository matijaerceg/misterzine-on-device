"""Build the offline Neo Geo game facts. Source editions and credits are in docs/NEOGEO_CATALOGUE.md."""
import gzip
import hashlib
import io
import json
import re
import subprocess
import sys
import zipfile


# Reviewed MVS release year, maker, genre and player mode for the five games
# outside the site's arcade DAT. A missing action-button count stays unknown.
MVS_RELEASES = {
    "captainbarrel": ("2024", "Neo Byte Force", "Puzzle", "2 (alternating)"),
    "cyborgforce": ("2024", "Neo Byte Force", "Run 'n gun", "2 (simultaneous)"),
    "gladmort": ("2025", "PixelHeart / Chipsonsteroids", "Platformer", "1"),
    "knightsch": ("2014", "Neobitz", "Tabletop & quiz", "1"),
    "xenocris": ("2021", "Bitmap Bureau", "Arena shooter", "2 (simultaneous)"),
}


def read_source(directory, name, url):
    if directory:
        return (directory / name).read_bytes()
    return subprocess.check_output(["curl", "--fail", "--silent", "--show-error", "--location",
                                    "--max-time", "60", "--retry", "2", url])


def archive_text(directory, name, url, sha, member, encoding):
    content = read_source(directory, name, url)
    if hashlib.sha256(content).hexdigest() != sha:
        raise ValueError(f"unexpected edition of {name}")
    with zipfile.ZipFile(io.BytesIO(content)) as archive:
        return archive.read(member).decode(encoding)


def player_modes(directory):
    text = archive_text(directory, "nplayers.zip", "https://nplayers.arcadebelgium.be/files/nplayers0278.zip",
                        "4d9dbbe6739fcea68e48951cf0e6937404be53f1ab30fc03a3d7e6466aa13b19", "nplayers.ini", "latin1")
    labels = {"1P": "1", "2P sim": "2 (simultaneous)", "2P alt": "2 (alternating)",
              "4P alt / 2P sim": "4 (alternating) / 2 (simultaneous)"}
    modes = {}
    for line in text.splitlines():
        name, _, mode = line.partition("=")
        if mode in labels:
            modes[name] = labels[mode]
    return modes


def action_buttons(directory):
    text = archive_text(directory, "command.zip", "https://www.progettosnaps.net/download/?tipo=command&file=pS_Command_273.zip",
                        "e281308aedc80ad41f053ac99a5309f107fd9a120e00959e925f3481321afc20", "dats/command.dat", "utf-8-sig")
    counts = {}
    for entry in text.split("$info=")[1:]:
        names, _, body = entry.partition("\n")
        # Only an explicit description of all four face buttons qualifies.
        layout = re.search(r"^_A *: *(.+)\n_B *: *(.+)\n_C *: *(.+)\n_D *: *(.+)$", body, re.M)
        if layout:
            count = sum(action.strip().lower() not in ("not used", "no use") for action in layout.groups())
            counts.update(dict.fromkeys(names.strip().split(","), count))
    # V-Liner's driver distinguishes its four game buttons from operator
    # controls. The newer games' sources are listed alongside their credits.
    counts.update(gladmort=3, xenocris=4, vliner=4)
    return counts


def add_details(games, site_dir, sources_dir):
    # Use the site's own genre vocabulary and mature-content rules offline.
    if not (site_dir / "misterzine.py").is_file():
        raise ValueError("pass --site with the path to the MisterZine website checkout")
    sys.path.insert(0, str(site_dir.resolve()))
    import misterzine as site

    cache = site_dir / "data/cache"
    with gzip.open(cache / "mame_meta.json.gz") as f:
        dat = json.load(f)
    with gzip.open(cache / "mame_current_specs.json.gz") as f:
        specs = json.load(f)["specs"]
    categories = site.local_catver()
    mature = site.local_catver_mature()
    players = player_modes(sources_dir)
    buttons = action_buttons(sources_dir)
    for game in games:
        key = game["id"]
        if key in MVS_RELEASES:
            year, maker, genre, mode = MVS_RELEASES[key]
            rotation, movement, special = "Horizontal", "8-way", ""
        else:
            year, maker = dat[key]["year"], dat[key]["manufacturer"]
            genre, mode = site.genre_for(key, categories, dat), players[key]
            spec = specs[key]
            rotation, special = spec["rot"], spec.get("spc", "")
            # MAME describes the cabinet's inputs. Its button count can exceed
            # the number the game uses, so keep only the movement description.
            movement = re.sub(r"(?:\s*·\s*)?\b\d+ buttons?$", "", spec.get("ctl", "")).strip()
        if not all((year, maker, genre, mode, rotation)):
            raise ValueError(f"incomplete Neo Geo details for {key}")
        details = dict(year=year, manufacturer=maker, genre=genre, plr=mode, rot=rotation)
        if key in buttons:
            n = buttons[key]
            details["buttons"] = n
            movement = " · ".join(filter(None, [movement, f"{n} button" + ("" if n == 1 else "s")]))
        if movement:
            details["ctl"] = movement
        if special:
            details["spc"] = special
        if site.mature_for(key, mature, categories, dat):
            details["mat"] = 1
        game["details"] = details
