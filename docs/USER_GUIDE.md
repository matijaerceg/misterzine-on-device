# User guide

## Installation

The recommended installation uses the downloadable `downloader_misterzine.ini`
file. Keep it beside `downloader.ini` on the card, run Update All/Downloader,
then run **MisterZine-Setup** from Scripts once. Setup enables the **MisterZine**
main-menu entry and its automatic startup helper.

If another frontend is your main menu (Degauss), its Scripts list has
**MisterZine-Run**, which opens MisterZine directly; the frontend returns when
MisterZine quits. Opened that way, leave through Options -> Quit MisterZine or
the pad's Menu button (set to leave), as keyboard F12 stays with the frontend's
script session. Choosing MisterZine from the MiSTer menu itself works too.

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

Each row has a favorite marker, title, on-card status, and date. Titles use a
narrow font so more of each name fits, and the status and date columns use
the same narrow font at the titles' height; Options -> Title font chooses the
font. A yellow β after a title marks a Patreon beta core, which needs
Jotego's jtbeta.zip. The date column format is chosen in Options -> Date
format.

| Mark | Meaning |
|---|---|
| + | Current build on the card |
| ^ | Older build on the card, or an undated core that differs from the shipped build |
| ~ | On the card, build date unknown |
| - | Not found on the card |

The scan compares each core file on the card with the build the catalogue
ships. Dated core filenames are compared by build date. Undated cores (Jotego's,
for example) are compared by file checksum against the shipped build, using the
same record Update All keeps, so “older build likely” means Update All would
replace the file. A core installed after the catalogue was generated is treated
as current.

“On card, date unknown” means the scan found the files but cannot compare them
with the catalogue: the filename carries no date and the catalogue has no
checksum or core mapping for the row. It does not mean the build is old. File
modification times are not used as build dates, since copying or installing a
file can change them.

Dates highlighted in green and the last-look divider help find changes since
your previous visit. Visits start at the top of the view you left.
Options -> Remember last view is on by default; turn it off to start each visit
in Core updated instead.
Y cycles Updated, Debut, Year, Alphabetical, Manufacturer, then Favorites; the
top bar names the order as core updated, MiSTer debut, original year, A-Z,
manufacturer or Favorites A-Z. Options -> Views can leave any of them out of the cycle.
With Recents on there, Recents follows Favorites: the games launched from
MisterZine, latest launch first, with the launch date in the date column
and L/R jumping between launch months.
Year orders games
by their original release year, newest first, with titles alphabetical within
a year and unknown or uncertain years last; a header line names each year,
the last one Year unknown, and the title takes the date column's room, with
only the card status glyph kept at the right.
Alphabetical sorting ignores case and accents and puts Game 2 before Game 10;
a header line names each letter, with numbers and symbols under 0-9 and
symbols before A.
Manufacturer groups the games by manufacturer: a header line names each one,
manufacturers run A-Z with titles A-Z under them, the date column shows the
original year, and L/R jump from manufacturer to manufacturer. Joint credits and licences count under the
first company named and corporate or regional suffixes are ignored, so Taito
Corporation Japan and Taito America Corporation sit under Taito and Data East
USA under Data East; Details still shows the full credit. Games without a
manufacturer come last under Unknown manufacturer.
In the three grouped orders (Year, A-Z, Manufacturer) a group whose header has
scrolled off the top keeps its name pinned on the top line until its last
row has gone.
Favorites shows only your starred entries in alphabetical order, with the same
letter jumps as A-Z. Star a game with Y in Details or Select + A on the list.
Search and filters still apply. Remember last view also remembers Favorites.
Changing sort keeps the selected title, search and filters. Dates in alphabetical
order still show the latest update; the last-look divider appears only in the
latest-update order.
In alphabetical order, L/R jumps to the first title of the previous/next letter
in your current search and filters; the title lands centred with its header
line right above it, as in the Year and Manufacturer orders. Empty letters are
skipped; numbers and symbols share a group before A. In the date sorts, L/R
jumps between months: L goes to a newer month, R to an older one. Empty months
are skipped, each month's first result appears at the top, and its month/year
briefly appears in the status bar. Unknown dates share a final group. In the
Year order the same keys jump between release years. Jumps stop at either end.
Keyboard Home/End still jumps to the first/last title.
Hold L/R to keep jumping letters or months with the same Hold delay and Scroll
speed as Up/Down row scrolling. Releasing the button stops the jumps.
The app checks data on launch and every 30 minutes; Options can request a check
immediately. An unset system clock hides relative dates until trusted time is
available.

Type a title in the list to search. Matching ignores case and spaces and works
together with Filters. Backspace edits, and held it erases quickly after the
Hold delay; Esc/B clears the query before B opens Options. Keyboard find currently uses a US layout. Search clears on restart;
filter choices are saved.

## Controls

| Pad | Keyboard | List | Details | Artwork | Filters | Options |
|---|---|---|---|---|---|---|
| Up/Down | arrows | move | page information | | move | move |
| Left/Right | arrows | page, keeping the row centered | choose version | change shot | collapse/expand section or years | change value |
| L/R | PageUp/PageDown | previous/next letter in A-Z; newer/older month in date sorts; previous/next year or manufacturer | | | previous/next section | first/last |
| A | Enter | details | artwork | | toggle a value; open/close a heading | run action |
| B | Esc | Options / clear search | back | back | back | back |
| X | Tab | Filters | | | back to the list | |
| Y | Space | change view | favorite | | only/all | |
| Start | gamepad Start | launch remembered version | launch selected version | launch selected version | | |
| Select (held) | | + Y: List layout, + X: List shots, + A: favorite | | | | |
| Menu | | Options; held 2 s: quit MisterZine (or quit at once: Options -> Menu button) | Options | Options | Options | back to the screen it was opened over |

A, B, X and Y are MiSTer's names from its define buttons screen. Options ->
Button labels can print them as Xbox letters, PlayStation symbols or numbers.

A pad that has been defined in the MiSTer menu is read by MisterZine
directly, slot by slot: D-pad, A B X Y, L R, Start, and the stick you gave
the MiSTer menu, if any. The button you defined as A is A here, whichever
buttons you chose for the MiSTer menu's own OK and back. A trigger or stick
direction used as a button counts once it passes a quarter of its travel,
as in MiSTer. Each pad follows its own definition, so two pads defined
differently work side by side. A pad that has never been defined works
through MiSTer's translation instead, and only while no defined pad is
connected. A button that is in no MiSTer slot does nothing here either;
the pad tester under Troubleshooting shows it with its raw code.

Hold Select on the list (the legend's last entry, Select +, is the
reminder) and the legend changes: Y cycles List layout, X switches List shots, saved exactly as from Options, and A stars or unstars
the game under the cursor (in the Favorites view an unstarred row drops
out and the cursor moves to its neighbour). Select alone does nothing, and
while it is down every other button waits, so nothing moves or launches by
accident. Only a pad defined in MiSTer has a Select; keyboards use Options
and Details.

MisterZine holds a defined pad exclusively while it runs, so MiSTer sees
nothing from it: the button you defined as MiSTer's menu (OSD) button is
MisterZine's Menu button, opening Options by default (Options -> Menu button
can make it quit instead). A tap opens or closes Options as you let go.
Held for two seconds it quits MisterZine for the MiSTer menu whatever the
setting; a hint appears after 300 ms, and from then on the screen stays as
it was: letting go early neither opens nor closes Options. While the hint
is up, the line under the top bar fills green from left to right, reaching
the far edge at the moment MisterZine quits. The other two holds, B to
cancel Update All and B to leave the pad tester, fill the same line from
the moment B goes down.
A pad whose A or B MisterZine cannot read is left to MiSTer's translation,
as before. Keyboard F12 and the board's own button still hand the screen to
MiSTer, which closes the app.

The Details version selector uses one row. Its tally shows the selection and
total, such as 2/5. Left/Right chooses among installed alternatives; Start
launches the chosen version. A dim entry with a leading minus is not on the
card. A version name too long for the row scrolls back and forth, pausing at
each end, so alternatives that share a long prefix can be told apart.
The version chosen is remembered for that game, across restarts: Details
opens on it next time and Start launches it from the list and from artwork.
Alternatives are read from `_Arcade/_alternatives` and from the
`_alternatives` folder inside a database's own folder, such as
`_Arcade/_MeatCores` and `_Arcade/_rmCores`.
Choosing the main version again forgets the choice, and an alternative that
has since left the card falls back to the main version.
A green scrollbar thumb shows when more information is available. Long values
wrap, and Up/Down pages with an overlapping line, sliding the text smoothly;
the legend mentions Left/Right only when a game has more than one version
and Up/Down only when there is more to read.
Returning from artwork preserves the selected version and information position.

Artwork starts at the first image each time it is opened. Left/Right chooses a
shot and B returns; A and X have no action in artwork. Start launches the version
selected in Details. Any launch failure appears over the artwork.

F12 on a keyboard saves a PNG to `/media/fat/misterzine/screenshots/`.
Screenshots work without remote debugging.

## Options

Options is in four groups under grey headings: Data (Refresh data now, Run
Update All, Last update result, Rescan card, Prefetch shots, Clear image
cache), List (Sources, Filter by rotation, Remember last view, Views,
Title font, List shots, Date format, List layout), Display (rotation, screensaver, button labels, safe zone, HDMI picture) and
Operation (scrolling, the launcher, Open at boot, Return after game,
Troubleshooting, Credits, Quit). The
headings cannot be selected. Values line up in one column; a small arrow at the right
edge of the top or bottom row shows when more options sit above or below the
visible part of the list. The framed text under the list describes the selected
option, and the build version and catalogue date sit below it in grey. B
(or the Menu button) returns to the screen Options was opened over: the
list, or Details, the artwork or Filters when the Menu button opened it
there, Filters as you left it. The next B reopens Options on the row you
left, the way the list keeps its row; Filters opened from the list always
starts at its top. Held
scrolling stops at either end. Release and press Up again from the first item
to wrap to Quit, or Down from Quit to wrap to the first item. R also jumps to
the bottom.

- **Prefetch shots:** downloads the full picture set in the background.
  Off downloads pictures as you browse. The tally shows cached pictures.
- **Clear image cache:** removes downloaded pictures, which return as needed.
- **Sources:** all by default. Installed only shows games from the databases
  listed in the card's `downloader.ini`, the file Update All rewrites on every
  run, so a source you never enabled (Coin-Op, Meathax, rmCores or Jotego) stays out of
  the list, search, Favorites and the Filters panel, and the release count
  leaves it out too. The file is read at every card scan: launch, Rescan card
  and the rescan after Update All. Without database sections Downloader
  fetches MiSTer Distribution alone, and the option follows that; without any
  `downloader.ini` nothing is hidden and the help text says so. Sources the
  app does not know stay visible. rmCores rows are rmonic79's own builds of
  games the MiSTer Distribution also has, with CRT Adjust and V-Size on the
  analog output and a pause overlay; Details says so under Source.
- **Filter by current rotation:** off by default. On immediately shows only
  entries explicitly marked Horizontal when the interface is horizontal, or
  Vertical when it is rotated either way, whatever set that rotation (the INI or
  a manual choice). Unknown orientations and system entries without matching
  orientation metadata are hidden, including in Favorites. Search and other
  filters remain active; saved manual rotation filters are temporarily
  superseded and restored when this option is turned off. Filters shows the
  active rule, and it follows the interface when the rotation changes.
- **Remember last view:** on by default, including after upgrading. Reopens
  in the view you left; off starts every visit in Core updated. Changing
  this preference leaves the current view alone.
- **Views:** which list orders Y cycles through, as a checkbox page: core
  updated, MiSTer debut, original year, A-Z, Manufacturer, Favorites and Recents.
  Everything is on except Recents by default. A toggles a view; the last one
  on cannot be turned off (it is greyed, and A there says so), and turning off
  the view you are in moves the list to the next one on. The Options row
  counts them, such as "Views (6 of 7)".
  Recents lists the games launched from MisterZine, latest first, each once
  with its last launch date, up to 100; search and filters apply. That history
  is MisterZine's own, kept in its state.json whether the view is on or off,
  and separate from MiSTer's recent lists.
- **Title font:** narrow tall by default, a condensed proportional font that
  fits about a third more of each title on a row, made one pixel taller so
  its capitals and lowercase match the body font's height. Narrow is the
  original, shorter face. Normal uses the body font for titles. The status
  and date columns always use the narrow font at the titles' height.
- **List shots:** which screenshot the list pane shows for the selected row:
  gameplay (default) or the title screen. Details and the artwork view still
  show every shot. Hold Select and press X on the list to switch it there.
- **Date format:** how list dates read: MM-DD (default), DD-MM, Mon D (Sep 7),
  D Mon (7 Sep) or YYMMDD (260907). The help line shows today's date in the
  chosen format. Rows from earlier years show the year alone, except with
  YYMMDD, which always carries the full date.
- **List layout:** list by default: the full list with the small pane beside
  it (or below it in tate). Split widens the pane to under half the screen
  with a picture to match, leaving shorter rows. Picture runs the picture
  across the screen, in horizontal above a few full-width rows with the
  details beside it, in tate below the rows; a vertical shot keeps its shape
  and the details sit beside it when there is room, otherwise below in three
  lines. Every layout keeps at least four rows, even with a wide safe zone.
  Hold Select and press Y on the list to cycle the layout there.
- **Follow INI rotation:** on by default, including after upgrading. Every
  startup reads `osd_rotate` from the MiSTer INI that is currently active: the
  main MiSTer.ini, or the alternative INI (`MiSTer_*.ini`) you selected in the
  MiSTer OSD, the same one Main itself uses. `[Menu]` and `[MisterZine]`
  sections override the global setting in file order, as Main applies them to
  MisterZine's menu entry. Turn this off to choose a rotation yourself below and keep it
  across launches; turning it back on applies at the next startup. MisterZine
  never edits the INI. If the file cannot be read, the saved manual rotation is
  used.
- **Rotation:** available when Follow INI rotation is off. Left/Right turns the
  image in that direction and the choice is saved. The labels describe the
  monitor's clockwise/counterclockwise turn. While following the INI, the row
  shows the current rotation in grey and cannot be changed here.
- **Screensaver:** after 1 minute idle by default, fade and blur the picture
  over a second and scroll full-height black MISTERZINE lettering, with glinting
  chrome edges, across the blurred picture. Left/Right chooses Off, 1, 2,
  5 or 10 minutes. A previews it immediately, even when Off. The first browsing
  button or typing key wakes without acting; release it before pressing again.
  The pad's Menu button opens Options (or quits, per Options -> Menu
  button; held 2 s it quits either way). The sweep covers the whole picture,
  including safe-zone margins, in horizontal and tate layouts. Background data
  downloads and Update All continue while dimmed.
- **Button labels:** how the legends name the pad's face buttons, in MiSTer's
  A B X Y order: A B X Y (default), B A Y X for an Xbox-lettered pad mapped by
  position, the PlayStation circle, cross, triangle and square by position, or
  1 2 3 4 in MiSTer's define order. Only the names change. Which physical
  button MiSTer calls A, B, X or Y comes from its define buttons screen; this
  guide uses those names.
- **Edit safe zone:** D-pad moves the top-right corner of the safe frame;
  Right/Up grows it and Left/Down shrinks it. B saves. Margins can reach 40 px.
- **HDMI picture:** fit display by default: the picture is sized so that
  MiSTer's integer scaling fills the screen height: 360x270 on 1080p, 340x256
  on 1024x768, 400x300 on 1600x900, instead of 320x240 with bars above and
  below. Text keeps its size; the list gains rows. CRT modes, 720p and 1440p
  already fit and stay 320x240, so the setting changes nothing there. Choose
  320x240 to keep the classic size on every display. Applies when MisterZine
  next starts.
- **Scroll speed:** 20, 30 or 60 rows/pages per second once a hold repeats.
- **Hold delay:** short 200 ms, normal 300 ms (default), or long 500 ms before
  navigation repeats. Artwork and calibration retain their own timing.
- **Main menu launcher:** off removes the menu entry and boot hook. If the
  launcher opened this session, it returns to Menu before stopping. Turning
  the option back on puts both back. With it off, open MisterZine through
  Scripts -> MisterZine-Run (or run Setup again).
- **Open at boot:** off by default. On: once the MiSTer menu is up after a
  power-on or reboot, the launcher picks the MisterZine entry for you, the same
  way you would. A `bootcore` in the INI takes precedence and nothing happens.
  Needs the Main menu launcher.
- **Return after game:** off by default. On: when a game you started from
  MisterZine exits to the MiSTer menu (Reset or Exit in the OSD), the launcher
  picks the MisterZine entry again and the list opens on that game. Games
  loaded some other way, and quitting through the Menu button, do not bring
  it back. Needs the Main menu launcher.
- **Menu button:** what the pad button you defined as MiSTer's menu (OSD)
  button does in MisterZine, which holds a defined pad while it runs: Options
  (default) from any screen, closing it when it is up, or quit MisterZine
  for the MiSTer menu, as earlier versions did. With Options chosen, holding
  the button for two seconds quits as well, so the way out is always there.
  Keyboard F12 and the board's button still quit.
- **Troubleshooting:** a guided Start-button test, a pad tester, and an
  A/Enter game-launch test, with results you can photograph. The last result
  survives restarting. The pad tester lists every connected pad with the raw
  button code in each MiSTer slot, then shows each press as it happens: which
  pad, which button, which slot and what MisterZine does with it, plus the
  gap since the previous press. Hold B for two seconds to leave it.
  See [troubleshooting](TROUBLESHOOTING.md#start-does-not-launch-games).
- **Credits:** a page naming the developer, the people whose work MisterZine
  builds on (MiSTer, Downloader and Update All, the fonts) and, under Special
  thanks, the early adopters who tested it and sent feedback. Up/Down move
  through the names; B returns to Options.
- **Quit MisterZine:** the last row: back to the MiSTer menu (or to the
  frontend that opened MisterZine). The pad's Menu button held for two
  seconds does the same.

Filters start with Clear all filters, greyed out until a filter is active,
then On the card, Since last look, Type and Source. Pressing Clear all filters
moves the selection to the first heading. The same edge arrows as in Options
show when the list continues above or below.
Favorites is a main-view mode and still respects all these filters.
Press Y on a value to select only that value in its section, leaving all other
sections and your search unchanged. Press Y again on that value to enable all
values in its section. A still toggles values individually. On a section
heading, A opens a closed section or closes an open one, the same as Left/Right;
the legend reads "A ◀ ▶ Open/close" while a heading is selected.
Y does nothing on section headings. For Since last look, Y toggles the choice.

Filters -> Original release year groups arcade games into decades. A toggles
the selected decade; Right expands its individual years and Left collapses
them again, returning to the decade's row. Left on a collapsed decade closes
the section. Each time you open Filters, only sections
with active choices start expanded. Unchanged sections start collapsed.
A right/down arrow marks collapsed/expanded sections. An asterisk after the
arrow marks active filters, including the current-rotation rule. It disappears
when that section returns to its defaults. Collapsing
changes only the display, not your filter choices. L/R jumps to the
previous/next section heading. Keyboard Home/End still goes to the first/last row.

In the main list, Filters and Options, Up/Down keeps the selected row near the center,
and Left/Right moves a screen of rows at a time with the same centering.
Scrolling stops at the beginning and end so the list stays filled. Main-list
letter/month jumps still align the first row at the top; moving Up/Down resumes
centered scrolling.
Y shows only that decade or year; press Y again to enable all years. A mixed
decade shows `[-]`. Choices are saved and combine with your other filters.
Unknown or uncertain years have an Unknown choice. System cores are unaffected.
These are the games' original release years, not their MiSTer debut/update dates.

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
hides its filter section while retaining your choices for later. Under Type,
Arcade opens (Right, or X) into Stable and Beta, the Patreon beta cores, like
a decade opens into years: A toggles one, Y shows only one, and a mixed choice
shows `[-]` on Arcade. Turning both off is the same as turning Arcade off.

Counts reflect your search and the other filter sections. A choice's count
ignores its own section, so unchecked choices still show how many entries they
could include. Choices remain selectable at zero. Y on a value twice enables
every choice in its section; Clear all filters restores everything.

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
