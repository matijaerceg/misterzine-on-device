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

## Page navigation

Opening a page reveals it from the right with a quick, soft-edged wipe. Going
back reverses the direction. Changing the browsing view uses the same effect;
list movement, page jumps, and option adjustments remain immediate. Buttons
keep working during the transition. Options -> Display -> Page transitions
turns the effect on or off; the choice is remembered.

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
as current. A core file is matched by name the way MiSTer itself loads it: the
exact name, or any file whose name starts with that name followed by an
underscore (Coin-Op ships `blkheart_mister_20260909.rbf` for `blkheart`).

“On card, date unknown” means the scan found the files but cannot compare them
with the catalogue: the filename carries no date and the catalogue has no
checksum or core mapping for the row. It does not mean the build is old. File
modification times are not used as build dates, since copying or installing a
file can change them.

Dates highlighted in green and the last-look divider help find changes since
your previous visit. Visits start at the top of the view you left.
Options -> Views -> Remember last view is on by default; turn it off to choose
a Default view for each new visit (Core updated initially).
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
The end of the Options list shows **Catalog**, when the catalog was published,
and **Last checked**, the last successful check during this session. An older publication time can
still be current. Routine checks and data age stay out of the list's top bar;
connection failures and app-update availability remain visible.

Type a title in the list to search. Matching ignores case and spaces and works
together with Filters. Backspace edits, and held it erases quickly after the
Hold delay; Esc/B clears the query before B opens Options. Keyboard find currently uses a US layout. Search clears on restart;
filter choices are saved.

## Controls

| Pad | Keyboard | List | Details | Artwork | Filters | Options |
|---|---|---|---|---|---|---|
| Up/Down | arrows | move | page information | | move | move |
| Left/Right | arrows | page, keeping the row centered | choose version | change shot | collapse/expand section or years | change value |
| L/R | PageUp/PageDown | previous/next letter in A-Z; newer/older month in date sorts; previous/next year or manufacturer | | | previous/next section | previous/next section |
| A | Enter | details | artwork | | toggle a value; open/close a heading | run action |
| B | Esc | Options / clear search | back | back | back | back |
| X | Tab | Filters | | | back to the list | |
| Y | Space | change view | favorite | | only/all | |
| Start | gamepad Start | launch remembered version | launch selected version | launch selected version | | |
| Select (held) | | + Y: Layout, + X: Art type, + A: favorite | | | | |
| Menu | | Options; held 2 s: quit MisterZine (or quit at once: Options -> Menu button) | Options | Options | Options | back to the screen it was opened over |

A, B, X and Y are MiSTer's names from its define buttons screen. Options ->
Button labels can print them as Xbox letters, PlayStation symbols or numbers.

A pad that has been defined in the MiSTer menu is read by MisterZine
directly, slot by slot: D-pad, A B X Y, L R, Start, and the stick you gave
the MiSTer menu, if any. A trigger or stick direction used as a button
counts once it passes a quarter of its travel, as in MiSTer. Each pad
follows its own definition, so two pads defined differently work side by
side. A button that is in no MiSTer slot does nothing here either; the pad
tester under Troubleshooting shows it with its raw code.

A pad that has never been defined in MiSTer is held as well when it
reports the standard Linux gamepad layout, as nearly every pad does: by
position, the right button is A, the bottom one B, the top one X and the
left one Y, the shoulders are L and R, the home or guide button is Menu,
and the hat and the left stick move. The bottom button confirms on such a
pad, as it does in MiSTer's own default, so B confirms and A goes back
until you define the pad in the MiSTer menu, which always wins. The pad
tester calls this "Linux default layout". A pad that reports no such
layout works through MiSTer's translation instead, and only while no held
pad is connected.

Which button confirms follows the MENU OK you chose at the end of MiSTer's
define screen. If you put it on the button you defined as B, B confirms
here too and A goes back, and the legends say so; the table above then
reads with A and B swapped for that pad. Options -> OK button shows the
choice for the pad in your hand and can override it, per pad.

Hold Select on the list (the legend's last entry, Select +, is the
reminder) and the legend changes: Y cycles Layout, X switches Art type, and A stars or unstars
the game under the cursor (in the Favorites view an unstarred row drops
out and the cursor moves to its neighbour). Select alone does nothing, and
while it is down every other button waits, so nothing moves or launches by
accident. Layout and Art type are saved, and both are also in Options.
Only a pad defined in MiSTer has a Select.

**Layouts:** List is the default: the full list with the small pane beside
it (or below it in tate). Split widens the pane to under half the screen
with a picture to match, leaving shorter rows. Picture runs the picture
across the screen, in horizontal above a few full-width rows with the
details beside it, in tate below the rows; a vertical shot keeps its shape
and the details sit beside it when there is room, otherwise below in three
lines. Text drops the pane: the rows alone fill the screen, so long titles
have more room in horizontal and tate shows more of them; A still opens the
details. Every layout keeps at least four rows, even with a wide safe zone.
Hold Select and press Y on the list to cycle the layout there, or pick one
in Options -> List -> Layout, where the help box draws all four.
The list and artwork move into place in a short animation; pressing again
redirects it immediately. Display -> Page transitions also turns this motion
off. The screenshot stays visible while resizing when it is already loaded.

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

For arcade games, Details shows a warning if a required game ROM archive is
missing. Start checks again and shows the missing archive name instead of
handing the game to MiSTer. This follows the selected version's requirements
and MiSTer's storage search order. It checks archive presence, not the files
inside it: a present archive can still be incomplete or incompatible. The card
status describes installation and core version, not a guarantee of playability.

## Options

Options is in five groups, each under a grey heading with a small mark and
a rule and an expansion arrow: Data (a diskette: Refresh data now, Run Update All, Last update
result, Rescan card, Prefetch shots, Clear image cache), List (stacked
lines: Sources, Filter by rotation, Views (with Remember last view and its
conditional Default view beneath it), Title font,
Art type, Date format, Layout), Display (a monitor: rotation,
screensaver delay, style, brightness and info, safe zone, HDMI picture), Controls
(an arcade stick: Button labels, OK button, Menu button, Scroll speed, Hold
delay) and Operation (sliders: the main menu shortcut, Open at boot, Return
after game). Troubleshooting, Credits and Quit sit below the sections and
are never hidden by collapsing one. Only Data starts expanded. Select a
heading and press A to open or close it, or Right to open and Left to close.
Several sections can stay open. MisterZine remembers their state during the
session, including trips through Filters and subpages; restarting restores
the default. Left/Right on a setting still changes its value.
A row that only applies given the row above it (Rotation under Follow INI
rotation, the screensaver's style, brightness and info, Open at boot and Return
after game under the shortcut) sits a step in behind a small branch mark.
Values line up in one column; a small arrow at the right
edge of the top or bottom row shows when more options sit above or below the
visible part of the list. The legend in the bottom bar follows the selected
row: Change with the arrow that can still move on a choice row, A with what
it does where A acts (Preview on the screensaver rows, Open, Edit, Run,
Refresh, Rescan, Clear, Quit), and only Back on a greyed row.
The framed text under the list describes the selected
option. The build version, the catalogue date and the last check are grey
rows after Quit MisterZine, at the end of the list. B
(or the Menu button) returns to the screen Options was opened over: the
list, or Details, the artwork or Filters when the Menu button opened it
there, Filters as you left it. The next B reopens Options on the row you
left, the way the list keeps its row; Filters opened from the list always
starts at its top. Held
scrolling stops at either end. Release and press Up again from the first item
to wrap to Quit, or Down from Quit to wrap to the first item. L and R jump to
the heading of the previous or next section (Data, List, Display, Controls,
Operation), stopping at either end; keyboard Home/End still go to the first
and last row. Rows marked ↗ open another screen or editor.

- **Prefetch shots:** downloads the full picture set in the background.
  Off downloads pictures as you browse. The tally shows cached pictures.
- **Clear image cache:** removes downloaded pictures, which return as needed.
- **Show deprecated cores:** off by default. Turn it on to include cores marked
  deprecated in the catalogue. This choice is remembered and is unaffected by
  Clear all filters. Hidden cores keep their saved favourites.
- **Show non-arcade cores:** off by default, including after upgrading an older
  installation. Turn on to include console, computer and other cores. This
  preference applies to every browsing view, search, Favorites, catalogue counts
  and refresh announcements. Hidden favorites and Type selections are kept;
  clearing filters does not change this preference. Upgraded installations show
  a one-time explanation. Hold A for two seconds to continue; the progress line
  fills while held. Releasing early resets it.
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
- **Remember last view:** a child of Views, on by default, including after
  upgrading. Reopens in the view you left. Turning it off reveals **Default
  view** one level further in: choose the enabled view used at startup.
  Core updated is the initial default; a disabled default falls back to the
  first enabled view. Changing either preference leaves the current view alone.
  The default applies on opening MisterZine, including at boot and when it
  reopens after a game or a restart. Returning from Options, Details or Filters,
  waking the screensaver, and rescanning keep the current view. Return after
  game also tries to select the game just played within the starting view.
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
- **Art type:** which screenshot the list pane shows for the selected row:
  gameplay (default) or the title screen. Details and the artwork view still
  show every shot. Hold Select and press X on the list to switch it there.
- **Layout:** List (default), Split, Picture or Text, as described under
  Layouts above. The help box draws the four at the current orientation
  with the chosen one framed. Hold Select and press Y on the list to cycle
  them there.
- **Date format:** how list dates read: MM-DD (default), DD-MM, Mon D (Sep 7),
  D Mon (7 Sep) or YYMMDD (260907). The help line shows today's date in the
  chosen format. Rows from earlier years show the year alone, except with
  YYMMDD, which always carries the full date.
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
Screensaver settings are under **Options → Screensaver**. Preview starts it immediately, even with delay off; waking returns here.

- **Screensaver delay:** after 1 minute idle by default, blur and dim the picture
  into a soft glow over a second and scroll full-height black MISTERZINE
  lettering, with glinting chrome edges, across it; a key brings the picture
  back sharp in a quarter of a second. Left/Right chooses Off, 1, 2,
  5 or 10 minutes. A previews it immediately, even when Off. The first browsing
  button or typing key wakes without acting; release it before pressing again.
  The pad's Menu button opens Options (or quits, per Options -> Menu
  button; held 2 s it quits either way). The sweep covers the whole picture,
  including safe-zone margins, in horizontal and tate layouts. Background data
  downloads and Update All continue while dimmed.
- **Screensaver enabled:** switch automatic activation on or off separately from Delay (1, 2, 5 or 10 minutes). Disabling keeps your chosen delay and style; preview still works.
- **Dim screensaver:** choose Dim as the third style to darken the current screen without animation. Brightness uses three touching boxes: one filled means 33% (default), two filled means 66% of the original brightness. Press A to preview.
- **Screensaver style:** what the screensaver shows. Lettering (default) is the
  MISTERZINE sweep above. Screenshots shows one arcade game's gameplay
  shot after another, every arcade game in the catalogue in a shuffled
  cycle, each new one wiping in from the side behind a soft, wandering
  edge about every twelve seconds. Once a shot is in, what the main view's
  pane says about the game (title on one line, cut short with an ellipsis
  when long, card answer, kind, core, year and maker, rotation, players,
  controls, badges) is typed out in a bottom
  corner, the other one each time, a character a frame on black strips
  that grow with the letters, behind an underline cursor that blinks
  once the block is complete. Online, a shot the card lacks is
  downloaded when its turn comes; offline, only the shots on the card show
  (Prefetch shots fills the card). Hold Start for 2 seconds on a shot to
  play that game: the picture comes up to full brightness and a line
  fills under the title; let go early and it fades back and stays a while
  longer. A hold that catches a wipe half way turns the wipe back, since
  the game you meant is the one you were looking at, and the shot on its
  way comes next once you let go. For a game that is not on the card the
  hold only says so. Every other button wakes. With no shots to show at
  all, the lettering runs instead.
- **Screensaver brightness:** with the screenshots only. Two touching boxes show the level: one filled for half, both filled for full. Half (default) shows the
  shots at half brightness, kinder to a CRT, and holding Start brings one
  up to full; full shows them at full brightness throughout.
- **Screensaver info:** with the screenshots only. Full (default) types out
  what the pane says about the shot's game; title only types just the
  title; none leaves the picture alone. Holding Start still shows the
  title over its filling line whichever is set.
- **Only on card:** screenshots of games found on the card, including older versions.
- **Match rotation:** games matching the current horizontal or tate orientation; unknown orientations are excluded.
- **Favorites only:** screenshots of starred games.
- **Resolution:** All, an original game resolution category (such as 15 kHz), or Unknown. Independent of list filters.

Brightness, Info and these four filters appear only with Screenshots selected.
Their values are remembered when switching styles. Filters combine; if no
screenshots match, lettering runs instead and the Preview help explains why.

- **Safe zone:** opens the safe-zone editor. D-pad moves the top-right corner of the safe frame;
  Right/Up grows it and Left/Down shrinks it. B saves. Margins can reach 40 px.
- **HDMI picture:** Full display, 320x240, or Fit 4:3. **Full display** is
  the default for new installations; existing installations keep their saved
  choice. It uses both HDMI dimensions with square integer-scaled pixels:
  480x270 on 1080p, or 270x480 as viewed in tate. **320x240** keeps the classic
  canvas size. **Fit 4:3** keeps a 4:3 picture, sized to the HDMI height:
  360x270 on 1080p, 340x256 on 1024x768, or 400x300 on 1600x900.
  Changes apply live after a brief pause, keeping your place in Options.
  Full display gives tate more rows and horizontal layouts more width;
  screensavers use the expanded canvas too. On widescreen horizontal displays,
  Picture puts a narrow title list on the left, full-height artwork in the
  middle and information on the far right. The narrow list keeps favorite
  stars and omits status/date columns. On 4:3 it retains the stacked arrangement.
  Tall tate Split and Picture layouts reserve the full caption area below
  horizontal art. Low-resolution CRT modes retain their existing sizing.
  Unusual video timings that cannot fit a bounded canvas fall back to Fit 4:3.
  If a live switch fails, MisterZine attempts to restore the previous picture.
  Safe-zone margins still apply.
- **Button labels:** how the legends name the pad's face buttons, in MiSTer's
  A B X Y order: A B X Y (default), B A Y X for an Xbox-lettered pad mapped by
  position, the PlayStation circle, cross, triangle and square by position, or
  1 2 3 4 in MiSTer's define order. Only the names change. Which physical
  button MiSTer calls A, B, X or Y comes from its define buttons screen; this
  guide uses those names.
- **OK button:** which face button confirms, per pad. Auto from MiSTer
  (default) follows the MENU OK you gave MiSTer's define screen: on a pad
  whose MENU OK is B, B confirms and A goes back, and the legends name the
  buttons that way round. A pad MiSTer has not defined reads Auto from
  Linux: its bottom button confirms. A or B overrides it for the pad shown. The row
  follows the pad that last pressed a button, so change it with that pad in
  hand; the hint names it. It is muted, with the reason, before any pad has
  pressed, for a pad that comes through MiSTer's translation, and for one
  whose A and B are the same button.
- **Menu button:** what the pad button you defined as MiSTer's menu (OSD)
  button does in MisterZine, which holds a defined pad while it runs: Options
  (default) from any screen, closing it when it is up, or quit MisterZine
  for the MiSTer menu, as earlier versions did. With Options chosen, holding
  the button for two seconds quits as well, so the way out is always there.
  Keyboard F12 and the board's button still quit.
- **Scroll speed:** 20, 30 or 60 rows/pages per second once a hold repeats.
- **Smooth scrolling:** shown beneath Scroll speed at 20 or 30 Hz. On (default)
  adds intermediate list movement; off moves whole rows. Both choices animate
  in the preview. Releasing finishes the current step. The choice is saved
  even when 60 Hz hides it.
- **Hold delay:** short 200 ms, normal 300 ms (default), or long 500 ms before
  navigation repeats. Artwork and calibration retain their own timing.
- **Main menu shortcut:** off removes the menu entry and boot hook. If the
  launcher opened this session, it returns to Menu before stopping. Turning
  the option back on puts both back. With it off, open MisterZine through
  Scripts -> MisterZine-Run (or run Setup again).
- **Open at boot:** off by default. On: once the MiSTer menu is up after a
  power-on or reboot, the launcher picks the MisterZine entry for you, the same
  way you would. A `bootcore` in the INI takes precedence and nothing happens.
  Needs the Main menu shortcut.
- **Return after game:** off by default. On: when a game you started from
  MisterZine exits to the MiSTer menu (Reset or Exit in the OSD), the launcher
  picks the MisterZine entry again and the list opens on that game. Games
  loaded some other way, and quitting through the Menu button, do not bring
  it back. Needs the Main menu shortcut.
- **Troubleshooting:** a guided Start-button test, a pad tester, and an
  A/Enter game-launch test, with results you can photograph. The last result
  survives restarting. The pad tester lists every connected pad with the raw
  button code in each MiSTer slot and which button is its OK button, then
  shows each press as it happens: which pad, which button, which slot and
  what MisterZine does with it, plus the gap since the previous press. Hold
  the back button for two seconds to leave it.
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
the legend reads "A ▶ Open" or "A ◀ Close" to match the selected heading.
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

The latest-update view's last-look line is a timeline marker for your previous
visit: it sits under the last entry that shipped since then, so everything above
it is new to you. When nothing in the view is new it sits at the top and says
**Nothing new since your last look**. An entry the catalogue added late under an
old date still shows its own unseen mark below the line, and the Since last look
filter finds it, but it does not move the line.
On-card choices include counts across the enabled catalogue, independent of the
current search and other filters. Favorites mode always shows favorites only;
use Y in the main view to leave it.

Card scans run quietly on launch. Options -> Rescan card shows a result screen
with up-to-date, older, undated, missing and unknown totals for the enabled
catalogue. B closes it even
while scanning; completion does not interrupt the screen you moved to.
Unreadable card scans retain the previous inventory and show a failure banner.

MisterZine checks GitHub for a newer stable app release on launch and at most
every 30 minutes. When available, the main bar shows **App update**, and Options
shows the version beside the Update All instructions. Run Update All, then quit
and reopen MisterZine to use the new app. Offline checks stay quiet. This is
separate from automatically refreshing the games catalogue; prerelease and
development builds do not advertise stable downgrades.
By default, **Type** offers Stable and Beta directly and the **Game filters**
section contains Rotation, Resolution, Genre, Controls, Buttons and Players.
Enable **Options > List > Show non-arcade cores** to restore the broader Type
choices and the **Arcade game filters** heading.

With non-arcade cores enabled, the **Arcade game filters** section contains Rotation, Resolution, Genre,
Controls, Buttons and Players. These choices affect arcade games only; Type
controls whether console, computer and other cores appear. Excluding Arcade
hides its filter section while retaining your choices for later. Under Type,
Arcade opens (Right, or X) into Stable and Beta, the Patreon beta cores, like
a decade opens into years: A toggles one, Y shows only one, and a mixed choice
shows `[-]` on Arcade. Turning both off is the same as turning Arcade off.

Counts reflect your search and the other filter sections. A choice's count
ignores its own section, so unchecked choices still show how many entries they
could include. Choices remain selectable at zero. Y on a value twice enables
every choice in its section; Clear all filters restores the ordinary filter choices within your enabled
catalogue; it does not change Options preferences.

Resolution uses the catalogue's 15kHz/31kHz labels. **Unknown** means that value
is missing. Controls includes separately recorded special controls such as
spinners, paddles and trackballs. A button count alone does not identify a
buttons-only game. **0 buttons** requires an explicit zero in the catalogue;
older feeds that omitted zero counts still show those entries as Unknown.

Provisional values are supplied from fallback sources pending the curated
Arcade Database. They match their ordinary filter categories and are marked
"(provisional)" in the game's details.

## Update All

Catalogue visibility does not change what Update All updates. It continues to
use your normal updater configuration.

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
