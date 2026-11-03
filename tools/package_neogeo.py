"""Package the private five-game MGL device pilot, without a Downloader feed."""
import argparse
from pathlib import Path
import zipfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--binary", type=Path, required=True)
parser.add_argument("--probe", type=Path, required=True)
parser.add_argument("--output", type=Path, required=True)
args = parser.parse_args()
repo = Path(__file__).resolve().parents[1]
prefix = "misterzine-neogeo-test/"
files = {
    prefix + "mz-neogeo": args.binary,
    prefix + "mzmglprobe": args.probe,
    prefix + "arcade-pilot.json": repo / "testdata/mgl/arcade-pilot.json",
    prefix + "README.md": repo / "docs/NEOGEO_DEVICE_TEST.md",
    "Scripts/MisterZine Neo Geo TEST.sh": repo / "deploy/neogeo-test.sh",
}
for name in ["LICENSE", "LICENSE-CATALOGUE", "SPLEEN-LICENSE", "SCIENTIFICA-LICENSE", "THIRD-PARTY-NOTICES.txt"]:
    source = repo / name
    if name in ("SPLEEN-LICENSE", "SCIENTIFICA-LICENSE"):
        source = repo / "internal/fonts" / name
    elif name == "THIRD-PARTY-NOTICES.txt":
        source = repo / "deploy" / name
    files[prefix + name] = source
args.output.parent.mkdir(parents=True, exist_ok=True)
with zipfile.ZipFile(args.output, "w", zipfile.ZIP_DEFLATED) as archive:
    for name, source in files.items():
        content = source.read_bytes()
        if source.suffix in (".sh", ".md", ".json", ".txt"):
            content = content.replace(b"\r\n", b"\n")
        entry = zipfile.ZipInfo(name)
        executable = name.endswith(("mz-neogeo", "mzmglprobe", ".sh"))
        entry.external_attr = (0o100755 if executable else 0o100644) << 16
        entry.compress_type = zipfile.ZIP_DEFLATED
        archive.writestr(entry, content)
print(args.output)
