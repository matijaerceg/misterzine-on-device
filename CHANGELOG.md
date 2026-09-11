# Changelog

## Unreleased

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
- The Details information slide is three times faster (six pixels a frame).
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
