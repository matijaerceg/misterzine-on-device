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

## Versions on other cores, and outside `_alternatives`

The compatible-core matcher above never sees two kinds of set a player can run
from MiSTer's menu: a set for another core (Pleiads' own `_alternatives` run on
the Pleiads core while the catalogue row uses Phoenix; Shinobi's System 16B sets
on jts16b beside the jts16 row) and a set of the row's own core outside the
`_alternatives` folders (`_Arcade/Jungle Hunt US.mra` beside Jungle King).
`DiscoverLocal` offers them as further versions, and `scan.MergeVersions` adds
them to the resolver's lists:

- **Ownership is strict, without ROM zip names.** The rows whose `sn` is the
  file's setname own it outright. Otherwise the rows claiming it through its
  parent or a family own it only when they are one release (the same `sn`, as
  with Black Heart in two implementations or Asteroids Deluxe in two colours).
  Families span different titles in the catalogue (Sprint 1 and 2, River
  Patrol and Silver Land, Gradius and Nemesis), so claimants of different
  releases make the file ambiguous, and it is offered nowhere. A family sibling
  never stands in for an exact owner that cannot run.
- **Only rows that run here offer it**, meaning the rows the list shows on the
  card (their scan status is Found: core and main MRA). The file's own core
  must be in `_Arcade/cores` too. Otherwise the stand-in rules below decide, so
  a game whose core is present but whose main MRA is missing shows greyed, and
  a copy that runs stands in for it.
- **Already-attached files** go only to the other runnable owners, on another
  core, with no duplicates. BIOS files are never versions.
- **The picker keeps up with the cores.** `LocalResult.VersionCores` carries
  each file's own core; the host drops a version whose core has left
  `_Arcade/cores` as soon as a rescan's first pass updates the core index, and
  labels one on another core with it. Details keeps the chosen version by
  path, so a rescan that reorders the list never switches the choice.
- **A stand-in that gives way** hands its favorite, remembered version and
  launch records to the row that now offers its file (`VersionOwner`), rather
  than to whichever row covers the setname on paper.

`mzaltaudit` reports these as `ExtraOffers`, `ExtraFiles` and `Extras`.

## Games the catalogue does not list

The same background scan also walks the rest of `_Arcade` (every subfolder
except `cores`, the `_alternatives` folders above, hidden folders and anything
whose name contains "organized", since arcade_organizer's `_Organized` tree
duplicates every MRA many times; symlinks are not followed, the walk stops six
levels down and after 20,000 files). Each MRA's header is read once and cached
per directory in `cache/local.json`, versioned like the alternatives cache.

A file is accounted for when its path is a catalogue MRA, when the family
resolver tied it to a catalogue row, or when it belongs to a catalogue game
that runs on this card (the ownership rules above): then it is one more version
of that game when its own core is on the card, and otherwise the report says
why it is left out. Everything else becomes a "Local" row: one per setname,
with clones filed under their parent when the parent is on the card too.

A game the catalogue knows only through cores the card has not got is the
exception to that rule. Its catalogue row is greyed and refuses to launch, so
a file that does play it on a core the card does have — another author's, or
one from a database the tracker does not follow — would fall between the two
rules and vanish. Such a file becomes a standin row instead (`Row.Standin`),
listed right under the catalogue row it stands in for, and its Details say why
it is there. `Ingest` records that row as the standin's anchor through the same
coverage `LocalTakeovers` uses (setname, then the parent for a clone), and
every order but Favorites and Recents compares the anchor in its place
(`Dataset.SortRow`), so the standin follows it whatever its own title, maker,
year or missing dates would say; group headers and jumps read the anchor too.
The core index decides it, so the row appears and goes as cores are
installed or removed: the moment the catalogue's own core lands on the card,
the standin gives way to it and hands over its favorite, remembered version
and launch records. `MergeLocal` keeps standin rows although the catalogue
names their setname; a catalogue refresh alone never takes their keys, since
only a card scan can tell whether the new rows are any more runnable here. The copy outside `_alternatives` with the shortest
path is the row's file; further copies are its alternatives. Title, year,
manufacturer, rotation, region, players and buttons come from the header;
the category is shown as a note rather than a genre. Local rows carry no
release dates and never count as new since the last look. A file whose name
or header names a BIOS is skipped. When a catalogue refresh names a local
row's game, the row goes and its favorite, remembered version and launch
records move to the catalogue row's key (`data.LocalTakeovers`,
`App.RenameKeys`).

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
