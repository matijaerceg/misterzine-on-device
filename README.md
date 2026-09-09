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

From the main list, press B for Options, select **Run Update All**, then A.
The dedicated screen shows five stages, an activity spinner, elapsed time,
time since the last output, and a live log. The bar advances as known
stages are announced; it is not a download percentage. Up/Down scroll the
log and L/R move by a page. Browsing and core launching are blocked until
the run finishes.

**Hold B for two seconds to cancel.** During a detected Linux, firmware or
bootloader write, cancellation waits until that work finishes. Cancelling
does not undo files already updated. A **Restart expected** ribbon appears
as soon as a Linux update or reboot requirement is detected. Update All
uses your saved configuration, including its automatic restart setting.

If Main's menu button closes MisterZine, this supervised run keeps working
and recording output. Reopen MisterZine to reconnect. After completion,
B returns to Options and the card is rescanned. The most recent summary
and bounded output log are retained in `/media/fat/misterzine/update-all/`.
After a reboot, the next launch shows the last known result; an interrupted
run is never assumed successful.

For the usual Scripts-menu or Remote launch, quit MisterZine before
starting Update All, let the updater finish (including any requested
reboot), then reopen MisterZine. Both apps use the same script console;
hiding Update All does not turn it into an independent background job.
MisterZine's menu helper refuses to take over while it detects an external
updater. Use the Options action for the supervised workflow above.

An updater started independently through SSH with its own input/output
can keep working while MisterZine is open. Quitting MisterZine does not
pause that process. Updates can replace cores or restart the system, so
wait for completion before launching a core, and use Options → Rescan
to refresh the on-card status if MisterZine stayed open. This is not a
promise that every Update All configuration supports concurrent use.

### Put it in the main menu

Once, from the main list: B, Options, "Main menu launcher", Right. From then on a
"MisterZine" entry sits in the main menu next to Arcade, Console and the
rest, and picking it opens the app; no Scripts menu.

How it works: the database ships `MisterZine.mgl` at the card root, an MGL
that reloads the menu core under the name "misterzine". A small resident
helper (started from `/media/fat/linux/user-startup.sh`, the same way the
Remote and Zaparoo services start) sees that name appear, opens the
framebuffer console the way the Scripts menu does, runs the app on it, and
reloads the plain menu when you quit. Main itself is untouched. Turning the
setting off removes the boot hook and menu entry. If the helper opened the
current app session, it first returns you to Menu, then stops. Turning the
setting back on before quitting keeps it enabled. The Scripts entry keeps
working either way, so you can reopen the app there to re-enable the launcher.

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

| Pad | Key | List | Details | Screen | Filters | Options |
|---|---|---|---|---|---|---|
| D-pad up/down | arrows | move (hold to fly) | pick launch target | | move | move |
| D-pad left/right | arrows | page (hold to fly) | | previous/next shot | page | change value |
| L / R | PageUp/Down | top / bottom | scroll info | | top / bottom | top / bottom |
| A | Enter | details | shots | back to details | toggle | open action |
| B | Esc | Options | back | back to details | back to list | back to list |
| Start | gamepad Start | launch main version | launch picked version | | | |
| Menu | | back to the MiSTer menu (the app closes) | | | | |
| Y | Space | sort: updated/debut | favorite | | toggle | |
| X | Tab | Filters | | back to details | | |

In Options, Left/Right change a value; A opens the safe-zone editor.
There, the d-pad moves the top-right corner of the safe frame, and B saves
and returns to Options.

The list shows one row per line: favorite star, title, card status, date.
Status glyphs: `+` current build on the card, `^` older build, `~` on the
card with an unknown build date, `-` not found on the card. Rows changed
since your last look show their date in green, and a "your last look" line
marks where you left off.

If the favorites file cannot be read, the app shows "Favorites unreadable;
file kept" and disables favorite edits for that session. It preserves
`favorites.json` in place, including on quit and subsequent launches.
After the file is readable or restored from a backup, reopen MisterZine
to load it and enable favorite edits again.

Options (B from the main list) has the safe-zone calibration for
overscan, rotation override, held-scroll speed, screenshot prefetch, the menu
launcher, Update All, rescan, refresh, cache clearing and Quit MisterZine. A safe zone up to 40 px is available for consumer sets.

Filters (X from the main list) is a separate screen, ordered: On the card,
Favorites, Since last look, Type, Source, Rotation, Genre, Input directions,
Buttons, Players. Filters uses the smaller font. Button counts omitted by
the data appear under "0 / not specified" because the feed combines those
cases. Options keeps its version and data timestamp visible in both
orientations, with three help lines in horizontal mode and four in tate.

## Data

Rows, screenshots and system photos come from misterzine.fyi (data CC BY
4.0, screenshots as credited there). The app checks for new data on launch
and every 30 minutes, keeps a cache under `/media/fat/misterzine/`, and works
offline from that cache. Pictures download on demand while you browse;
"Prefetch shots" is an off/on setting changed with Left/Right; its tally shows
how many images are on the card. When on, it downloads the full image set.

A MiSTer without a clock (no RTC, no network) hides relative times rather
than showing wrong ones.

## Building

Go 1.27 or newer (matching `go.mod`). The device binary is a static ARMv7 build:

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

### Debug access and update results

The opt-in LAN debug server requires `X-MisterZine-Token` on every request.
On first use it creates `/media/fat/misterzine/debug-token`; retrieve it through
SSH, not through HTTP. `tools/dev.sh` reads it over SSH automatically. Mutations
(keys, goto and quit) require POST. Browser-origin requests and non-IP Host names
(other than localhost for tunnels) are rejected. The app shows a startup notice
when remote debugging is enabled. Do not publish the token or include it in logs.

Options → Last update result reviews the saved output without running Update All
again. Unseen terminal results appear once on launch; B acknowledges that result.
Run Update All still starts a new run. Rescan card remains immediately below it.

The Scripts wrapper keeps a temporary restore helper in RAM during the app run.
On an error it shows the recent log and allows up to 15 seconds to read it before
returning to Menu. Normal exits have no extra pause.

### Render regression baseline

CI compares all 16 deterministic harness PNGs against `testdata/render_golden.json`.
For an intentional visual change, run the four Harness renders commands from
`.github/workflows/ci.yml`, inspect the PNGs in `out/`, then update the baseline:

```sh
python3 tools/render_golden.py out testdata/render_golden.json --write
```

Do not regenerate the baseline merely to silence a failed comparison. Keep `out/`
limited to those harness outputs; extra or missing PNGs also fail the check.
