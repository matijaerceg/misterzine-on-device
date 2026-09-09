# Changelog

## Unreleased — v1.0.0

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
