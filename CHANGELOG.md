# Changelog

## Unreleased

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
