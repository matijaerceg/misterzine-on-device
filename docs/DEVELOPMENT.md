# Development

## Build and test

Use the Go version in `go.mod`. Go code uses the standard library; the bitmap
fonts and an initial catalogue are embedded in the device binary.

```sh
go vet ./...
go test ./...
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go vet ./...
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/misterzine ./cmd/misterzine
python3 tools/test_maintenance.py
python3 tools/test_package.py
python3 tools/test_wrapper.py
python3 tools/test_snapshot.py
node --test tools/sort_golden.test.js
```

Wrapper and snapshot tests require Bash. Linux-specific host, input and updater
tests run on Linux; Windows tests do not cover those packages. ARM test binaries
can run in a temporary directory on the device. The complete suite includes Unix
permissions checks, so run it in `/tmp` rather than on a FAT card.

To check package installation, legacy upgrade and both removal paths against
an installed Downloader:

```sh
python3 tools/test_downloader_integration.py /path/to/downloader_latest.zip dist/release
```

This uses temporary card directories. Only HTTP transport is replaced with
fixture bytes; Downloader's URL validation, hash checks, file operations and
local registration/removal run normally. It does not run a real system update.

## Visual checks

`cmd/mzharness` runs the UI against fixed data and a controllable clock.
Generate the CI scenarios and compare them with the checked-in baseline:

```sh
python3 tools/render_all.py
python3 tools/render_golden.py out testdata/render_golden.json
```

An intentional UI change requires inspecting the affected PNGs in both
orientations before updating the baseline:

```sh
python3 tools/render_golden.py out testdata/render_golden.json --write
```

Keep `out/` limited to those scenarios. Missing, changed or extra PNGs fail the
comparison. Placeholder-image renders are deterministic and do not replace
checking real artwork, controller feel, and physical CRT/HDMI output.

For a separate picture-enabled review:

```sh
go run ./cmd/mzharness -images /path/to/misterzine/docs/images -out preview
go run ./cmd/mzharness -rot left -logical -images /path/to/misterzine/docs/images -out preview-tate
go run ./cmd/mzharness -canvas 360x270 -out preview-1080p   # the fit-display size for 1080p
go run ./cmd/mzharness -launcher -out preview-launcher   # the main menu shortcut on: its child rows are live
```

The harness leaves page and layout transitions off, so every scripted shot
is an end state. `-motion` runs them on the scripted clock, and the script
command `frames NAME MS COUNT` saves COUNT shots NAME-00, NAME-01, ... MS
apart, for frame-by-frame review or an animated preview:

```sh
go run ./cmd/mzharness -motion -images /path/to/misterzine/docs/images -out motion \
  -script "shot start; press select; press space; frames split 33 7; release space; release select"
```

README screenshots are harness end states, captured against the site's
published export (`docs/releases/data.json` and `meta.json`) with `-now` set to
the day of the shoot and `-images` pointing at the site's `docs/images`, so
dates, counts, sources and artwork are the current ones; tate shots add
`-rot left -logical`. They are individual PNGs enlarged to 2x with
nearest-neighbor resampling. Their HTML dimensions stay at 320x240 (240x320 in
tate), providing sharp pixels on 2x displays. Keep the images inline so the gallery wraps with the
available width; do not combine views into one image or put them in a table.

The startup logo is a paint-only overlay that dithers away over one second from the
first list frame. Preview it with `mzharness -splash` and scripted `wait 500`
steps. `tools/startup_logo.py` renders the checked-in outlined SVG into the
embedded 160×106 two-tone PNG (requires Pillow and svgpathtools). The app
draws it at native size in every orientation, without runtime scaling.

## Device tools and debugging

Set `PI` to your device's IP or SSH hostname when using `tools/dev.sh`; Go
must be on PATH for its build command. It uses SSH and, for some commands, the
separately installed Remote service. Run it against an idle test device.

```sh
tools/dev.sh build
PI=your-device tools/dev.sh deploy
PI=your-device tools/dev.sh launch
```

Debugging is **off by default**, including packaged installations. To enable
remote debugging, create `/media/fat/misterzine/debug.flag` and restart the app.
The wrapper then adds `--debug-http=:8195`. Explicitly passing that argument
also enables debugging. A startup notice confirms it is on.

Every request needs `X-MisterZine-Token`. Retrieve the persistent token from
`/media/fat/misterzine/debug-token` over SSH; the dev tool does this automatically.
Keep tokens out of logs, screenshots, issue reports and source control.
Mutations require POST. Browser-origin requests and non-IP Host names are rejected,
except localhost for tunnels. This opt-in HTTP interface is for a private LAN.

Debug mode enables remote input, state/canvas inspection, detailed input logs and
frame performance collection. `POST /api/saver?knee=&gain=&shade=&div=&passes=&dither=&bits=&cell=`
tunes the screensaver's ground (the bloom knee and gain in 256ths, the shade in
256ths, the blur radius as width/div, the blur passes, the ordered dither 0 or 1,
the depth in bits a channel 1 to 8, the dither cell 2, 4 or 8)
live, and `?saver=on|off`
starts or wakes it; the reply is the look in force. The values are for tuning
sessions only and reset with the app. Ordinary mode retains failure/startup/scan logs and
the F12 screenshot feature, but skips the detailed input and timing instrumentation.
To return a test device to ordinary mode, remove `debug.flag`, remove any explicit
debug argument, and restart. Removing the file cannot stop an already running server.

## On-device troubleshooting

The on-device Troubleshooting screen is separate from remote debugging. Its
explicit controller test temporarily opens independent read-only evdev readers,
including devices omitted by normal keyboard/Start filtering. It never grabs
devices or injects raw events into navigation. A three-second arming interval
precedes a six-second capture; the UI suppresses normal actions until the result.
Original and MiSTer-translated sources remain separate. Device and signal counts
are bounded, and SYN_DROPPED is recorded while invalid event batches are skipped.
Readers close at completion or shutdown. The last report is saved atomically as
`troubleshooting.json`; no listener, detailed normal-input logging, or upload is
enabled. Support launch results distinguish command delivery from game startup.
Harness `-support-report` fixtures render this flow without opening devices.

## Data and generated files

`tools/snapshot.sh` refreshes the embedded first-run catalogue. It does not
replace the coupled sorting fixtures in `testdata/`; those belong to
`tools/sort_golden.js`. The regular network client updates the on-device cache.

`cmd/mzgen` imports the site's display names, source tables and Unit-01
palette into `internal/gen/`. Regenerate those files from the website source
rather than editing generated values by hand:

    go run ./cmd/mzgen -site ../misterzine -out internal/gen

A source the site starts tracking needs nothing else in the app: its display
name, `downloader.ini` section and, for an opt-in database, its `db_url` all
arrive with the regeneration, and the short chip derives from the display name
(`internal/data/labels.go` keeps hand-picked overrides). Only a database that
Update All has a settings toggle for needs its `db_url` added to
`sourceExtras` in `internal/data/sources.go`, because the site records URLs
for the opt-in ones alone. CI clones the public site repository and fails when
`internal/gen/` is behind it, so a site change that touches these tables shows
up as a red run rather than a stale device app.

## Host and updater behavior

The UI owns dataset and navigation state. Background workers deliver results to
it; image decoding and network work stay off the drawing loop. Held navigation
advances at framebuffer pace. The launcher watches an MGL selection, opens the
script console and restores Menu afterward. Its console switch waits at most
two seconds, because the kernel drops the request while the front console is
in graphics mode. A `main=` frontend (Degauss) that took the menu load is
closed first; the menu restore brings it back.

Update All runs under a detached supervisor so it can survive the UI closing or
its executable being replaced. Live state is in RAM, with recovery checkpoints
and bounded output under `misterzine/update-all/`. A run ID, worker PID and boot
ID distinguish a live run from saved history. Acknowledgements do not delete the
original result.

Protected-write detection follows known upstream output markers and checks
known writers before cancellation signals. Unknown output stays visible;
MisterZine cannot guarantee protection against arbitrary external root commands
or loss of power. No timeout guesses when a protected write is safe to kill.
Launcher entry is deferred while known external updaters own the console.

Regression tests use simulated updaters and temporary files. Run a real system
update only as a separately planned device test.
