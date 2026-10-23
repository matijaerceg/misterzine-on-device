#!/usr/bin/env python3
"""Apply a reviewed local media manifest to a MisterZine catalogue."""

from __future__ import annotations

import argparse
import json
import os
import re
import struct
import tempfile
import zlib
from pathlib import Path
from typing import Any

try:  # package import in tests, sibling import when run as a script
    from .prepare_media import _jpeg_size
except ImportError:  # pragma: no cover - exercised by the command-line invocation
    from prepare_media import _jpeg_size


SLOTS = {"video": ".mzv", "flyer": ".png"}
SAFE_KEY = re.compile(r"^[a-z0-9][a-z0-9_-]*$")
MAX_VIDEO_BYTES = 8 << 20
MAX_VIDEO_DIMENSION = 320
VIDEO_FPS = 15
MAX_VIDEO_FRAMES = 300
MAX_FLYER_BYTES = 4 << 20
MAX_FLYER_DIMENSION = 640


class MediaCatalogError(ValueError):
    """Input data cannot be applied safely to the catalogue."""


def _validate_mzv(path: Path) -> None:
    try:
        with path.open("rb") as stream:
            data = stream.read(MAX_VIDEO_BYTES + 1)
    except OSError as exc:
        raise MediaCatalogError(f"cannot read video asset {path}: {exc}") from exc
    if len(data) > MAX_VIDEO_BYTES:
        raise MediaCatalogError(f"video asset exceeds {MAX_VIDEO_BYTES} bytes: {path}")
    if len(data) < 12 or data[:4] != b"MZV1":
        raise MediaCatalogError(f"invalid MZV1 header: {path}")
    width, height, fps, frame_count = struct.unpack_from("<HHHH", data, 4)
    if not (1 <= width <= MAX_VIDEO_DIMENSION and 1 <= height <= MAX_VIDEO_DIMENSION):
        raise MediaCatalogError(f"invalid MZV1 dimensions {width}x{height}: {path}")
    if fps != VIDEO_FPS:
        raise MediaCatalogError(f"MZV1 frame rate must be {VIDEO_FPS}: {path}")
    if not (1 <= frame_count <= MAX_VIDEO_FRAMES):
        raise MediaCatalogError(f"invalid MZV1 frame count {frame_count}: {path}")
    offset = 12
    for frame_number in range(1, frame_count + 1):
        if offset + 4 > len(data):
            raise MediaCatalogError(f"truncated MZV1 frame length {frame_number}: {path}")
        (frame_size,) = struct.unpack_from("<I", data, offset)
        offset += 4
        if frame_size == 0 or offset + frame_size > len(data):
            raise MediaCatalogError(f"truncated or empty MZV1 frame {frame_number}: {path}")
        frame = data[offset:offset + frame_size]
        try:
            frame_size_px = _jpeg_size(frame)
        except ValueError as exc:
            raise MediaCatalogError(f"invalid JPEG in MZV1 frame {frame_number}: {path}: {exc}") from exc
        if frame_size_px != (width, height):
            raise MediaCatalogError(f"MZV1 frame {frame_number} dimensions {frame_size_px} do not match header: {path}")
        if not frame.endswith(b"\xff\xd9"):
            raise MediaCatalogError(f"incomplete JPEG in MZV1 frame {frame_number}: {path}")
        offset += frame_size
    if offset != len(data):
        raise MediaCatalogError(f"unexpected trailing bytes after MZV1 frames: {path}")


def _validate_png(path: Path) -> None:
    try:
        with path.open("rb") as stream:
            data = stream.read(MAX_FLYER_BYTES + 1)
    except OSError as exc:
        raise MediaCatalogError(f"cannot read flyer asset {path}: {exc}") from exc
    if len(data) > MAX_FLYER_BYTES:
        raise MediaCatalogError(f"flyer asset exceeds {MAX_FLYER_BYTES} bytes: {path}")
    if not data.startswith(b"\x89PNG\r\n\x1a\n"):
        raise MediaCatalogError(f"invalid PNG signature: {path}")
    offset = 8
    width = height = None
    seen_idat = False
    seen_iend = False
    chunk_number = 0
    while offset < len(data):
        chunk_number += 1
        if offset + 12 > len(data):
            raise MediaCatalogError(f"truncated PNG chunk header {chunk_number}: {path}")
        length = struct.unpack_from(">I", data, offset)[0]
        chunk_type = data[offset + 4:offset + 8]
        chunk_end = offset + 12 + length
        if chunk_end > len(data):
            raise MediaCatalogError(f"truncated PNG chunk {chunk_type!r}: {path}")
        chunk_data = data[offset + 8:offset + 8 + length]
        expected_crc = struct.unpack_from(">I", data, offset + 8 + length)[0]
        actual_crc = zlib.crc32(chunk_type + chunk_data) & 0xffffffff
        if actual_crc != expected_crc:
            raise MediaCatalogError(f"bad PNG CRC for {chunk_type.decode('ascii', 'replace')}: {path}")
        if chunk_number == 1 and (chunk_type != b"IHDR" or length != 13):
            raise MediaCatalogError(f"PNG must begin with a 13-byte IHDR: {path}")
        if chunk_type == b"IHDR":
            if chunk_number != 1 or width is not None or length != 13:
                raise MediaCatalogError(f"invalid PNG IHDR placement or length: {path}")
            width, height = struct.unpack_from(">II", chunk_data)
            if not (1 <= width <= MAX_FLYER_DIMENSION and 1 <= height <= MAX_FLYER_DIMENSION):
                raise MediaCatalogError(f"invalid PNG dimensions {width}x{height}: {path}")
        elif chunk_type == b"IDAT":
            if width is None or seen_iend:
                raise MediaCatalogError(f"PNG IDAT appears outside image data: {path}")
            seen_idat = True
        elif chunk_type == b"IEND":
            if length != 0 or width is None or not seen_idat:
                raise MediaCatalogError(f"invalid PNG IEND: {path}")
            seen_iend = True
            offset = chunk_end
            if offset != len(data):
                raise MediaCatalogError(f"unexpected bytes after PNG IEND: {path}")
            break
        elif seen_idat and chunk_type[:1].isupper():
            raise MediaCatalogError(f"PNG critical chunk follows IDAT: {path}")
        offset = chunk_end
    if not seen_iend:
        raise MediaCatalogError(f"PNG is missing IEND: {path}")


def _nonempty_text(entry: dict[str, Any], field: str, label: str) -> str:
    value = entry.get(field)
    if not isinstance(value, str) or not value.strip():
        raise MediaCatalogError(f"{label}: {field} must be a non-empty string")
    return value.strip()


def validate_manifest(manifest: Any, image_dir: Path) -> list[dict[str, Any]]:
    if not isinstance(manifest, list):
        raise MediaCatalogError("manifest must be a JSON array")

    checked: list[dict[str, Any]] = []
    seen_keys: set[str] = set()
    exact_owners: dict[str, str] = {}
    for i, entry in enumerate(manifest):
        label = f"manifest entry {i + 1}"
        if not isinstance(entry, dict):
            raise MediaCatalogError(f"{label} must be an object")
        key = _nonempty_text(entry, "key", label)
        if not SAFE_KEY.fullmatch(key):
            raise MediaCatalogError(f"{label}: unsafe key {key!r}; use lowercase letters, digits, '_' or '-'")
        if key in seen_keys:
            raise MediaCatalogError(f"{label}: duplicate key {key!r}")
        seen_keys.add(key)

        names = entry.get("setnames")
        if not isinstance(names, list) or not names:
            raise MediaCatalogError(f"{label}: setnames must be a non-empty array")
        normalized_names: list[str] = []
        for name in names:
            if not isinstance(name, str) or not name.strip():
                raise MediaCatalogError(f"{label}: each setname must be a non-empty string")
            name = name.strip()
            if name in normalized_names:
                raise MediaCatalogError(f"{label}: duplicate setname {name!r}")
            owner = exact_owners.get(name)
            if owner is not None:
                raise MediaCatalogError(f"ambiguous setname {name!r} is assigned to both {owner!r} and {key!r}")
            exact_owners[name] = key
            normalized_names.append(name)

        slots = entry.get("slots")
        if not isinstance(slots, list) or not slots:
            raise MediaCatalogError(f"{label}: slots must be a non-empty array")
        normalized_slots: list[str] = []
        for slot in slots:
            if not isinstance(slot, str) or slot not in SLOTS:
                raise MediaCatalogError(f"{label}: unknown slot {slot!r}; expected video or flyer")
            if slot in normalized_slots:
                raise MediaCatalogError(f"{label}: duplicate slot {slot!r}")
            normalized_slots.append(slot)

        # These fields are a record of supplied provenance, not a legal review.
        source_url = _nonempty_text(entry, "source_url", label)
        credit = _nonempty_text(entry, "credit", label)
        permission = _nonempty_text(entry, "permission", label)
        for slot in normalized_slots:
            path = image_dir / slot / f"{key}{SLOTS[slot]}"
            if not path.is_file():
                raise MediaCatalogError(f"{label}: missing {slot} asset: {path}")
            if slot == "video":
                _validate_mzv(path)
            else:
                _validate_png(path)

        checked.append({
            "key": key,
            "setnames": normalized_names,
            "slots": normalized_slots,
            "source_url": source_url,
            "credit": credit,
            "permission": permission,
        })
    return checked


def apply_manifest(catalog: Any, manifest: Any, image_dir: Path) -> list[dict[str, Any]]:
    if not isinstance(catalog, list) or any(not isinstance(row, dict) for row in catalog):
        raise MediaCatalogError("catalogue must be a JSON array of row objects")
    entries = validate_manifest(manifest, image_dir)
    exact: dict[str, dict[str, Any]] = {}
    by_family: dict[str, dict[str, Any]] = {}
    for entry in entries:
        for setname in entry["setnames"]:
            exact[setname] = entry
            by_family[setname] = entry

    output: list[dict[str, Any]] = []
    matched_keys: set[str] = set()
    for index, source_row in enumerate(catalog):
        row = dict(source_row)
        # This manifest is authoritative: rows no longer matched lose stale media.
        row.pop("media_key", None)
        row.pop("media_slots", None)
        sn = row.get("sn")
        match = exact.get(sn) if isinstance(sn, str) else None
        if match is None:
            family = row.get("family")
            match = by_family.get(family) if isinstance(family, str) else None
        if match is not None:
            row["media_key"] = match["key"]
            row["media_slots"] = list(match["slots"])
            matched_keys.add(match["key"])
        output.append(row)
    unused = [entry["key"] for entry in entries if entry["key"] not in matched_keys]
    if unused:
        raise MediaCatalogError(f"manifest entries match no catalogue sn or family: {', '.join(unused)}")
    return output


def _load_json(path: Path) -> Any:
    try:
        with path.open("r", encoding="utf-8") as stream:
            return json.load(stream)
    except (OSError, json.JSONDecodeError) as exc:
        raise MediaCatalogError(f"cannot read JSON from {path}: {exc}") from exc


def _atomic_write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temp_name: str | None = None
    try:
        with tempfile.NamedTemporaryFile("w", encoding="utf-8", newline="\n", dir=path.parent,
                                         prefix=f".{path.name}.", suffix=".tmp", delete=False) as stream:
            temp_name = stream.name
            json.dump(data, stream, ensure_ascii=False, indent=2)
            stream.write("\n")
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temp_name, path)
    finally:
        if temp_name and os.path.exists(temp_name):
            os.unlink(temp_name)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--catalog", required=True, type=Path, help="input catalogue JSON array")
    parser.add_argument("--manifest", required=True, type=Path, help="reviewed media provenance JSON array")
    parser.add_argument("--images", required=True, type=Path, help="local asset root with video/ and flyer/ folders")
    parser.add_argument("--output", required=True, type=Path, help="output catalogue JSON; must differ from --catalog")
    args = parser.parse_args(argv)
    try:
        if args.output.resolve() in (args.catalog.resolve(), args.manifest.resolve()):
            raise MediaCatalogError("--output must differ from both --catalog and --manifest")
        result = apply_manifest(_load_json(args.catalog), _load_json(args.manifest), args.images)
        _atomic_write_json(args.output, result)
    except MediaCatalogError as exc:
        parser.error(str(exc))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
