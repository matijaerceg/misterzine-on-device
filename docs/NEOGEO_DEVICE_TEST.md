# Neo Geo MGL device test

Neo Geo shortcut support is now included in normal MisterZine builds; see the [user guide](USER_GUIDE.md#neo-geo-games). This page documents the earlier separate test installation, which remains isolated from the normal app and keeps its own preferences.

This experimental build recognises the Neo Geo arcade catalogue, including known regional and revision set names. It uses one shared list of 161 game families for discovery and launch checks; the old five-game restriction is gone. See [catalogue sources and coverage](NEOGEO_CATALOGUE.md). Recognition does not certify ROM contents, BIOS compatibility or MVS settings.

## Install

Download `MisterZine-NeoGeo-Install-test.5.sh`, copy it into the card's `Scripts` folder, and run it from MiSTer's Scripts menu with an internet connection. Then open **MisterZine-NeoGeo-TEST**. Alternatively, extract the test ZIP at the card root.

Exit the test app and its game before installing. The installer verifies the pinned ZIP, preserves existing test preferences and saves a backup of a previous test installation. It uses Downloader's maintained certificate bundle when available. Normal MisterZine, boot settings and Update All configuration stay unchanged. This trial does not update itself automatically.

## Shortcuts and controls

Use existing working `.mgl` shortcuts in the card root or underscore-prefixed menu folders at every level. The scan does not walk `games/`. The shortcut must name a supported Neo Geo core and exactly one cartridge file with `type="f" index="1"`. Other cores, CD targets, demos and unknown names are not imported. The original MGL is launched unchanged, including its paths and configuration options. No cores, games or BIOS files are included.

Configure **Options > Return after game** and **Exit game with** as desired. The temporary return helper runs only for this test app's games. To hide the imports, turn off **Filters > Source/core > Local > Neo Geo**. Settings and logs live in `/media/fat/misterzine-neogeo-test`.

## Browsing and filters

**Development after test 5:** **Choose view > Core** groups recognised Neo Geo shortcuts under one **Neo Geo** heading alongside the other arcade cores, with games A-Z. The heading is kept even with one shortcut, and supported core variants share it. The Neo Geo filter still applies.

Enable **Neo Geo** under **Options > Available views > Cores** (or **Y > Edit available views**) for an A-Z list of your recognised shortcuts. It starts off and can also be chosen as the default view once enabled. It respects your filters; if Neo Geo is hidden, the empty list says so, with Filters available in the footer.

**Filters > Source/core** lists Local first, then catalogue sources by their displayed names. Neo Geo is inside Local. Unknown values appear last in filter sections; years and player counts keep their natural order.

## Feedback

Try any recognised Neo Geo arcade game you already have. Check launch, controls, exit chord, favourites and the Neo Geo filter. Metal Slug and Art of Fighting were tested on a DE10 before the catalogue expansion; recognising additional games does not mean each has been hardware-tested.

Missing games are explained in `mgl-report.json` in the test folder. Review its local paths before sharing. The standalone read-only diagnostic is `mzmglprobe -card /media/fat -arcade-list /media/fat/misterzine-neogeo-test/neogeo-arcade.json`. For requested remote troubleshooting only, `debug.flag` in the test folder enables the authenticated debug API on port 8197.

## Remove or build

Exit the test app, then remove `Scripts/MisterZine-NeoGeo-TEST.sh`, the installer and `misterzine-neogeo-test`. Any `misterzine-neogeo-test-backup-*` folders can be removed when no longer needed. Keep your games, shortcuts and usual MisterZine installation.

Build `cmd/misterzine` for Linux ARMv7 with `CGO_ENABLED=0`, the `neogeopilot` tag and a distinct version stamp. Build `cmd/mzmglprobe` for the same target, then run `python3 tools/package_neogeo.py --binary PATH --probe PATH --output PATH`. Normal builds omit the tag and import the same recognised MGL rows through their ordinary card scan and launcher. The tag now selects only the isolated test installation and its restrictions.
