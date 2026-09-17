# How alternate versions are grouped

The catalog keeps the selected release (`sn`) separate from its MAME game
family (`family`). `family_sets` lists known clone setnames in that family.
These optional fields come from the catalog generator's existing MAME metadata;
older clients can ignore them and older catalogs remain supported.

The background card scan reads installed alternative MRA headers. A compatible
core is always required. An alternative matches a row when its setname, parent,
or ROM ZIP stem identifies that row's release or game family. ZIP directory
prefixes and case do not affect matching; launch paths are unchanged. Shared
BIOS ZIPs, core names alone, and folder names do not establish a relationship.
Separate catalog entries remain separate, even when they share a family.

When catalog family data is absent, the scan reads the installed main MRA's
parent. Missing or malformed metadata preserves the existing setname matches.
Successful main-header reads are cached in memory by path, size and mtime;
failed reads are retried. Version 4 of the alternatives cache includes parent
metadata and file stamps, so old caches rebuild once and in-place edits are
noticed. Resolved lists are applied only to the catalog snapshot they describe.

## Games the catalogue does not list

The same background scan also walks the rest of `_Arcade` (every subfolder
except `cores`, the `_alternatives` folders above, hidden folders and anything
whose name contains "organized", since arcade_organizer's `_Organized` tree
duplicates every MRA many times; symlinks are not followed, the walk stops six
levels down and after 20,000 files). Each MRA's header is read once and cached
per directory in `cache/local.json`, versioned like the alternatives cache.

A file is accounted for when its path is a catalogue MRA, when the family
resolver tied it to a catalogue row, or when its setname or parent names a
catalogue release, family root or known clone. Everything else becomes a
"Local" row: one per setname, with clones filed under their parent when the
parent is on the card too. The copy outside `_alternatives` with the shortest
path is the row's file; further copies are its alternatives. Title, year,
manufacturer, rotation, region, players and buttons come from the header;
the category is shown as a note rather than a genre. Local rows carry no
release dates and never count as new since the last look.

## Audit an installed card

`go run ./cmd/mzaltaudit -card CARD_ROOT -data CATALOG_JSON` reports legacy and
family-aware counts, recovered paths, and unresolved entries in same-named
folders. The folder check is a diagnostic lead, not a grouping rule. The command
does not launch games or alter the card. To measure cached scans, optionally
pass `-cache DISPOSABLE_CACHE_PATH` outside the live app directory.

The reported cold/warm timings include alternative discovery and family
resolution, not the initial core/status scan. Without `-cache`, both passes
re-read the alternative files. No ROM data is required for matching.

The `Local*` fields report the discovery walk separately: files read, rows it
would add, skipped files, and its own cold and warm timings (the warm pass
uses `<cache>.local`). `Locals` lists each would-be row with its file, core and
further copies, which is the quickest way to see what a card holds that the
catalogue does not.
