#!/usr/bin/env python3
"""The access batch of MisterZine Arcade, the Patreon members' beta.

    beta_batch.py new BATCH [--private DIR]   start a batch with a new code
    beta_batch.py ldflags                     linker flags for the beta build
    beta_batch.py verify BINARY               check a beta build carries them

A batch is a name and the SHA-256 of its six-digit code. deploy/beta-batch.json
holds both and is committed; the release workflow builds every vX.Y.Z-beta.N
tag with them. The code itself never enters the repository: `new` keeps it in
<private>/<batch>.code (default ~/.misterzine-beta) and prints only that path.
A new batch means a new code for the members' post; a new version alone does
not. This is a convenience gate: six digits are recoverable from their hash.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import sys

ROOT = Path(__file__).resolve().parent.parent
BATCH_FILE = ROOT / "deploy/beta-batch.json"
PRIVATE = Path.home() / ".misterzine-beta"
MODULE = "github.com/matijaerceg/misterzine-on-device/internal/beta"
BATCH = re.compile(r"[a-z0-9][a-z0-9-]{0,47}")
SHA256 = re.compile(r"[0-9a-f]{64}")


def batch_name(value):
    if not BATCH.fullmatch(value):
        raise ValueError("A batch name is 1-48 lowercase letters, digits or hyphens, starting with a letter or digit")
    return value


def load(path=None):
    """The committed batch and code hash; refuses a missing or partial one."""
    path = Path(path or BATCH_FILE)
    if not path.is_file():
        raise ValueError("No beta batch in " + path.name + ". Run tools/beta_batch.py new NAME and commit it.")
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict) or set(data) != {"batch", "code_sha256"}:
        raise ValueError(path.name + " must hold exactly batch and code_sha256")
    batch, digest = data["batch"], data["code_sha256"]
    if not isinstance(batch, str) or not batch:
        raise ValueError(path.name + " has no batch")
    if not isinstance(digest, str) or not digest:
        raise ValueError(path.name + " has no code hash")
    batch_name(batch)
    if not SHA256.fullmatch(digest):
        raise ValueError(path.name + ": code_sha256 is not a lowercase SHA-256")
    return batch, digest


def ldflags(path=None):
    batch, digest = load(path)
    return " ".join(("-X " + MODULE + ".Channel=beta", "-X " + MODULE + ".Batch=" + batch,
                     "-X " + MODULE + ".CodeSHA256=" + digest))


def verify(binary, path=None):
    """The Go linker skips -X for a variable that does not exist, silently;
    a beta build that lacks the batch or the hash would open for anyone."""
    batch, digest = load(path)
    data = Path(binary).read_bytes()
    for value in (batch, digest):
        if value.encode() not in data:
            raise ValueError("The beta build does not carry " + value + ": internal/beta is missing Batch or CodeSHA256")


def new(batch, private=None, path=None):
    """Start a batch: a random six-digit code in a private file that is never
    overwritten, and its hash in the committed batch file."""
    batch_name(batch)
    path, private = Path(path or BATCH_FILE), Path(private or PRIVATE)
    try:
        current = load(path)[0]
    except ValueError:
        current = None
    if current == batch:
        raise ValueError("The current batch is already " + batch + "; choose a new name")
    private.mkdir(parents=True, exist_ok=True)
    secret = private / (batch + ".code")
    code = "%06d" % secrets.randbelow(1000000)
    with open(secret, "x", encoding="ascii") as out:
        out.write(code + "\n")
    try:
        os.chmod(secret, 0o600)
    except OSError:
        pass
    digest = hashlib.sha256(code.encode("ascii")).hexdigest()
    path.write_text(json.dumps({"batch": batch, "code_sha256": digest}, indent=1) + "\n", encoding="utf-8", newline="\n")
    return secret


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    commands = parser.add_subparsers(dest="command", required=True)
    start = commands.add_parser("new", help="start a batch with a new code")
    start.add_argument("batch")
    start.add_argument("--private", type=Path, default=PRIVATE)
    commands.add_parser("ldflags", help="print the linker flags for the beta build")
    check = commands.add_parser("verify", help="check a beta build carries the batch")
    check.add_argument("binary")
    args = parser.parse_args()
    try:
        if args.command == "new":
            secret = new(args.batch, args.private)
            print("New batch " + args.batch + " in " + str(BATCH_FILE.relative_to(ROOT)) + "; commit it.")
            print("Its code is in " + str(secret) + ". Keep a copy; it is not stored anywhere else.")
        elif args.command == "ldflags":
            print(ldflags())
        else:
            verify(args.binary)
            print("The beta build carries batch " + load()[0])
    except (OSError, ValueError) as exc:
        print("beta_batch: " + str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
