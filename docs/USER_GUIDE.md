# User guide

## Installation

The recommended installation uses the downloadable `downloader_misterzine.ini`
file. Keep it beside `downloader.ini` on the card, run Update All/Downloader,
then run **MisterZine-Setup** from Scripts once. Setup enables the **MisterZine**
main-menu entry and its automatic startup helper.

Alternatively, add this section to `/media/fat/downloader.ini`:

```ini
[misterzine]
db_url = https://github.com/matijaerceg/misterzine-on-device/releases/latest/download/misterzine.json.zip
filter =
```

Use one configuration method to avoid duplicate database warnings. Keep the
database ID `misterzine` unchanged. Normal installations follow the latest
stable release; release candidates use an explicitly selected version.

The empty `filter =` applies only to MisterZine. It prevents global core filters
from excluding the app's files; your other databases keep their existing filters.

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
your previous visit. Visits start at the top of your last selected sort order.
Options -> Remember sort order is on by default; turn it off to start each visit
with latest updates instead.
Y cycles through latest update, MiSTer debut, and alphabetical title order.
Alphabetical sorting ignores case and accents and puts Game 2 before Game 10.
Y cycles Updated, Debut, Alphabetical, then Favorites. Favorites shows only your
starred entries in alphabetical order, with the same letter jumps as A-Z.
Search and filters still apply. Remember sort order also remembers Favorites.
Changing sort keeps the selected title, search and filters. Dates in alphabetical
order still show the latest update; the last-look divider appears only in the
latest-update order.
In alphabetical order, L/R jumps to the first title of the previous/next letter
in your current search and filters, placing that title at the top of the list
in either direction (even when the final group leaves blank space below).
Empty letters are skipped; numbers and
symbols share a group before A. In the date sorts, L/R jumps between months:
L goes to a newer month, R to an older one. Empty months are skipped, each
month's first result appears at the top, and its month/year briefly appears in
the status bar. Unknown dates share a final group. Jumps stop at either end.
Keyboard Home/End still jumps to the first/last title.
Hold L/R to keep jumping letters or months with the same Hold delay and Scroll
speed as Up/Down row scrolling. Releasing the button stops the jumps.
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
| L/R | PageUp/PageDown | previous/next letter in A-Z; newer/older month in date sorts | scroll information | | first/last | first/last |
| A | Enter | details | artwork | | toggle | run action |
| B | Esc | Options / clear search | back | back | back | back |
| X | Tab | Filters | | | | |
| Y | Space | change view | favorite | | toggle | |
| Start | gamepad Start | launch main version | launch selected version | launch selected version | | |
| Menu | | return to MiSTer Menu from any screen; closes the app | | | | |

The Details version selector uses one row. Its tally shows the selection and
total, such as 2/5. Up/Down chooses among installed alternatives; Start launches
the chosen version. A dim entry with a leading minus is not on the card.
The information scrollbar shows when more text is available. Long values wrap,
and L/R pages with an overlapping line. Returning from artwork preserves the
selected version and information position.

Artwork starts at the first image each time it is opened. Left/Right chooses a
shot and B returns; A and X have no action in artwork. Start launches the version
selected in Details. Any launch failure appears over the artwork.

F12 on a keyboard saves a PNG to `/media/fat/misterzine/screenshots/`.
Screenshots work without remote debugging.

## Options

Options starts with Refresh data now, Run Update All, Rescan card and Last
update result. Display and browsing preferences follow, with maintenance and
Quit at the bottom. Held scrolling stops at either end. Release and press Up
again from the first item to wrap to Quit, or Down from Quit to wrap to the first
item. R also jumps to the bottom.

- **Rotation:** Left/Right turns the image in that direction. The labels describe
  the monitor's clockwise/counterclockwise turn.
- **Follow INI rotation:** on by default, including after upgrading. Every
  startup reads `osd_rotate` from MiSTer.ini (Menu overrides the global setting).
  Manual rotation changes work for the current session. Turn this off to retain
  your current orientation across launches. Enabling it applies on the next
  startup; MisterZine never edits the INI. If the file cannot be read, the saved
  manual rotation is used.
- **Scroll speed:** 20, 30 or 60 rows/pages per second once a hold repeats.
- **Hold delay:** short 200 ms, normal 300 ms (default), or long 500 ms before
  navigation repeats. Artwork and calibration retain their own timing.
- **Remember sort order:** on by default, including after upgrading. Reopens
  with your last sort choice; off starts new visits with latest updates. Changing
  this preference leaves the current sort order alone.
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
- **Troubleshooting:** a guided Start-button test and an A/Enter game-launch
  test, with results you can photograph. The last result survives restarting.
  See [troubleshooting](TROUBLESHOOTING.md#start-does-not-launch-games).

Filters start with On the card, Favorites, Since last look, Type and Source.
Press Y on a value to select only that value in its section, leaving all other
sections and your search unchanged. Press Y again on that value to restore the
section's previous selection. If an isolated selection was loaded from a previous
session, Y restores all values in that section. A still toggles values individually.
The bottom legend shows the Y shortcut where available; Y does nothing on
section headings. For Favorites/Since, Y toggles the choice.

The latest-update view's last-look marker examines at most the first 200 matching
entries. With no changes there it says **No changes in top 200**; if the entire
view fits within that window it says **No changes in this view**. These compare
against your previous visit. The Since last look filter checks the full catalogue.
On-card choices include counts across the whole catalogue, independent of the
current search and other filters. Favorites mode always shows favorites only;
use Y in the main view to leave it.

Card scans run quietly on launch. Options -> Rescan card shows a result screen
with up-to-date, older, undated, missing and unknown totals. B closes it even
while scanning; completion does not interrupt the screen you moved to.
Unreadable card scans retain the previous inventory and show a failure banner.

MisterZine checks GitHub for a newer stable app release on launch and at most
every 30 minutes. When available, the main bar shows **App update**, and Options
shows the version beside the Update All instructions. Run Update All, then quit
and reopen MisterZine to use the new app. Offline checks stay quiet. This is
separate from automatically refreshing the games catalogue; prerelease and
development builds do not advertise stable downgrades.
The **Arcade game filters** section contains Rotation, Resolution, Genre,
Controls, Buttons and Players. These choices affect arcade games only; Type
controls whether console, computer and other cores appear. Excluding Arcade
hides its filter section while retaining your choices for later.

Counts reflect your search and the other filter sections. A choice's count
ignores its own section, so unchecked choices still show how many entries they
could include. Choices remain selectable at zero. Each section heading toggles
all its choices; Clear all filters restores everything.

Resolution uses the catalogue's 15kHz/31kHz labels. **Unknown** means that value
is missing. Controls includes separately recorded special controls such as
spinners, paddles and trackballs. A button count alone does not identify a
buttons-only game. **0 buttons** requires an explicit zero in the catalogue;
older feeds that omitted zero counts still show those entries as Unknown.

Provisional values are supplied from fallback sources pending the curated
Arcade Database. They match their ordinary filter categories and are marked
"(provisional)" in the game's details.

## Update All

Choose **Run Update All** in Options. Browsing pauses while a dedicated screen
shows stages, elapsed time, an activity spinner and a live log. Stages are
milestones, not a percentage of bytes downloaded.

Up/Down scrolls output; L/R pages. Hold either to repeat using your Hold delay and
Scroll speed settings, during a run or in a saved result. Scrolling back freezes the displayed log
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
**MisterZine-Uninstall**. Up/Down selects a choice, A/Enter selects and confirms,
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
