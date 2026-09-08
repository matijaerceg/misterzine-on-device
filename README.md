# misterzine on your MiSTer

The [misterzine](https://misterzine.fyi) release tracker, running on the MiSTer
itself. It answers three questions from the Scripts menu, on a 240p CRT in
horizontal or tate orientation, with a gamepad:

- what shipped or got rebuilt, newest first (the same rows and order as the site),
- is it on my card, and is my copy the current build,
- launch it.

It is not a launcher for your whole card. Degauss, Zaparoo Frontend and Console
Mode do that. This is the "what's new" view, with screenshots and specs, and a
way to start the game you are looking at.

## Install

Add the database to `/media/fat/downloader.ini` on the card:

```ini
[misterzine]
db_url = https://github.com/matijaerceg/misterzine-on-device/releases/latest/download/misterzine.json.zip
```

(or copy the release asset `downloader_misterzine.ini` next to downloader.ini,
it is the same two lines). Run update_all (or downloader) once. You get
`Scripts/misterzine.sh` and the binary under `/media/fat/misterzine/`. Then:
main menu, Scripts, misterzine.

`fb_terminal=1` must be set in MiSTer.ini (it is the default).

The September 7, 2026 MiSTer Linux release has a framebuffer driver bug
that prevents apps such as MisterZine from opening their display. Current
MisterZine source includes a compatibility fallback for that release. It
uses the usual framebuffer interface whenever the kernel supports it;
Main, Menu and Linux do not need to be downgraded. MiSTer has also
[fixed the driver in its kernel source](https://github.com/MiSTer-devel/Linux-Kernel_MiSTer/commit/ea2212221ad137cf26bf5caa7ad3dab7216435a6).

### Running Update All

For the usual Scripts-menu or Remote launch, quit MisterZine before
starting Update All, let the updater finish (including any requested
reboot), then reopen MisterZine. Both apps use the same script console;
hiding Update All does not turn it into an independent background job.

An updater started independently through SSH with its own input/output
can keep working while MisterZine is open. Quitting MisterZine does not
pause that process. Updates can replace cores or restart the system, so
wait for completion before launching a core, and use Settings → Rescan
to refresh the on-card status if MisterZine stayed open. This is not a
promise that every Update All configuration supports concurrent use.

### Put it in the main menu

Once, from the app: X, Settings, "Main menu launcher", A. From then on a
"MisterZine" entry sits in the main menu next to Arcade, Console and the
rest, and picking it opens the app; no Scripts menu.

How it works: the database ships `MisterZine.mgl` at the card root, an MGL
that reloads the menu core under the name "misterzine". A small resident
helper (started from `/media/fat/linux/user-startup.sh`, the same way the
Remote and Zaparoo services start) sees that name appear, opens the
framebuffer console the way the Scripts menu does, runs the app on it, and
reloads the plain menu when you quit. Main itself is untouched. Turning the
setting off removes the boot hook and stops the helper; the Scripts entry
keeps working either way.

### CRT only? One line in MiSTer.ini

Every script's screen, this one and update_all's alike, is drawn into the
framebuffer of the menu core, and the framebuffer goes to HDMI unless the ini
says otherwise. Owners of an HDMI-to-VGA DAC running `direct_video` already
see it. Owners of the analog board's own VGA/YPbPr port need a `[Menu]`
section so the framebuffer reaches the analog port while a script runs:

```ini
[Menu]
direct_video=1
```

That works when nothing is attached to HDMI. With an HDMI display attached
too, use the scaler on VGA with a 15 kHz modeline instead:

```ini
[Menu]
vga_scaler=1
video_mode=640,54,56,106,224,16,0,28,13764
```

Game cores are untouched either way. If you see MiSTer's own "modify
MiSTer.ini ... vga_scaler=1" box when the app starts, this is what it means.

## Controls

MiSTer turns the gamepad into these keys while a script runs.

| Pad | Key | List | Details | Screen | Filters |
|---|---|---|---|---|---|
| D-pad up/down | arrows | move (hold to fly) | pick launch target | | move |
| D-pad left/right | arrows | page (hold to fly) | | previous/next shot | jump section |
| L / R | PageUp/Down | top / bottom | scroll info | | page |

In Settings, Left/Right change a value; A opens the safe-zone editor, where
Left/Right set the side margins and Up/Down the top and bottom ones.
| A | Enter | details | launch | back to details | toggle |
| B | Esc | | back | back to details | close |
| Menu | | back to the MiSTer menu (the app closes) | | | |
| Y | Space | sort: updated/debut | favorite | | toggle |
| X | Tab | filters and settings | shots | | close |

The list shows one row per line: favorite star, title, card status, date.
Status glyphs: `+` current build on the card, `^` older build, `~` on the
card with an unknown build date, `-` not found on the card. Rows changed
since your last look show their date in green, and a "your last look" line
marks where you left off.

Settings (first entry of the X panel) has the safe-zone calibration for
overscan, rotation override, screenshot prefetch, rescan, refresh and cache
clearing. A safe zone up to 32 px is available for consumer sets.

## Data

Rows, screenshots and system photos come from misterzine.fyi (data CC BY
4.0, screenshots as credited there). The app checks for new data on launch
and every 30 minutes, keeps a cache under `/media/fat/misterzine/`, and works
offline from that cache. Pictures download on demand while you browse;
"Prefetch all shots" fetches the whole set (about 55 MB).

A MiSTer without a clock (no RTC, no network) hides relative times rather
than showing wrong ones.

## Building

Go 1.22 or newer. The device binary is a static ARMv7 build:

```
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/misterzine ./cmd/misterzine
```

`go run ./cmd/mzharness` renders every view to PNG on a PC (see its flags),
`tools/dev.sh` is the copy-launch-screenshot loop against a device, and
`tools/make-db.py` builds the downloader database a release attaches.

Debug channel: create `/media/fat/misterzine/debug.flag` and the wrapper
starts a LAN-only HTTP server on port 8195 that injects keys, dumps the
canvas as PNG and reports state. Remove the file to disable it.

## License

MIT. Fonts: Spleen (BSD-2-Clause). Site data and pictures: see misterzine.fyi.
