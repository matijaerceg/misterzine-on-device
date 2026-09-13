# Changelog

## v1.0.23 — 2026-09-13

- The A-Z and Year views gain header lines like Maker: a line before every
  letter and every release year, numbers and symbols first under "0-9 and
  symbols" and the unknown years last under "Year unknown". L/R jump from
  group to group with the row centred under its header and no status
  notice, and a group whose header has scrolled off keeps its name pinned
  on the top line until its last row is gone. The Year view drops its date
  column, since the header names the year: the title takes the room and
  the card status glyph stays at the right.
- rmCores (rmonic79) is a known source: the list, Details and Filters name
  it rmCores instead of the raw id, Options -> Sources: installed only can
  hide it when the card's downloader.ini has no `[rmonic79/rmcores]`
  section, and Details explains under Source that the rm build adds CRT
  Adjust and V-Size on the analog output and a pause overlay to a game the
  MiSTer Distribution also ships.
- Alternatives are read from the `_alternatives` folder inside a database's
  own folder as well as from `_Arcade/_alternatives`, so the MeatCores and
  rmCores alternatives reach the Details version selector. A database that
  keeps to that layout is covered without being named.
- Options -> Credits: a page naming the developer, the projects MisterZine
  is built on and, under Special thanks, the early adopters who tested it
  and sent feedback.
- The Views page loses its top line, and pressing A on the greyed last view
  says "Keep at least one view on" instead of doing nothing.
- The embedded first-run catalogue is refreshed to the 2026-09-12 feed.

## v1.0.22 — 2026-09-12

- A Maker view joins the Y cycle after A-Z: the games grouped by
  manufacturer under header lines, makers A-Z with titles A-Z under each,
  the original year in the date column and Unknown maker last. L/R jump from
  maker to maker with the row centred, and a maker whose header has scrolled
  off keeps its name pinned on the top line until its last row is gone.
  Spellings are merged: a credit counts under the first company named and
  corporate or regional suffixes are ignored, so Taito Corporation Japan
  sits under Taito and Data East USA under Data East; Details still shows
  the full credit.
- Options -> Views replaces the Recents view switch: a checkbox page that
  chooses which of the seven views Y cycles through. Everything is on except
  Recents by default, the last view on cannot be turned off, and turning off
  the view you are in moves the list to the next one. The choice is saved as
  `views_off` in settings.json; an existing Recents view setting carries
  over.
- With Options -> Menu button set to Options, holding the pad's Menu button
  for two seconds leaves MisterZine, so the way out is always there without
  changing the setting. A hint appears after half a second and releasing
  earlier stays; the short press still opens or closes Options at once.
- Closing Options with B or the Menu button returns to the screen it was
  opened over: the list, Details, the artwork, or Filters with its browsing
  state kept, also after a trip through Troubleshooting or a scan.

## v1.0.21 — 2026-09-12

- A `[MisterZine]` section in MiSTer.ini, placed below `[Menu]`, changes
  `direct_video`, `vga_scaler` and `osd_rotate` only while MisterZine is open,
  because Main applies sections named after the MGL's setname as well as the
  core's own name. That lets an HDMI display keep the menu, terminal and
  Update All while the CRT shows MisterZine alone. Follow INI rotation and
  the CRT-only warning now read that section the way Main does, in file
  order, instead of `[Menu]` alone; the first-run notice names both sections.
  Troubleshooting documents the layout, the ordering rule and that the
  Scripts entries keep the `[Menu]` output.

## v1.0.20 — 2026-09-12

- Degauss, installed as MiSTer's `main=` frontend, takes over every load of
  the menu core, MisterZine's menu entry included: choosing MisterZine
  brought up Degauss and left the launcher waiting for a console it could
  never get until the next reboot. The launcher now closes Degauss for the
  MisterZine session and the usual menu restore brings it back when
  MisterZine exits; Return after game and Open at boot work the same way.
- The launcher's console switch waits at most two seconds and reports the
  failure instead of waiting for the rest of the boot.
- Scripts gains MisterZine-Run, which opens MisterZine from a Scripts list
  (Degauss lists Scripts, not MisterZine's menu entry); the frontend returns
  when MisterZine quits. Opened that way, leave through Options -> Quit
  MisterZine or the pad's Menu button, as keyboard F12 stays with the Scripts
  session.

## v1.0.19 — 2026-09-12

- An alternative MRA that holds no readable header (empty, cut short, no
  `<rbf>`) no longer marks the card scan incomplete: it is skipped, named in
  the log once, and counted on every scan. MRAs with `--` inside a comment
  (Seibu SPI sets) are now read. "Card scan incomplete" is kept for a folder
  or file that could not be read, and the result screen then keeps its
  totals with the problem under them. watch.log rotates at 1 MiB like
  log.txt.

## v1.0.18 — 2026-09-12

- Options -> Canvas, fit display by default: where MiSTer's integer scaling
  of a 320x240 picture leaves bars above and below, the picture is sized to
  fill the screen height instead: 360x270 on 1080p, 340x256 on 1024x768,
  400x300 on 1600x900. Text keeps its size and the list gains rows. CRT
  modes, 720p and 1440p already fit and stay 320x240; 320x240 keeps the
  classic size on every display. Applies when MisterZine next starts.
- A pad defined in MiSTer whose A and B MisterZine can read is held while
  MisterZine runs, so MiSTer sees nothing from it, and the button defined as
  MiSTer's menu (OSD) button becomes MisterZine's Menu button: Options from
  any screen by default, or leave to the MiSTer menu (Options -> Menu
  button). A pad whose A or B cannot be read here stays on MiSTer's
  translation, as before v1.0.17, so a partial definition can no longer
  lose its back button. Keyboard F12 and the board's button still leave.
- Hold Select on the list: Y cycles List layout, X switches List shots; the
  legend names the chords while Select is held.
- The main list legend reads A B X Y; the pad tester shows the Menu button
  and whether a pad is held.

## v1.0.17 — 2026-09-12

- A pad defined in the MiSTer menu is read by MisterZine directly, slot by
  slot: D-pad, A B X Y, L R, Start and the menu stick, with trigger and
  stick edges counted as MiSTer counts them. The button you defined as A is
  A here whichever buttons you gave the MiSTer menu's own OK and back, and
  each pad follows its own definition, so two pads defined differently work
  side by side. A pad never defined in MiSTer keeps working through MiSTer's
  translation while no defined pad is connected.
- Options -> Button labels: how the legends name the face buttons, in
  MiSTer's A B X Y order: A B X Y (default), B A Y X for an Xbox-lettered
  pad mapped by position, PlayStation symbols by position, or 1 2 3 4 in
  MiSTer's define order. Only the names change.
- Troubleshooting -> Test pad buttons: lists every pad being read with the
  raw code in each MiSTer slot and where it came from, then shows each press
  as it happens: which pad, which button or axis, which slot, what MisterZine
  does with it, and the gap since the previous press. Hold B for two seconds
  to leave.

## v1.0.16 — 2026-09-12

- Options -> Return after game (off by default): when a game started from
  MisterZine exits to the MiSTer menu (Reset or Exit in the OSD), the menu
  launcher picks the MisterZine entry again and the list opens on that game.
  Games started from the MiSTer menu itself, and leaving MisterZine with the
  pad's Menu button, do not bring it back. Needs the Main menu launcher.
- Options -> Open at boot (off by default): once the MiSTer menu is up after
  a power-on or reboot, the launcher picks the MisterZine entry for you. A
  bootcore in the INI takes precedence. Needs the Main menu launcher.
- Each list order is sorted once per loaded catalogue and reused, and the
  Year order no longer re-parses years while sorting, so Y presses, search
  keystrokes and card-status updates skip the 6-20 ms sort on the ARM core.

## v1.0.15 — 2026-09-12

- Options -> Recents view (off by default): on adds Recents to the Y cycle
  after Favorites, listing the games launched from MisterZine, latest launch
  first, each once, with the launch date in the date column and L/R jumping
  between launch months. Every launch from the list, Details or the artwork
  view is recorded in state.json (up to 100) whether the view is on or off;
  MiSTer's own recent lists are untouched.
- Options -> List layout: list (default), split (a pane under half the
  screen wide with a bigger picture and shorter rows) or picture (the
  picture across the screen with a few full-width rows: above them in
  horizontal, with the details beside it; below them in tate). A vertical
  shot keeps its shape with the details beside it when there is room.
- B reopens Options on the row it was closed from, the way the list keeps
  its row between visits; the first visit still starts at the top.

## v1.0.14 — 2026-09-11

- Options -> Sources: all (default) or installed only, which shows games only
  from the databases in the card's downloader.ini, read at every card scan,
  so sources you never enabled (Coin-Op, Meathax, Jotego) leave the list,
  search, Favorites, the release count and the Filters panel. Without a
  downloader.ini nothing is hidden and the help text says so.
- Options is regrouped as Data (Refresh data now, Run Update All, Last update
  result, Rescan card, Prefetch shots, Clear image cache), List (Sources,
  Filter by rotation, Remember sort order, Title font, List shots, Date
  format), Display (rotation, screensaver, safe zone) and Operation (scroll
  speed, hold delay, the launcher, Troubleshooting, Quit).

## v1.0.13 — 2026-09-11

- List titles are drawn in a narrow font (scientifica, SIL Open Font License)
  with proportional spacing, so a row fits about a third more of a title. The
  default, "narrow tall", is that font made one pixel taller to match the body
  font's capital and lowercase heights; Options -> Title font also offers the
  original "narrow" face and "normal", the body font. The status and date
  columns use the narrow font at the titles' height.
- Patreon beta cores carry a yellow β after their title in the list, and the
  pane's "beta" chip is yellow to match the details page.
- Options -> Date format chooses how list dates read: MM-DD (default), DD-MM,
  Mon D, D Mon or YYMMDD. Rows from earlier years still show the year alone,
  except YYMMDD, which always carries the full date.
- Options -> List shots chooses whether the pane thumbnail is a gameplay shot
  (default) or the title screen.
- The version chosen in Details is remembered per game, across restarts:
  Details opens on it next time and Start launches it from the list and from
  artwork. Choosing the main version again forgets the choice; an alternative
  no longer on the card falls back to the main version.
- A Details version name too long for its line scrolls back and forth by the
  pixel, pausing briefly at each end, so alternatives that share a long prefix
  can be told apart.
- Details swaps its keys: Left/Right choose the version and Up/Down page the
  information, which now slides by the pixel (two pixels a frame) instead of
  jumping. The information scrollbar is a green thumb alone, like the list's.
- Left/Right in the main list move a screen of rows at a time and keep the
  selected row centered, the same as stepping.
- Options values start at the half mark of the list rather than two thirds.
- The top bar names the order as core updated, MiSTer debut, original year,
  A-Z or Favorites A-Z (favorites are alphabetical).
- A Year sort order, after Debut in the Y cycle: newest original release year
  first, titles alphabetical within a year, unknown years last; L/R jump
  between years and the date column shows the year.
- Options is grouped under grey Data:, Display: and Operation: headings, with
  half-height gaps between the groups (Filters' blank rows are half height too). The list scrolls with the selected row
  kept near the center, like the main list and Filters, and the edge arrows
  show only while entries are actually out of view.
- Filters: Type > Arcade opens into Stable and Beta children, like a decade
  into years, so the Patreon beta cores can be hidden or shown alone.
- The Details information slide is three times faster (six pixels a frame),
  and the legend mentions Left/Right only when a game has more than one
  version and Up/Down only when the information scrolls.
- Filters: A on a section heading opens or closes it, like Left/Right; the
  legend says so while a heading is selected. Headings no longer toggle every
  value of their section (Y only/all still does).
- Filters always lists Clear all filters at the top, greyed out until a filter
  is active, with a blank line below it; the page opens on the first heading
  while the row is greyed out, and pressing it moves on to the first heading.
- Options: the list is unframed and the help text framed instead, with the
  build version and catalogue date in grey just above the hint bar. Values
  line up in one column at the two-thirds mark where the labels allow (the
  narrow tate layout keeps them right-aligned). A small arrow at the right
  edge of the top or bottom row shows when more entries sit above or below,
  in Options and Filters alike.

## v1.0.12 — 2026-09-11

- The screensaver letters are now traced from Figtree Black (SIL Open Font
  License; only the outlines ship) instead of hand-drawn block capitals, with
  a generator script for re-tracing from any font.
- Three more glints on the screensaver outline: a faded green one that the
  left-facing edges meet twice and a gray one for the right-facing edges, both
  dim and fast, running against their face's main glint for depth. The lime
  glint stands more upright, so it travels faster, and is slightly dimmer.

## v1.0.11 — 2026-09-11

- Follow INI rotation reads the MiSTer INI that is actually active: the main
  MiSTer.ini or the alternative `MiSTer_*.ini` selected in the MiSTer OSD,
  resolved the same way Main resolves it. Previously only MiSTer.ini was read,
  so an `osd_rotate` set in an alternative INI was ignored.
- Options orders the rotation rows as Follow INI rotation, Rotation, then the
  filter. Rotation is greyed out and locked while following is on, and becomes
  the saved manual choice once following is off.
- “Filter by INI rotation” is now “Filter by current rotation”: it follows the
  interface's orientation whatever set it, INI or manual, and updates as soon as
  the rotation changes. The saved setting carries over.

## v1.0.10 — 2026-09-11

- The screensaver lettering carries a one-pixel chrome edge. Two still lights
  stand on the screen, violet for left-facing edges and lime for right-facing
  ones; as the letters scroll through them the outline glints, sliding along
  each edge with the lettering's own motion, and stays black elsewhere.

## v1.0.9 — 2026-09-11

- Card status compares against the shipped core build itself (the catalogue's
  new build date and checksum fields) instead of the catalogue's last-updated
  date, so an MRA-only fix or a date rollover no longer shows a current core
  as an older build.
- Undated cores such as Jotego's are matched by checksum, using the Update All
  record on the card. A match shows as current; a mismatch shows as “older
  build likely” and counts with older builds in the card summary and filters.
- Arcade games only match cores in `_Arcade/cores` and system entries only
  their own folders, so a core name shared between the two (Astrocade) no
  longer reports the wrong file.

## v1.0.8 — 2026-09-11

- Filter arcade games by original release year or decade. Expand decades into
  individual years; A toggles and Y selects only/all. Choices are saved.
- Filters opens with only active sections expanded. Disclosure arrows show
  open/closed sections and decades; an asterisk marks non-default filters.
- Left/Right collapses/expands filter sections and decades; L/R jumps between
  sections. X also toggles decade expansion.
- Main-list and Filters row scrolling follows the selection near the center,
  stopping at the top and bottom. Letter/month jumps retain top alignment.
- Clearer filter styling and control hints, with a neutral selected-row color.
  Removed redundant navigation hints and the separate Favorites filter section;
  Favorites mode still supports all other filters. Existing Favorites-only
  filter settings migrate to the Favorites main-view mode.

## v1.0.7 — 2026-09-11

- Optional strict INI rotation filter: show only known matching orientations,
  excluding unknown and opposite orientations, including in Favorites. Defaults
  off; turning it off restores manual filters. Independent of UI rotation.

- Follow MiSTer.ini rotation at every startup by default, with an Options toggle.
  Manual session rotation remains available; turning following off retains it.
- Y toggles isolation of the highlighted filter value within its section,
  enabling every value in that section on a second press. The legend
  advertises the shortcut. A retains individual toggling.
- Last-look messages distinguish an unchanged top-200 window from an unchanged
  complete view, instead of claiming nothing new across the whole catalogue.

## v1.0.6 — 2026-09-10

- Quiet automatic card scans; manual Rescan opens a readable result screen.
  Card-status filters show catalogue-wide counts. Scan failures retain a banner.
- Main view quietly indicates when a newer stable MisterZine app is available;
  Options identifies the version and offers Update All. Reopen after updating.
- Favorites is a fourth main-view mode after Alphabetical, with alphabetical
  ordering, held letter jumps, search/filter support and saved view restoration.

## v1.0.5 — 2026-09-10

- Remember the last selected sort order by default. Options -> Remember sort
  order can restore the previous behavior of starting each visit with latest updates.
- Hold Up/Down or L/R to scroll/page the Update All log using the configured
  hold delay and scroll speed, including saved results.
- Start launches the selected version from full-screen artwork. Launch failures
  appear over the picture.
- L/R jumps between letters in alphabetical order, using the current search and
  filters. Each jump puts the letter's first title at the top of the view in both
  directions, including short groups at the end of the list.
- L/R jumps between months in latest-update and MiSTer-debut order. Empty months
  are skipped and the selected month/year is shown briefly. Home/End retains
  first/last jumps in every sort order.
- Holding L/R repeats letter/month jumps with the same hold delay and scroll
  speed as Up/Down row scrolling.
- Refreshed the bundled catalogue for offline first use.

## v1.0.4 — 2026-09-10

- Fixed launching with arcade USB encoders whose Start button uses a different
  Linux button code. Start now follows the controller's saved global MiSTer
  assignment; the standard Linux Start button remains the fallback without a map.
- Added Options -> Troubleshooting with a guided Start-button test. It detects
  raw controller signals even from devices the normal input reader skips, and
  freezes a paged result with device IDs, button codes and press/release counts.
- Added an A/Enter launch test for the highlighted game's main version, with
  saved failure details and a clear distinction between sending a launch command
  and confirming that the game started.
- Troubleshooting results survive restarting and can be shared with a photo.
  Tests do not change mappings, enable remote debugging or upload information;
  extra input observation stops when the test ends.
- Refreshed the bundled catalogue for offline first use.

## v1.0.3 — 2026-09-10

- Added alphabetical title sorting as the third Y sort choice.
- Grouped game-specific filters under Arcade game filters, excluding system
  cores from their counts and effects. Counts now reflect other filters and
  search, including zero-count choices.
- Controls includes separately recorded special controls; explicit zero-button
  counts are separate from Unknown. Provisional values are labelled in Details.
- Added a remembered Resolution filter for 15kHz, 31kHz and unknown entries.

## v1.0.2 — 2026-09-09

- Setup and Uninstall scripts use hyphenated filenames so MiSTer's Scripts menu
  can launch them. Downloader removes the previous filenames during an update.
- The installer INI includes an empty MisterZine-specific filter so global core
  filters do not exclude the app. Existing installer files need this line added
  manually or replaced with the current download if affected.

## v1.0.1 — 2026-09-09

- Held scrolling stops at either end of Options. Wrapping requires releasing
  the direction and pressing again while already at the end.

## v1.0.0 — 2026-09-09

- Main-menu installation with one-time Setup and two removal choices: keep
  favorites/preferences or remove all MisterZine data.
- Keyboard title find and remembered filters; separate Filters and Options.
- A single-row version selector, wrapped information, and L/R paging in Details.
- Action-first Options, a shorter default hold delay, and separate delay/speed choices.
- Options navigation wraps from the first item to Quit and back.
- A one-minute idle screensaver with dimming, full-height black scrolling lettering,
  a selectable delay/Off, and an immediate preview from Options.
- Clearer unknown-build-date wording and HDMI/CRT setup guidance.
- Refined rotation, artwork controls, safe-zone layout and header styling.
- Supervised Update All with live output, protected-write cancellation handling,
  recovery checkpoints, reconnection and saved result review.
- Reliability fixes for launching and shutdown, favorites/settings preservation,
  data refresh and card scanning, screenshot downloads, and menu launcher updates.
- Framebuffer compatibility with the September 2026 Linux driver.
- Opt-in remote debugging; detailed input/performance instrumentation stays off
  in ordinary installs. F12 screenshots remain available.
- Split user/developer documentation, bundled license notices and checked release assets.

## v0.2.1

- The pad's menu button returns to the MiSTer menu cleanly: the app notices
  Main took the screen back and exits, so the launcher can reopen it.
- Every visit starts at the top of the updated sort, like the site.
- Screen view: Up/Down walk rows, Left/Right change the shot.
- Faster held scrolling; status bar says "1206 releases".

## v0.2.0

- Main menu launcher: a "misterzine" entry in the main menu, enabled once
  from Settings. A resident helper opens the app when the entry's MGL loads,
  and puts the plain menu back when you quit. The database ships the MGL.
- The app's input reader ignores the launcher's own virtual keyboard.

## v0.1.1

- The downloader database no longer lists the drop-in ini: downloader rejects
  root-level ini files from a database. It stays a release asset for manual use.

## v0.1.0

First release. Runs from the Scripts menu on a 320x240 framebuffer, horizontal
or tate, with the gamepad.

- List of every misterzine row sorted by last updated (Y toggles debut), with
  card status per row, a "your last look" divider and a side or bottom pane
  with thumbnail and specs.
- Details with all specs, three shots, alternatives, and launch (A twice).
- Screen view: full-bleed screenshots, walk rows with left/right.
- Filters: type, source, rotation, players, genre, card status, favorites,
  changed since last look. Settings: safe zone calibration, rotation,
  prefetch, rescan, refresh, cache clearing.
- Live data from misterzine.fyi with an offline cache and an embedded
  first-run snapshot.
