# Troubleshooting

## Display setup

Set up your connection using the [display setup guide](DISPLAY_SETUP.md).
This is part of installation, even if games already display correctly.

If MiSTer shows a message asking you to modify MiSTer.ini when you open
MisterZine, follow the guide for your connection. Leave `fb_terminal` enabled:
MisterZine needs it to display its interface.

### JAMMA cabinets (MiSTercade)

See [JAMMA cabinet display setup](DISPLAY_SETUP.md#jamma-cabinets-mistercade).

### HDMI for the menu, CRT for MisterZine

See [separate HDMI and CRT setup](DISPLAY_SETUP.md#hdmi-for-the-menu-crt-for-misterzine).

### S-Video and composite (Y/C) output

See [Y/C display setup and its colour limitation](DISPLAY_SETUP.md#s-video-and-composite-yc-output).

### Can HDMI and CRT show MisterZine together?

See [simultaneous output and its limitations](DISPLAY_SETUP.md#can-hdmi-and-crt-show-misterzine-together).

### Older Linux framebuffer driver

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
startup helper. Downloader alone installs the files and no menu entry: a card
that shows none has not run Setup yet, whichever installer put the files there.
Setup is repeatable. If files are missing, run Downloader first. After a
successful setup, return to the main menu and select MisterZine Arcade.

An external updater can temporarily prevent the launcher from opening. Let it
finish before trying again. Errors from the app wrapper show recent log lines
before returning; the logs are `misterzine/log.txt` and `misterzine/watch.log`.

### The menu entry is now MisterZine Arcade

The entry was called MisterZine (`MisterZine.mgl`) in earlier versions. After
the update, the launcher writes `/media/fat/MisterZine Arcade.mgl` and removes
the old file when it restarts: within seconds of the update finishing with
MisterZine closed, or at the next boot. Nothing needs doing. If the menu
still shows MisterZine, open another folder and come back, or reboot. Your
settings, favorites and the `[MisterZine]` INI section stay as they were.

Anything outside MisterZine that names the old file stops working: an NFC or
Zaparoo tag, a `bootcore` line in MiSTer.ini, a favorite or shortcut in
another frontend, or a script of your own. Point it at
`/media/fat/MisterZine Arcade.mgl` instead; the space is part of the name.

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
every scan repeats the count (`N unreadable MRAs skipped`).

A file or folder that its folder lists but that cannot be opened is left out
the same way, and the scan still completes. It was deleted while the scan
ran, is a broken link, or is a damaged entry on the SD card. Every scan logs
it (`scan: skipped ...: listed in its folder, but the path no longer
resolves`), and a card report lists it under CARD SCAN. If the same entry
keeps coming back after a rescan, the card's file system probably needs
repair: a disk check on a PC (Windows: the drive's Properties -> Tools ->
Check) may clear it. Back up the card first.

A "Card scan incomplete" result means a folder or file could not be read,
for example a read error on the card; the log line above it names the path.

When a game you have on the card does not show up and none of this explains
it, use **Send a report** (below): it lists every game file MisterZine leaves
out, and why.

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

If every arcade game starts to a black screen, look for a folder named `mame`
at the top of the SD card (`/media/fat/mame`). MiSTer uses it in place of
`games/mame` and then finds no ROM zip at all, even the ones inside it.
MisterZine says so in Details and at Start. Move the folder's contents into
`games/mame` and delete the empty folder.

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

### Send a report

Options -> Troubleshooting -> **Send a report** describes your card to the
developer and gives you a short code, such as `K7M4`, to post wherever
you asked for help. The report holds:

- the app version, your board, the display settings MisterZine reads from
  MiSTer.ini, and the screen rotation
- your settings, the list's order and the filters in force
- every local game (a game read from your card's own MRA files), and which
  filter hides it, if one does
- every game file that is not in the list, with the reason: a file without
  a set name, a folder MisterZine does not read, a game whose core is not in
  `_Arcade/cores`, and so on
- the card's folders: the top level, the `_Arcade` tree with its MRA
  counts, every file in `_Arcade/cores`, the other core folders' counts, the
  arcade ROM folder MiSTer uses with its zip count, and MisterZine's own folder
- how the MisterZine Arcade menu entry is set up: whether the entry, its
  startup line and the helper that opens MisterZine are in place, and which
  program MiSTer.ini names as its `main=`, if any (Degauss and Console Mode
  replace MiSTer's own)
- the last few hundred lines of MisterZine's log, and the helper's latest
  lines (`misterzine/watch.log`)

It holds no passwords, Wi-Fi details or Downloader database addresses, and
the log lines lose any web address's query part. The screen says what goes
into it before anything is sent, and nothing is sent until you press A.

A copy is always saved on the card as `misterzine/report.txt`. If the MiSTer
is offline, or the report service does not take it, the screen says so and you
can send that file instead. Reports go to `api.misterzine.fyi`, which keeps
nothing about who sent them, not even the IP address. Only the developer can
read them, using the code you post, and they are deleted 30 days after they
arrive.

### Other details

Include the app version and data date from Options, your board, display
connection, rotation, and the steps that reproduce it. A photo helps for physical
display problems; F12 captures what the app draws. Remote debugging is optional
and described in the development guide.

For Start/controller problems, begin with the built-in test above and a photo
of its result. For launch failures, also give the game title and the outcome of
**Test game launch**.
