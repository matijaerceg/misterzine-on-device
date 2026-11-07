# Neo Geo arcade catalogue

The app, separate device test and diagnostic package share `internal/mgl/neogeo-arcade.json`: 161 game families with 1,282 identifiers, titles and aliases before case folding. This replaces the original five-game pilot list. Classic arcade releases, arcade prototypes and known regional, revision and bootleg variants are included. Specific later MVS releases are also covered; this is not a claim to recognise every future homebrew filename.

## Sources and regeneration

Run `python3 tools/generate_neogeo_catalogue.py --site /path/to/misterzine` with the [website checkout used for this edition](https://github.com/matijaerceg/misterzine/tree/941fa60c245b773d503cd25331632c0c4b21b86c). A sibling checkout named `misterzine` is the default. For offline regeneration, add `--sources DIR` with the three identity files below named `neogeo.cpp`, `romsets.xml` and `software.xml`, plus the two versioned data archives below saved as `nplayers.zip` and `command.zip`. Online generation uses curl. Both modes verify the archives against their SHA-256 hashes.

- [MAME Neo Geo driver](https://github.com/mamedev/mame/blob/114a5f239188be9cbf965cf0d0085dd7167e6ed0/src/mame/snk/neogeo.cpp): arcade game identities and clone relationships.
- [MiSTer Neo Geo set map](https://github.com/MiSTer-devel/NeoGeo_MiSTer/blob/a8496bfe0203c69f13b6c646c4bbf6a95f231482/releases/romsets.xml): Darksoft/MiSTer set aliases and English/Japanese display names.
- [MAME Neo Geo software list](https://github.com/mamedev/mame/blob/954def46685cd0276671138fbd032036b1a771fb/hash/neogeo.xml): additional entries explicitly recording an MVS release, including Xeno Crisis.
- Additional MVS identities: Knight's Chance (MVS/AES noted in the MAME driver), [GladMort](https://www.pixelheart.eu/fr/produit/gladmort-neo-geo-mvs-us-2/), [Cyborg Force](https://ozzyouzo.itch.io/cyborg-force) and [Captain Barrel](https://ozzyouzo.itch.io/captain-barrel/comments). MiSTer's map supplies their exact set identifiers.

## Arcade game details

Both discovery methods attach facts to the recognised game family. Old saved rows pick up the current facts on startup. The website's MAME data supplies year, manufacturer, orientation and input device. Its genre and mature-content rules also apply here. Original MVS video uses the 15 kHz CRT class (a 6 MHz pixel clock and 384 clocks per scanline). [Arcade data](https://github.com/matijaerceg/misterzine/tree/941fa60c245b773d503cd25331632c0c4b21b86c/data/cache) · [MAME clock](https://github.com/mamedev/mame/blob/114a5f239188be9cbf965cf0d0085dd7167e6ed0/src/mame/snk/neogeo.h) · [MAME scanline timing](https://github.com/mamedev/mame/blob/114a5f239188be9cbf965cf0d0085dd7167e6ed0/src/mame/snk/neogeo_spr.h)

Player counts and alternating or simultaneous play come from NPlayers 0.278 by Nomax, © 2004–2025 Made Of Dreams Corp. The app expands its play-mode abbreviations. This subset is available under [CC BY-SA 3.0](https://creativecommons.org/licenses/by-sa/3.0/). [NPlayers](https://nplayers.arcadebelgium.be/) · [Source archive](https://nplayers.arcadebelgium.be/files/nplayers0278.zip)

Action-button counts use game-specific assignments. Progetto-SNAPS Command 0.273 documents all four button positions for 62 game families, including unused positions. For example, Metal Slug assigns three actions and Blazing Star assigns two. V-Liner's MAME driver names its four game buttons separately from the operator inputs. Other titles with a documented count include GladMort and Xeno Crisis. A blank count means the available evidence covers only the input device. [Command project](https://www.progettosnaps.net/command/) · [Source archive](https://www.progettosnaps.net/download/?tipo=command&file=pS_Command_273.zip) · [V-Liner inputs](https://github.com/mamedev/mame/blob/114a5f239188be9cbf965cf0d0085dd7167e6ed0/src/mame/snk/neogeo.cpp)

The five games outside the main arcade data use the following MVS records. The [FinalBurn Neo cartridge records](https://github.com/finalburnneo/FBNeo/blob/0c65008a226434235b2a4b01e2036013d8fd4cd9/src/burn/drv/neogeo/d_neogeo.cpp) also identify their player counts and joystick hardware.

| Game | MVS year | Maker | Genre | Players | Sources |
| --- | --- | --- | --- | --- | --- |
| Knight's Chance | 2014 | Neobitz | Tabletop & quiz | 1 | [Developer](https://neobitz.itch.io/knights-chance) · [MVS developer interview](https://www.archaic.fr/interviews/interview-jeff-kurtz-repond-a-nos-questions/3/) |
| Xeno Crisis | 2021 | Bitmap Bureau | Arena shooter | 2 simultaneous | [Publisher](https://eushop.bitmapbureau.com/products/xeno-crisis-neo-geo-mvs) · [MVS release dates](https://github.com/mamedev/mame/blob/954def46685cd0276671138fbd032036b1a771fb/hash/neogeo.xml) · [Neo Geo control screen](https://drinkwildsoda.wordpress.com/2021/10/07/624/) |
| Cyborg Force | 2024 | Neo Byte Force | Run 'n gun | 2 simultaneous | [Developer](https://ozzyouzo.itch.io/cyborg-force) · [Developer's March 2024 shipping announcement following 2023 preorders](https://www.arcade-projects.com/threads/cyborg-force-new-game.25554/page-3) |
| Captain Barrel! | 2024 | Neo Byte Force | Puzzle | 2 alternating | [Developer's Neo Geo release](https://ozzyouzo.itch.io/captain-barrel) · [Developer's play-mode description](https://store.steampowered.com/app/3041340/CAPTAIN_BARREL/) |
| GladMort | 2025 | PixelHeart / Chipsonsteroids | Platformer | 1 | [MVS release and studio credits](https://www.pixelheart.eu/en/produit/gladmort-neo-geo-mvs-us/) · [Publisher's control guide](https://www.pixelheart.eu/en/become-a-gladmort-master-the-ultimate-guide/) |

## Screenshots

The app requests pictures from the MisterZine image service using the recognised family name. All 161 families have a gameplay capture, and 160 also have a title capture. GladMort uses gameplay alone because the available title capture is marked Demo. The picture viewer, list thumbnail preference, launch cabinet and screenshot screensaver use the same card cache as other arcade games.

The image service already held gameplay and title screens for 156 families from its arcade screenshot collection. The additional Neo Geo captures for Captain Barrel!, Cyborg Force, Knight's Chance, Xeno Crisis and GladMort come from the FBNeo arcade thumbnails. The source filenames are `Captain Barrel (HB)`, `Cyborg Force (HB)`, `Knight's Chance (HB)`, `Xeno Crisis (HB)` and `GladMort (HB)`. These five families use the service’s `fbneo_<setname>` keys so routine MAME pack updates preserve the reviewed captures. Each added object records its exact source URL. [MisterZine image service](https://github.com/matijaerceg/misterzine/tree/main/images) · [FBNeo gameplay captures](https://github.com/libretro-thumbnails/FBNeo_-_Arcade_Games/tree/550049b1bb175edbb858bdd533cfc03f684ef680/Named_Snaps) · [FBNeo title captures](https://github.com/libretro-thumbnails/FBNeo_-_Arcade_Games/tree/550049b1bb175edbb858bdd533cfc03f684ef680/Named_Titles)

## Matching and boundaries

The catalogue preserves the matched set identifier within each family. MAME descriptions and MiSTer's set aliases supply picker labels. A readable filename identifies a particular revision when its spelling is unique to that set. The list groups matching card rows in every browsing view while keeping their original paths, shortcut configurations and per-core availability. The original set leads the version picker when present. A named-version search selects the matching installed target, and the family remembers an explicit version choice by path.

Matching uses the target's basename. It recognises exact case-insensitive set names, source titles, regional titles and explicit filename-safe variants. For `.neo`, `.rom` and `.zip` filenames, an exact match takes priority. A filename such as `3 Count Bout (3countb).neo` can also match a known set identifier in the final parentheses. This includes MAME clones and MiSTer set names. Known variants share their family's title and classification. Each distinct shortcut is selectable in Details and launches with its original configuration. Existing pilot IDs and titles are retained to preserve favourites.

The MAME-only exclusions are `lasthope` (home-only conversion without credits), `mvstemp` (development subsystem) and the CD-only conversions `zintrckb`, `crswd2bl`, `froman2b`. Unmapped MiSTer entries such as test ROMs, demos and home-only games remain diagnostic candidates. Other cores and CD/mounted or multiple-file MGLs are rejected independently of game-name matching. Neo Geo home versions of arcade games share the family, so a recognised shortcut is not a guarantee of MVS mode.

Coverage is tested by scanning a synthetic shortcut for every family. Alias, exclusion, saved-identity and host launch-validation tests supplement that coverage. Hardware testing is reported separately from recognition support.
