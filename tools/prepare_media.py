#!/usr/bin/env python3
"""Prepare local preview videos and flyers for the MisterZine image tree."""

from __future__ import annotations

import argparse
import os
from pathlib import Path
import re
import struct
import subprocess
import tempfile


FPS = 15
MAX_VIDEO_DIMENSION = 320
MAX_VIDEO_SECONDS = 20
MAX_VIDEO_FRAMES = FPS * MAX_VIDEO_SECONDS
MAX_VIDEO_BYTES = 8 << 20
MAX_FLYER_DIMENSION = 640
MAX_FLYER_BYTES = 4 << 20
KEY_RE = re.compile(r"[a-z0-9][a-z0-9_-]*\Z")


def _run(command: list[str]) -> None:
    try:
        subprocess.run(command, check=True)
    except FileNotFoundError as exc:
        raise RuntimeError(f"ffmpeg was not found: {command[0]}") from exc
    except subprocess.CalledProcessError as exc:
        raise RuntimeError(f"ffmpeg failed with exit code {exc.returncode}") from exc


def _jpeg_size(data: bytes) -> tuple[int, int]:
    if not data.startswith(b"\xff\xd8"):
        raise ValueError("frame is not a JPEG")
    pos = 2
    standalone = {0x01, *range(0xD0, 0xDA)}
    sof = {*range(0xC0, 0xC4), *range(0xC5, 0xC8), *range(0xC9, 0xCC), *range(0xCD, 0xD0)}
    while pos < len(data):
        if data[pos] != 0xFF:
            pos += 1
            continue
        while pos < len(data) and data[pos] == 0xFF:
            pos += 1
        if pos >= len(data):
            break
        marker = data[pos]
        pos += 1
        if marker in standalone:
            continue
        if pos + 2 > len(data):
            break
        length = struct.unpack_from(">H", data, pos)[0]
        if length < 2 or pos + length > len(data):
            break
        if marker in sof:
            if length < 7:
                break
            height, width = struct.unpack_from(">HH", data, pos + 3)
            if width and height:
                return width, height
        pos += length
    raise ValueError("JPEG has no usable size marker")


def _png_size(data: bytes) -> tuple[int, int]:
    if len(data) < 24 or data[:8] != b"\x89PNG\r\n\x1a\n" or data[12:16] != b"IHDR":
        raise ValueError("ffmpeg did not produce a valid PNG")
    return struct.unpack_from(">II", data, 16)


def _write_atomic(path: Path, data: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + ".part")
    temporary.write_bytes(data)
    os.replace(temporary, path)


def _encode_mzv(frames: list[bytes]) -> bytes:
    if not 1 <= len(frames) <= MAX_VIDEO_FRAMES:
        raise ValueError(f"video produced {len(frames)} frames; expected 1-{MAX_VIDEO_FRAMES}")
    width, height = _jpeg_size(frames[0])
    if width > MAX_VIDEO_DIMENSION or height > MAX_VIDEO_DIMENSION:
        raise ValueError(f"video frame is {width}x{height}; maximum is {MAX_VIDEO_DIMENSION}x{MAX_VIDEO_DIMENSION}")
    for number, frame in enumerate(frames[1:], 2):
        if _jpeg_size(frame) != (width, height):
            raise ValueError(f"video frame {number} has different dimensions")
    chunks = [b"MZV1", struct.pack("<HHHH", width, height, FPS, len(frames))]
    for frame in frames:
        chunks.extend((struct.pack("<I", len(frame)), frame))
    return b"".join(chunks)


def prepare_video(source: Path, output_dir: Path, key: str, ffmpeg: str = "ffmpeg") -> Path:
    destination = output_dir / "video" / f"{key}.mzv"
    with tempfile.TemporaryDirectory(prefix="misterzine-media-") as temporary:
        frame_dir = Path(temporary)
        pattern = frame_dir / "frame-%03d.jpg"
        for quality in (4, 7, 10, 14, 18, 24, 31):
            for old in frame_dir.glob("frame-*.jpg"):
                old.unlink()
            _run([
                ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", str(source),
                "-an", "-t", str(MAX_VIDEO_SECONDS),
                "-vf", f"fps={FPS},scale={MAX_VIDEO_DIMENSION}:{MAX_VIDEO_DIMENSION}:force_original_aspect_ratio=decrease",
                "-frames:v", str(MAX_VIDEO_FRAMES), "-q:v", str(quality), str(pattern),
            ])
            frames = [path.read_bytes() for path in sorted(frame_dir.glob("frame-*.jpg"))]
            encoded = _encode_mzv(frames)
            if len(encoded) <= MAX_VIDEO_BYTES:
                _write_atomic(destination, encoded)
                return destination
        raise ValueError(f"video cannot fit within {MAX_VIDEO_BYTES} bytes")


def prepare_flyer(source: Path, output_dir: Path, key: str, ffmpeg: str = "ffmpeg") -> Path:
    destination = output_dir / "flyer" / f"{key}.png"
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="misterzine-media-") as temporary:
        rendered = Path(temporary) / "flyer.png"
        _run([
            ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", str(source),
            "-vf", f"scale={MAX_FLYER_DIMENSION}:{MAX_FLYER_DIMENSION}:force_original_aspect_ratio=decrease",
            "-frames:v", "1", str(rendered),
        ])
        data = rendered.read_bytes()
        width, height = _png_size(data)
        if width > MAX_FLYER_DIMENSION or height > MAX_FLYER_DIMENSION:
            raise ValueError(f"flyer is {width}x{height}; maximum is {MAX_FLYER_DIMENSION}x{MAX_FLYER_DIMENSION}")
        if len(data) > MAX_FLYER_BYTES:
            raise ValueError(f"flyer cannot fit within {MAX_FLYER_BYTES} bytes")
        _write_atomic(destination, data)
    return destination


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="kind", required=True)
    for kind in ("video", "flyer"):
        sub = subparsers.add_parser(kind)
        sub.add_argument("input", type=Path, help="local source file")
        sub.add_argument("--output-dir", type=Path, required=True, help="image tree root")
        sub.add_argument("--key", required=True, help="media key used in the catalogue")
        sub.add_argument("--ffmpeg", default="ffmpeg", help=argparse.SUPPRESS)
    return parser


def main(argv: list[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    if not KEY_RE.fullmatch(args.key):
        raise SystemExit("key must contain only lowercase letters, numbers, underscore, or hyphen")
    if not args.input.is_file():
        raise SystemExit(f"input is not a local file: {args.input}")
    try:
        if args.kind == "video":
            output = prepare_video(args.input, args.output_dir, args.key, args.ffmpeg)
        else:
            output = prepare_flyer(args.input, args.output_dir, args.key, args.ffmpeg)
    except (OSError, RuntimeError, ValueError) as exc:
        raise SystemExit(str(exc)) from exc
    print(output)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
