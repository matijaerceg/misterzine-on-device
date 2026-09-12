# Troubleshooting

## Display setup

MisterZine draws into Linux's framebuffer: a memory area containing the screen's
pixels. Its launcher needs Main's framebuffer console to display those pixels.
`fb_terminal=1` is already the MiSTer default, even if the line is absent from
MiSTer.ini. Only change it if you previously disabled it with `fb_terminal=0`.

Choose the output for Menu and MisterZine in the existing `[Menu]` section of
MiSTer.ini. Save a backup before editing, quit MisterZine, and restart MiSTer
afterward. These settings apply to Menu/script output; game cores use their
own settings. MisterZine does not rewrite MiSTer.ini.

For a normal HDMI display:

```ini
[Menu]
direct_video=0
vga_scaler=0
```

For a 15 kHz CRT on the analog board's VGA/YPbPr port:

```ini
[Menu]
direct_video=1
vga_scaler=0
```

A CRT connected through a supported HDMI-to-VGA direct-video adapter already
receives the direct-video output. Power off before changing between that adapter
and a normal HDMI display. Keep the sync and RGB/YPbPr settings your CRT needs.

### Can HDMI and CRT show MisterZine together?

The current interface cannot provide normal HDMI resolution and native 240p CRT
output simultaneously. Unlike a game core's native picture, Linux's framebuffer
enters through MiSTer's scaler. Enabling `vga_scaler=1` copies that scaler output
to the analog port at the same timing; it does not separately convert 1080p to
240p. See MiSTer's [video settings](https://mister-devel.github.io/MkDocs_MiSTer/advanced/ini/#general-video-settings).

An advanced shared-timing setup may work if **both displays accept the same
CRT-compatible mode**. Ordinary HDMI TVs often do not accept that signal. Do not
enable `vga_scaler=1` with a 720p/1080p mode on a 15 kHz CRT.

Main's CRT message suggesting `fb_terminal=0` or `vga_scaler=1` is generic console
help. Disabling the console prevents MisterZine's launcher from working; choose
the output configuration above instead. For frequent switching, MiSTer supports
[named alternate INI files](https://mister-devel.github.io/MkDocs_MiSTer/advanced/ini/#alternate-mister-ini-naming-option);
MisterZine reads whichever one is active, and its log's `ini:` line names the file
and the `osd_rotate` it found.

Use Options -> Edit safe zone if text reaches outside the visible CRT picture.

The September 7, 2026 MiSTer Linux framebuffer driver lacked the usual mapping
operation. MisterZine includes a fallback for that driver and uses normal mapping
when available. The [kernel fix](https://github.com/MiSTer-devel/Linux-Kernel_MiSTer/commit/ea2212221ad137cf26bf5caa7ad3dab7216435a6)
is also available upstream. Downgrading Main or Linux is not an installation step.

## Installer recognised, but Scripts entries are missing

Check the `SECTION: misterzine` part of Update All's latest log. Recognising the
INI does not mean its files were installed. A global positive filter such as
`filter = arcade console` can exclude the whole app. In `downloader_misterzine.ini`,
add an empty `filter =` below `db_url` in the `[misterzine]` section, then run
Update All again. Current copies of the installer INI include this line.
If you configured MisterZine directly in `downloader.ini`, add the line to its
existing `[misterzine]` section there instead.

After installation, the card should contain `Scripts/MisterZine-Run.sh`,
`Scripts/MisterZine-Setup.sh` and `Scripts/MisterZine-Uninstall.sh`. If they exist on the card, leave and reopen
the Scripts menu. If they do not, keep the log's MisterZine section and any
download errors for troubleshooting. Avoid deleting Downloader's stored state.

Scripts from v1.0.0 and v1.0.1 had spaces in their filenames, which MiSTer's
Scripts launcher does not handle. Updating replaces them with the hyphenated names.

## Main-menu entry missing or not opening

Run **MisterZine-Setup** in Scripts. It creates the menu entry and enables the
startup helper. Setup is repeatable. If files are missing, run Downloader first.
After a successful setup, return to the main menu and select MisterZine.

An external updater can temporarily prevent the launcher from opening. Let it
finish before trying again. Errors from the app wrapper show recent log lines
before returning; the logs are `misterzine/log.txt` and `misterzine/watch.log`.

### Degauss opens instead

Degauss installs itself as MiSTer's `main=` frontend and takes over every load
of the menu core, and MisterZine's menu entry loads the menu core. Choosing
MisterZine therefore brought up Degauss, and the launcher then waited for a
console it could never get until the next reboot. The launcher now closes
Degauss for the MisterZine session (`watch.log` says "Degauss has taken this
menu load") and the usual menu restore brings Degauss back when MisterZine
exits. Return after game and Open at boot work the same way on such a card.
Degauss does not list MisterZine's menu entry, so its Scripts list carries
**MisterZine-Run** instead: it opens MisterZine directly and Degauss returns
when MisterZine quits (through Options -> Quit MisterZine or the pad's Menu
button; keyboard F12 does not leave from a Scripts session).

## Favorites unreadable

The app keeps the original favorites file on read errors and disables favorite
edits for that session. Restore or repair `misterzine/favorites.json`, then
reopen the app. Do not delete the file as a generic troubleshooting step.

## A game or an alternative is missing

Use Rescan card after an external update. Alternatives must be installed:
Downloader filters can intentionally exclude them. MisterZine leaves those
preferences unchanged. A missing or invalid launch target reports an error.

An alternative whose MRA file holds no readable header (an empty file, one
cut short, or one without an `<rbf>` element) is left out of the picker, as
MiSTer could not load it either. The scan still completes. The log names
each such file the first time its folder is read (`scan: skipped ...`) and
every scan repeats the count (`N unreadable MRAs skipped`). A "Card scan
incomplete" result means a folder or file could not be read at all; the log
line above it names the path.

## Start does not launch games

MisterZine uses the Start button saved through MiSTer's main-menu **Define
joystick buttons** for ordinary global controller mappings. Reopen MisterZine
after changing that assignment. Without a saved mapping, standard Linux Start
remains the default. The app does not modify your MiSTer configuration.

Open **Options -> Troubleshooting -> Test Start button**. Release all buttons
during the countdown. When **GO** appears, press and release your Start button
once or twice. The test ends automatically after six seconds and freezes a
result for a photo. Games cannot launch from button presses during this test.

Send a photo of the first result page to the person helping you. L/R or the
arrows show additional pages, including full controller names, device IDs,
button codes, press/release counts, and whether the normal input reader uses
each device. Connect the controller before starting; rerun the test after
changing connections. MiSTer's Menu button still closes the app.

**Button received: YES / App saw Start: NO** suggests an input compatibility
problem. The device pages show the selected Start button and mapping source.
Raw button numbers stay the same after remapping; the app uses the saved
assignment to interpret them. Per-device hashed or alternate-mode maps and
Start assignments represented as keyboard keys or axes are not imported by
this release's direct Start reader.
**No button received** means further device investigation is needed, not that
the controller has necessarily failed. The test also observes devices that
the normal reader skips.

To check launching separately, highlight an installed game in the list, then
open **Options -> Troubleshooting -> Test game launch**. Press **A / Enter**
again to launch its main version. This uses the normal launch path without
requiring a Start button. Tell the person helping you whether the game actually
started; a saved "Command sent" result cannot establish that by itself.

**Last troubleshooting result** remains available after closing or restarting
the app. Results do not dim while being viewed. A controller test replaces the
previous report; a subsequent launch test adds its result. The report is kept
locally in `misterzine/troubleshooting.json`; sharing a photo is sufficient.
No extra tools, file uploads, account or remote-debug setup are needed.

Additional raw input is observed only during an explicit controller test. Troubleshooting
does not change mappings, enable the remote-debug server, or upload information.

## No connection or an old catalogue

Cached data remains usable. Check the connection, then use Refresh data now.
Pictures load as you browse and failed downloads retry with increasing delays.
The bundled catalogue is a starting point for a first launch without a connection.

## Reporting a problem

Include the app version and data date from Options, your board, display
connection, rotation, and the steps that reproduce it. A photo helps for physical
display problems; F12 captures what the app draws. Remote debugging is optional
and described in the development guide.

For Start/controller problems, begin with the built-in test above and a photo
of its result. For launch failures, also give the game title and the outcome of
**Test game launch**.
