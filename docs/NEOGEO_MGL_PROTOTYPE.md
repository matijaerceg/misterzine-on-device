# Neo Geo MGL import prototype

Normal builds do not enable this feature. This document describes the desktop/card-audit prototype; the separate opt-in ARM device build is documented in [Neo Geo device test](NEOGEO_DEVICE_TEST.md). It discovers existing shortcuts, explains inclusion/exclusion, and exports recognised games to the existing UI harness. It does not write to the card, install anything, generate shortcuts, or launch a game on hardware.

## Try the preview

With Go 1.27 and Python 3 available, run `python3 tools/preview_mgl.py` from the repository. It scans the fixture card, writes `dist/mgl-prototype/report.json` and `rows.json`, and renders horizontal and vertical list/details previews. The harness prints the original MGL path when its scripted Start button is pressed; this is a simulated launch callback, not hardware validation.

The example includes five deliberately reviewed arcade identities: Metal Slug, Art of Fighting, Blazing Star, Samurai Shodown and The King of Fighters '98. This is a pilot matching list, not a complete Neo Geo catalogue. The synthetic card contains no ROMs, BIOS files or FPGA binaries. Installed status remains unknown in the preview.

## Inspect a real card or a copy of its menus

```sh
go run ./cmd/mzmglprobe -card /path/to/card
go run ./cmd/mzmglprobe -card /path/to/card -arcade-list testdata/mgl/arcade-pilot.json
go run ./cmd/mzmglprobe -card /path/to/card -arcade-list testdata/mgl/arcade-pilot.json -rows > /tmp/neogeo-rows.json
```

The first command reports eligible cartridge shortcuts as candidates. Supplying a reviewed list allows recognised arcade identities into the exported rows. Each identity has `id`, `title` and an `aliases` array of exact target basenames without extensions. Matching ignores case, but does not guess at abbreviated titles or strip regions/revisions. Contradictory aliases fail loading rather than silently choosing a game. Unknown targets remain visible in the report without entering the arcade list.

## XML rules

- Read one `mistergamedescription` root, exactly one `rbf`, at most one `setname`, and all direct `file` children.
- Accept exact core basenames NeoGeo, NeoGeo_LLAPI and NeoGeoYC, optionally dated or ending in `.rbf`. Those alternate names come from Favorites; they have not been hardware-tested here. Other variants, including rmNeoGeo, need an explicit follow-up decision.
- Require exactly one `file` with `type="f"` and `index="1"`. Reject mounted/CD media, other slots, core-only and multi-file shortcuts. Recognise `.neo`, whole-set `.zip`, generator-style `.rom` and extensionless extracted-set targets.
- Preserve the target string, delay and `setname`/`same_dir`. Paths may be relative to the core's games folder, contain the generator's `../../../../../media/...` prefix, or point inside an archive. No host filesystem path is constructed from a ROM target.
- Match arcade identities from the target's final basename, never the MGL display filename or `setname`. A setname can select an arbitrary configuration and is not an arcade certification. A matching filename also does not prove the file contains the expected ROM.
- Keep the original MGL as the launch target. Reset/options and per-game settings are not reconstructed or removed. Only byte-identical shortcuts coalesce; their other paths are reported as alternatives. Different configurations or differently formatted XML remain separate entries in this prototype. Renaming/moving an unchanged shortcut retains its identity; editing its contents changes the provisional identity.

The XML can establish Neo Geo cartridge loading, but MVS/AES use the same slot and it cannot establish a game's historical arcade release. The separate reviewed identity list enforces that boundary. The preview's existing `mra` field carries the MGL path only as a launch target; it must not be fed into MRA validation when wiring the device host.

## Discovery and limits

Scan root-level MGLs and underscore-prefixed menu folders, including `_@Favorites`, `_Games`, `_Arcade` and `_alternatives`. Nested folders must also start with an underscore, matching the menu convention. Do not crawl ordinary `games/` trees or follow file/directory symlinks. Linked shortcuts consequently remain unsupported. The scanner reads at most 20,000 MGLs, visits at most 100,000 entries, descends eight menu levels and limits each MGL to 64 KiB. Unreadable folders and scan limits appear as problems; incomplete scans cannot export a replacement row list.

Legacy generators sometimes leave raw ampersands in filenames. Parsing repairs only bare ampersands in memory and records that compatibility path; it never rewrites the original. Other malformed XML is reported. XML directives, multiple roots and duplicate core/setname elements are rejected. Unexpected extension elements are skipped for classification but remain in the original launcher and its identity hash.

## Source observations and fixtures

The fixtures are small, independently authored examples of the following generator output conventions, inspected on 2026-11-02. They are not files captured from a user's card or proof that every current generator output has been tested.

- [Favorites launcher](https://github.com/wizzomafizzo/mrext/blob/fe4536d31cbbc5f55fece5c1aaae3611cddd1fad/pkg/favorites/launcher.go): relative Neo Geo game paths; ZIP support uses the cartridge slot; LLAPI and YC core variants.
- [GamesMenu Neo Geo handling](https://github.com/wizzomafizzo/mrext/blob/fe4536d31cbbc5f55fece5c1aaae3611cddd1fad/pkg/gamesmenu/neogeo.go): mapped ZIP/directory sets and a root-climbing path prefix. Its `romsets.xml` is a naming/loading map, not itself the reviewed arcade-only list.
- [Shared MGL generator](https://github.com/ZaparooProject/zaparoo-core/blob/1e854eba8603fb2a6a1d1790520e5ba393b2454b/mister/mgl/mgl.go): XML escaping, optional setname, file slot/delay, reset and options, including archive-member paths.
- [NeoGeo-Arcade generator](https://github.com/TheJesusFish/NeoGeo-Arcade/blob/9a709463957b1c0662ee7747af5854f1d7e9a941/neogeo_arcade_set.py): relative paths, `setname same_dir="1"`, and separate per-game configuration files. Those configurations must remain installed; an MGL alone does not carry them.
- [Official MGL format](https://mister-devel.github.io/MkDocs_MiSTer/advanced/mgl/): menu folder conventions and Neo Geo cartridge loading parameters.

## Remaining integration work

Expand and review the identity catalogue and aliases against real collections. Define stable game/configuration identities beyond the prototype's exact-file hash. The tagged device build now connects discovery to rescans and the local cache, checks the referenced core directory separately from arcade classification, and leaves ROM validity unknown. Storage-target and BIOS validation remain open. Exercise Favorites/Recents and configuration variants across restarts. Finally, test launch and return on hardware using working original shortcuts. These steps are not claimed by the desktop prototype.

## Prototype validation

Linux `go test ./...` and `go vet ./...` passed; changed packages were rerun after the Details wording and parser checks changed. The parser also passed a short fuzz run. The probe cross-builds as a static Linux ARMv7 executable. All 1,220 existing UI golden renders and 250 controller golden renders match unchanged. The four new prototype previews were inspected separately. Fixture discovery reports five included games, three excluded shortcuts and one unclassified cartridge candidate. UI integration tests verify that Start passes each of the five original MGL paths to the launch callback. No hardware launch, ROM validation, installation or release is claimed.
