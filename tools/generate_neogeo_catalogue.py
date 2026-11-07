"""Generate the shared arcade identity list from pinned MAME/MiSTer metadata.

Only metadata is downloaded. See docs/NEOGEO_CATALOGUE.md for offline inputs.
"""
import argparse
from collections import defaultdict
import csv
import json
from pathlib import Path
import re
import xml.etree.ElementTree as ET

from neogeo_details import add_details, read_source

SOURCES = {
    "neogeo.cpp": "https://raw.githubusercontent.com/mamedev/mame/114a5f239188be9cbf965cf0d0085dd7167e6ed0/src/mame/snk/neogeo.cpp",
    "romsets.xml": "https://raw.githubusercontent.com/MiSTer-devel/NeoGeo_MiSTer/a8496bfe0203c69f13b6c646c4bbf6a95f231482/releases/romsets.xml",
    "software.xml": "https://raw.githubusercontent.com/mamedev/mame/954def46685cd0276671138fbd032036b1a771fb/hash/neogeo.xml",
}
# These are not arcade game releases: CD-only conversions, a home-only
# conversion without credits, and an empty development subsystem.
EXCLUDE = {"zintrckb", "crswd2bl", "froman2b", "lasthope", "mvstemp"}
# MVS releases absent from the MAME driver. See NEOGEO_CATALOGUE.md.
EXTRA = {
    "knightsch": "Knight's Chance",
    "gladmort": "GladMort",
    "cyborgforce": "Cyborg Force",
    "captainbarrel": "Captain Barrel!",
}
# MiSTer spellings/variants which don't directly name a MAME set.
RELATED = {
    "xenocrisis": "xenocris", "quizdask": "quizdais", "wjammss": "wjammers",
    "samsho5x": "samsho5", "svccpru": "svc", "kof2003ps2": "kof2003",
    "samsh5fe": "samsh5sp", "samsh5pf": "samsh5sp",
    "kof98ae2016": "kof98", "kof98Ultimate": "kof98", "kof99ae": "kof99",
}
# Keep the first test build's identities/titles stable for saved favorites.
PRESERVE = {"mslug": "Metal Slug", "aof": "Art of Fighting", "blazstar": "Blazing Star",
            "samsho": "Samurai Shodown", "kof98": "The King of Fighters '98"}


def forms(title):
    """Known metadata spellings plus filesystem-safe forms, not fuzzy matching."""
    results = set()
    for text in [title, *title.split(" / ")]:
        for value in [text, re.sub(r"\s*\([^)]*\)", "", text).strip()]:
            for candidate in [value, value.replace(":", " -"), value.replace(":", "")]:
                candidate = candidate.replace("/", "-").replace("\\", "-").strip()
                if candidate:
                    results.add(candidate)
                    # MiSTer's display metadata often sorts articles last.
                    if ", The" in candidate:
                        results.add("The " + candidate.replace(", The", ""))
                    if candidate.startswith("The "):
                        results.add(candidate[4:])
    return results


def version_label(title, parent, setname, is_parent):
    """A short picker label from the source's English name and set qualifiers."""
    def stem(s):
        return s.split(" / ", 1)[0].split(" (", 1)[0].strip()
    name = stem(title)
    qualifiers = []
    for value in re.findall(r"\(([^()]*)\)", title):
        code, sep, detail = value.partition(", ")
        if re.fullmatch(r"NGM-\w+\s*~\s*NGH-\w+", code):
            value = detail
        elif re.fullmatch(r"NG[MH]-\w+", code):
            value = ("Home cartridge" if code.startswith("NGH") else "Arcade cartridge") + sep + detail
        elif value.startswith("bootleg of "):
            value = "bootleg"
        if value:
            qualifiers.append(value)
    if name != stem(parent):
        return name + "".join(f" ({q})" for q in qualifiers)
    if qualifiers:
        label = ", ".join(qualifiers)
        return label[:1].upper() + label[1:]
    return "Standard cartridge" if is_parent else f"Alternate set ({setname})"


def generate(sources):
    sets = {}
    for line in sources["neogeo.cpp"].splitlines():
        if line.startswith("GAME("):
            fields = next(csv.reader([line[line.index("(")+1:line.rfind(")")]], skipinitialspace=True))
            fields = [f.strip() for f in fields]
            sets[fields[1]] = (fields[2], fields[9])
    if len(sets) < 250:
        raise ValueError("incomplete MAME input")

    def family(name):
        seen = set()
        while name in sets and sets[name][0] != "neogeo":
            if name in seen:
                raise ValueError("cyclic MAME parent")
            seen.add(name)
            name = sets[name][0]
        return name

    games = {}
    set_names = defaultdict(set)
    descriptions = {}
    spellings = defaultdict(set)
    for name, (_, title) in sets.items():
        root = family(name)
        if root in EXCLUDE:
            continue
        if root not in sets:
            raise ValueError("unknown family " + root)
        base = re.sub(r"\s*\([^)]*\)", "", sets[root][1].split(" / ")[0]).replace("/", "-").strip()
        game = games.setdefault(root, {"id": root, "title": PRESERVE.get(root, base), "aliases": set()})
        set_names[root].add(name)
        game["aliases"].update(forms(title))
        descriptions[name] = title
        spellings[name].update(forms(title))
    for entry in ET.fromstring(sources["software.xml"]).findall("software"):
        name = entry.get("name")
        info = " ".join(e.get("value", "") for e in entry.findall("info"))
        if name not in sets and "(MVS)" in info:
            title = entry.findtext("description")
            games[name] = {"id": name, "title": title, "aliases": forms(title)}
    for name, title in EXTRA.items():
        games[name] = {"id": name, "title": title, "aliases": forms(title)}
    for entry in ET.fromstring(sources["romsets.xml"]):
        names = entry.get("name", "").split(",")
        roots = {family(RELATED.get(n, n)) for n in names} & games.keys()
        if len(roots) > 1:
            raise ValueError("ambiguous MiSTer set " + str(names))
        if roots:
            root = roots.pop()
            game = games[root]
            set_names[root].update(names)
            for field in ("altname", "altnamej"):
                if entry.get(field):
                    game["aliases"].update(forms(entry.get(field)))
                    for name in names:
                        spellings[name].update(forms(entry.get(field)))
            for name in names:
                descriptions.setdefault(name, entry.get("altname") or name)
    games["kof98"]["aliases"].add("King of Fighters '98")
    owners = {}
    for game in games.values():
        sets = set_names[game["id"]] - {game["id"]}
        for name in {game["id"], game["title"], *sets, *game["aliases"]}:
            key = name.lower()
            if key in owners and owners[key] != game["id"]:
                raise ValueError("ambiguous alias " + name)
            owners[key] = game["id"]
        game["aliases"] = sorted(game["aliases"] - {game["id"], game["title"]} - sets, key=str.lower)
        if sets:
            game["set_names"] = sorted(sets, key=str.lower)
            root = game["id"]
            parent = descriptions.get(root, game["title"])
            # A shared readable title cannot distinguish a revision. Only an
            # alias unique to a particular set may override the family match.
            users = defaultdict(set)
            for name in sets | {root}:
                for alias in spellings[name]:
                    users[alias.lower()].add(name)
            variants = {}
            for name in [root, *game["set_names"]]:
                label = version_label(descriptions.get(name, name), parent, name, name == root)
                entry = {"label": label}
                aliases = sorted(a for a in spellings[name] if name != root and users[a.lower()] == {name}
                                 and a.lower() != game["title"].lower())
                if aliases:
                    entry["aliases"] = aliases
                variants[name] = entry
            game["variants"] = variants
    return sorted(games.values(), key=lambda g: g["id"])


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sources", type=Path)
    parser.add_argument("--site", type=Path, default=Path(__file__).resolve().parents[2] / "misterzine",
                        help="MisterZine website checkout for the shared arcade metadata and genre rules")
    parser.add_argument("--output", type=Path, default=Path(__file__).resolve().parents[1] / "internal/mgl/neogeo-arcade.json")
    args = parser.parse_args()
    sources = {name: read_source(args.sources, name, url).decode("utf-8") for name, url in SOURCES.items()}
    games = generate(sources)
    add_details(games, args.site, args.sources)
    args.output.write_text(json.dumps(games, indent=2, ensure_ascii=True) + "\n", encoding="utf-8", newline="\n")
    print(f"{len(games)} game families; {sum(len({g['id'], g['title'], *g['aliases'], *g.get('set_names', [])}) for g in games)} names")
