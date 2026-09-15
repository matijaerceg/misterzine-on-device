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

## Audit an installed card

`go run ./cmd/mzaltaudit -card CARD_ROOT -data CATALOG_JSON` reports legacy and
family-aware counts, recovered paths, and unresolved entries in same-named
folders. The folder check is a diagnostic lead, not a grouping rule. The command
does not launch games or alter the card. To measure cached scans, optionally
pass `-cache DISPOSABLE_CACHE_PATH` outside the live app directory.

The reported cold/warm timings include alternative discovery and family
resolution, not the initial core/status scan. Without `-cache`, both passes
re-read the alternative files. No ROM data is required for matching.
