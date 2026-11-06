# Neo Geo arcade catalogue

The app, separate device test and diagnostic package share `internal/mgl/neogeo-arcade.json`: 161 game families with 1,282 identifiers, titles and aliases before case folding. This replaces the original five-game pilot list. Classic arcade releases, arcade prototypes and known regional, revision and bootleg variants are included. Specific later MVS releases are also covered; this is not a claim to recognise every future homebrew filename.

## Sources and regeneration

Run `python3 tools/generate_neogeo_catalogue.py` to regenerate from pinned metadata, or use `--sources DIR` containing `neogeo.cpp`, `romsets.xml` and `software.xml` for an offline run. The generator downloads metadata only, never games. Names and parent relationships are factual catalogue data; no emulator code or ROM data is embedded.

- [MAME Neo Geo driver](https://github.com/mamedev/mame/blob/114a5f239188be9cbf965cf0d0085dd7167e6ed0/src/mame/snk/neogeo.cpp): arcade game identities and clone relationships.
- [MiSTer Neo Geo set map](https://github.com/MiSTer-devel/NeoGeo_MiSTer/blob/a8496bfe0203c69f13b6c646c4bbf6a95f231482/releases/romsets.xml): Darksoft/MiSTer set aliases and English/Japanese display names.
- [MAME Neo Geo software list](https://github.com/mamedev/mame/blob/954def46685cd0276671138fbd032036b1a771fb/hash/neogeo.xml): additional entries explicitly recording an MVS release, including Xeno Crisis.
- Additional MVS identities: Knight's Chance (MVS/AES noted in the MAME driver), [GladMort](https://www.pixelheart.eu/fr/produit/gladmort-neo-geo-mvs-us-2/), [Cyborg Force](https://ozzyouzo.itch.io/cyborg-force) and [Captain Barrel](https://ozzyouzo.itch.io/captain-barrel/comments). MiSTer's map supplies their exact set identifiers.

## Matching and boundaries

Matching uses the target's basename, not the MGL's label or setname. It recognises exact case-insensitive set names, source titles, regional titles and explicit filename-safe variants. For `.neo`, `.rom` and `.zip` filenames, an exact match takes priority. A filename such as `3 Count Bout (3countb).neo` can also match a known set identifier in the final parentheses. This includes MAME clones and MiSTer set names. MAME clones and known MiSTer variants map to their game family for title and classification; the original shortcut and configuration remain unchanged and distinct XML still produces distinct rows. Existing pilot IDs and titles are retained to preserve favourites.

The MAME-only exclusions are `lasthope` (home-only conversion without credits), `mvstemp` (development subsystem) and the CD-only conversions `zintrckb`, `crswd2bl`, `froman2b`. Unmapped MiSTer entries such as test ROMs, demos and home-only games remain diagnostic candidates. Other cores and CD/mounted or multiple-file MGLs are rejected independently of game-name matching. Neo Geo home versions of arcade games share the family, so a recognised shortcut is not a guarantee of MVS mode.

Coverage is tested by scanning a synthetic shortcut for every family. Alias, exclusion, saved-identity and host launch-validation tests supplement that coverage. Hardware testing is reported separately from recognition support.
