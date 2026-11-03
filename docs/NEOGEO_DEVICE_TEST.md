# Neo Geo MGL device test

This experimental build adds existing Neo Geo cartridge shortcuts to MisterZine's local arcade list. The pilot recognizes Metal Slug, Art of Fighting, Blazing Star, Samurai Shodown and The King of Fighters '98 by the target filename. It excludes other cores, CD games and unrecognized games. A name match does not authenticate a ROM or prove that its settings select MVS mode.

## Install and run

Download `MisterZine-NeoGeo-Install-test.3.sh` from the test release, copy it into the card's `Scripts` folder, and run it from MiSTer's Scripts menu with an internet connection. It downloads exactly test 3, verifies the ZIP's SHA-256 checksum, and adds **MisterZine-NeoGeo-TEST**. Exit the test app and its game before rerunning the installer. Rerunning repairs the test files while preserving test preferences and saving the previous test folder as a backup. It does not register an Update All feed. Future test versions require their own installer.

Alternatively, extract the ZIP at the root of the MiSTer card. In MiSTer's Scripts menu, choose **MisterZine-NeoGeo-TEST**. The app uses `/media/fat/misterzine-neogeo-test` for its settings, cache and logs. It has separate preferences; configure rotation and controls there if needed. Your usual MisterZine installation stays in place.

Existing `.mgl` files must be in the card root or menu folders whose names start with `_` at every level. The scan does not walk `games/`. The shortcut must name a supported Neo Geo cartridge core and exactly one file with `type="f" index="1"`. Its core must exist in the directory named by `<rbf>`. The original MGL is passed to MiSTer unchanged, preserving its game path, delay, setname and options. ROM, archive-member and BIOS validation is not implemented.

This build does not install a resident launcher or change boot settings. Options Ã¢â€ â€™ Return after game and Exit game with work through a temporary helper for games launched from this test app. Enable Return after game and choose Select+Start to hold those buttons for one second and reopen the test app on the game. The helper exits when the game is replaced or the returned app closes. The application updater is unavailable. No core, ROM or BIOS is included.

To hide imported Neo Geo games, open Filters Ã¢â€ â€™ Source/core Ã¢â€ â€™ Local Ã¢â€ â€™ Neo Geo and turn it off. This saved choice covers all supported Neo Geo core variants and games discovered later; other local arcade games remain available. Turn it back on to restore them.

## Smoke test

1. Prepare one working shortcut for one of the five games. Launch it directly from MiSTer's menu first to confirm the game and BIOS work.
2. Open the test app, rescan, and find the game among local arcade entries. Check that a shortcut for another core or a CD image does not appear as a Neo Geo arcade game.
3. Open the game details and launch. Verify the correct game, expected settings and controller input on the display.
4. Hold the configured exit chord for one second and confirm that Return after game reopens the test app on the same game. Launch again, then exit through MiSTer's menu and check the same return. Confirm favorites and the Neo Geo filter survive reopening and rescanning.
5. Inspect `misterzine-neogeo-test/mgl-report.json` for included, excluded, unrecognized or malformed shortcuts. The report contains local paths, so review it before sharing.

The standalone diagnostic command is `mzmglprobe -card /media/fat -arcade-list /media/fat/misterzine-neogeo-test/arcade-pilot.json`. It reads files without launching games. For remote troubleshooting only, create `misterzine-neogeo-test/debug.flag`; the test wrapper then enables the existing authenticated debug API on port 8197.

## Remove

Exit the test app, then delete `Scripts/MisterZine-NeoGeo-TEST.sh`, the downloaded installer and the `misterzine-neogeo-test` folder. Any `misterzine-neogeo-test-backup-*` folders created by repeat installs can also be removed when no longer needed. This removes only the test build and its settings. Keep your game shortcuts and your usual MisterZine installation.

## Build

Build `cmd/misterzine` for Linux ARMv7 with `CGO_ENABLED=0`, the `neogeopilot` build tag and a distinct `internal/buildinfo.Version` stamp. Build `cmd/mzmglprobe` for the same target. Run `python3 tools/package_neogeo.py --binary PATH_TO_TEST_BINARY --probe PATH_TO_PROBE --output dist/misterzine-neogeo-test.zip`. Normal builds omit the tag and do not import MGL rows.
