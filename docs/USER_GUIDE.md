# User guide

## Installation

The recommended installation uses the downloadable `downloader_misterzine.ini`
file. Keep it beside `downloader.ini` on the card, run Update All/Downloader,
then run **MisterZine Setup** from Scripts once. Setup enables the **MisterZine**
main-menu entry and its automatic startup helper.

Alternatively, add this section to `/media/fat/downloader.ini`:

```ini
[misterzine]
db_url = https://github.com/matijaerceg/misterzine-on-device/releases/latest/download/misterzine.json.zip
```

Use one configuration method to avoid duplicate database warnings. Keep the
database ID `misterzine` unchanged. Normal installations follow the latest
stable release; release candidates use an explicitly selected version.

Existing launcher settings and saved favorites/preferences survive an update.
Earlier versions' Scripts launch entry is replaced by the one-time Setup entry.
If the main-menu launcher was previously off, run Setup to enable it.

## List and search

Each row has a favorite marker, title, on-card status, and date:

| Mark | Meaning |
|---|---|
| + | Current build on the card |
| ^ | Older build on the card |
| ~ | On the card, build date unknown |
| - | Not found on the card |

“On card, date unknown” means the scan found the files but cannot compare their
build date with the catalogue. Some core filenames carry no date; sometimes the
catalogue lacks a comparison date or core mapping. It does not mean the build is
old. File modification times are not used as build dates, since copying or
installing a file can change them.

Dates highlighted in green and the last-look divider help find changes since
your previous visit. Visits start at the top of the latest-update sort.
The app checks data on launch and every 30 minutes; Options can request a check
immediately. An unset system clock hides relative dates until trusted time is
available.

Type a title in the list to search. Matching ignores case and spaces and works
together with Filters. Backspace edits; Esc/B clears the query before B opens
Options. Keyboard find currently uses a US layout. Search clears on restart;
filter choices are saved.

## Controls

| Pad | Keyboard | List | Details | Artwork | Filters | Options |
|---|---|---|---|---|---|---|
| Up/Down | arrows | move | choose version | | move | move |
| Left/Right | arrows | page | | change shot | page | change value |
| L/R | PageUp/PageDown | first/last | scroll information | | first/last | first/last |
| A | Enter | details | artwork | | toggle | run action |
| B | Esc | Options / clear search | back | back | back | back |
| X | Tab | Filters | | | | |
| Y | Space | change sort | favorite | | toggle | |
| Start | gamepad Start | launch main version | launch selected version | | | |
| Menu | | return to MiSTer Menu from any screen; closes the app | | | | |

The Details version selector uses one row. Its tally shows the selection and
total, such as 2/5. Up/Down chooses among installed alternatives; Start launches
the chosen version. A dim entry with a leading minus is not on the card.
The information scrollbar shows when more text is available. Long values wrap,
and L/R pages with an overlapping line. Returning from artwork preserves the
selected version and information position.

Artwork starts at the first image each time it is opened. Left/Right chooses a
shot and B returns; A and X have no action in artwork.

F12 on a keyboard saves a PNG to `/media/fat/misterzine/screenshots/`.
Screenshots work without remote debugging.

## Options

Options starts with Refresh data now, Run Update All, Rescan card and Last
update result. Display and browsing preferences follow, with maintenance and
Quit at the bottom. Held scrolling stops at either end. Release and press Up
again from the first item to wrap to Quit, or Down from Quit to wrap to the first
item. R also jumps to the bottom.

- **Rotation:** Left/Right turns the image in that direction. The labels describe
  the monitor's clockwise/counterclockwise turn. Initially follows `osd_rotate`
  in MiSTer.ini until you choose an override.
- **Scroll speed:** 20, 30 or 60 rows/pages per second once a hold repeats.
- **Hold delay:** short 200 ms, normal 300 ms (default), or long 500 ms before
  navigation repeats. Artwork and calibration retain their own timing.
- **Screensaver:** after 1 minute idle by default, dim the picture and scroll
  full-height black MISTERZINE lettering across it. Left/Right chooses Off, 1, 2,
  5 or 10 minutes. A previews it immediately, even when Off. The first browsing
  button or typing key wakes without acting; release it before pressing again.
  MiSTer's Menu button still exits the app. The sweep covers the whole picture,
  including safe-zone margins, in horizontal and tate layouts. Background data
  downloads and Update All continue while dimmed.
- **Edit safe zone:** D-pad moves the top-right corner of the safe frame;
  Right/Up grows it and Left/Down shrinks it. B saves. Margins can reach 40 px.
- **Prefetch shots:** downloads the full picture set in the background.
  Off downloads pictures as you browse. The tally shows cached pictures.
- **Main menu launcher:** off removes the menu entry and boot hook. If the
  launcher opened this session, it returns to Menu before stopping. Run Setup
  from Scripts to re-enable it later.
- **Clear image cache:** removes downloaded pictures, which return as needed.

Filters are ordered On the card, Favorites, Since last look, Type, Source,
Rotation, Genre, Input directions, Buttons and Players. The feed combines zero
and unspecified button counts, so they appear together.

## Update All

Choose **Run Update All** in Options. Browsing pauses while a dedicated screen
shows stages, elapsed time, an activity spinner and a live log. Stages are
milestones, not a percentage of bytes downloaded.

Up/Down scrolls output; L/R pages. Scrolling back freezes the displayed log
snapshot, and reaching the bottom resumes live output.

**Hold B for two seconds to request cancellation.** If a system write is
detected, cancellation waits for it to finish. Cancelling does not undo
completed updates. Restart messages reflect Update All's saved reboot policy.

The supervised updater continues if the app closes. Reopen MisterZine to see
its progress. On completion the card is rescanned. Results appear once and B
acknowledges them; **Last update result** reopens the saved output without
starting another update. After an interrupted run or restart, the app reports
the last known state rather than assuming success.

For an Update All run started from Scripts or Remote, quit MisterZine first,
let the updater finish, then reopen. The menu launcher waits while it detects
an external updater. Use the Options action when you want the supervised screen.

## Removal

Quit MisterZine and wait for any updater to finish. In Scripts, choose
**MisterZine Uninstall**. Up/Down selects a choice, A/Enter selects and confirms,
and B/Esc cancels.

- **Keep favorites and preferences:** removes app files, pictures, logs and
  launcher setup. Keeps `favorites.json`, `settings.json`, `state.json` and
  their recovery/backup files under `/media/fat/misterzine/`. State includes
  remembered filters and last-look information.
- **Remove everything:** removes that saved data as well.

Both choices unregister the `misterzine` database through the installed
Downloader, including its configuration entries. Other databases and startup
services are preserved. Removal uses the installed Downloader locally and
requires its `--uninstall` feature. If it is too old, update Downloader first.

If Downloader reports a removal failure, the helper stops before cleaning up
local data. Follow its reported recovery instructions and retry. Setup can
restore the main-menu entry if removal stopped after disabling it.

To reinstall, repeat the README installation steps. Preserved favorites,
preferences and filters load automatically.
