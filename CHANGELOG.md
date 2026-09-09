# Changelog

## v0.2.2

- Update All recognizes system-update warnings beyond the display's line
  limit and across output-buffer boundaries. Detection follows incoming text
  in order, independently of the shortened and redacted visible log.
- Update All flushes saved recovery checkpoints before replacing the previous
  record. Card saves use a copied state without holding the output-reader lock.
- Update All checks for protected system work before both the initial cancel
  signal and a later force-stop. A newly detected write can finish first;
  the screen explains the wait, and known writers are not repeatedly paused.
- Scan messages expire while Update All is open, preventing an expired
  message from driving continuous full-screen redraws during or after a run.
- Returning from artwork keeps the selected game version and details scroll
  position, so Start still launches the version picked before viewing a shot.
- Turning the main-menu launcher off keeps the current session's helper
  alive until it returns to Menu, then stops it. Off removes the menu entry;
  checking launcher status no longer recreates it. Boot-hook changes use a
  synced replacement file and preserve the script's existing permissions.
- An unreadable favorites file is preserved on quit and later launches;
  the app reports the failure and disables favorite edits for that session.
  Options now spells the quit action "Quit MisterZine".
- Image workers stop even with pictures still queued, preventing a pending
  download or unreadable cached image from trapping shutdown before a launch.
- Updated sort matches the site's arrival batches: later same-day refreshes
  come first, then core and title. Debut order and older feeds are unchanged.
- Filters: smaller font, Input directions and Buttons sections, and Players
  last. Options keeps the version and data timestamp visible in either
  orientation; horizontal help uses three lines. Rescan follows Update All.
- Options: Run Update All opens a dedicated stage bar and live-log screen.
  Normal navigation is blocked; hold B for two seconds to request
  cancellation, with system writes allowed to finish. An early restart
  ribbon honours Update All's existing reboot policy. A detached supervisor
  retains output and lets the app reconnect after closing; completion
  triggers a card rescan. External updater launches are guarded.
- Main list: X opens Filters; B opens Options (formerly Settings). Both
  screens return directly to the list with B. Filters starts with On the
  card, Favorites and Since last look, followed by the existing facets.
- Compatibility with the September 2026 Linux framebuffer driver: use the
  driver's reported pixel memory when its normal mmap operation is absent.

- List: titles start one column further left; the release count is muted
  beside "by: latest update"; a one-cell ellipsis replaces ".." everywhere;
  the "+" glyph is lighter so "filters+settings" reads cleanly.
- Horizontal pane: picture and text share a left edge 3 px in from the
  separator; horizontal games fill the 4:3 box like the site, vertical ones
  keep their shape. In tate the pane picture is centred in its box.
- Details strip: boxes hug the pictures and pack from the left in both
  orientations.
- Start launches: from the list it launches the main version, in details the
  picked version. A opens details from the list and the shots from details.
  (Start is read from the pad itself, since Main does not pass it on.)
- Held scrolling waits half a second, then runs at full speed at once.
- Clock: while the system clock is still at 1970 (no NTP yet), the app takes
  the time from the site's Date header so relative dates are right.
- Main menu entry: a press made before the watcher was up after a cold boot
  is honoured; every fresh selection counts, keyed by CORENAME's mtime.
- Safe zone editor: the d-pad nudges the top-right corner of the frame
  (Right and Up grow it, Left and Down shrink it); an arrow marks the corner.
- Launching a game from the main menu entry no longer risks the menu
  watcher putting the menu back over the game.
- Quitting or launching cuts any screenshot download short instead of
  waiting for it (a slow download could hold up a launch).
- A failed screenshot download waits before it is retried (15 s, doubling
  to 8 min) instead of retrying at once.
- The cached data only advances its hash once the data itself is on disk;
  a malformed hash from the site is an error, not a crash; a card scan that
  finished after a data refresh no longer attaches statuses to the wrong rows.
- Settings from before the two-axis safe zone keep their inset on both axes.
- The main menu entry is "MisterZine" (the MGL is MisterZine.mgl; an older
  lowercase file is renamed on the next boot or launcher start).
- Left/Right page the list like Main's menu. Shots open from details with X
  and go back to details; details still walk rows with Left/Right.
- Held scrolling runs at the framebuffer's pace: one row per frame at most,
  normal 20 / fast 30 / turbo 60 rows a second, and nothing else (state
  saves, network checks) runs between two frames. Left/Right page at the
  same pace. A scrollbar beside the list.
- L and R jump to the top and bottom.
- The pane pictures of the next 16 rows (and the previous 6) are fetched
  and decoded ahead while you browse, nearest first; only a key that has
  started repeating pauses that work, a tap does not.
- Screen view uses the whole screen: a shot up to 15% larger than the
  screen shows pixel for pixel, cropped at the edges; only much bigger ones
  are scaled down. (A 240p shot on a 240p screen is untouched.),
  with only the slot count overlaid in a corner. Pictures in the list pane
  and details draw without frames; missing ones are a black shape with a
  word on it. Details lose the status bar and the batch chip; the launch
  list is headed "Versions".
- Safe zone editor: B saves and goes back (no A); Left/Right are the sides
  as viewed, Up/Down the top and bottom, in both orientations.
- Filters: Left/Right page, L/R first and last, only B closes.
- Settings: Left/Right change a value (arrows show which way can still
  move); scroll speed reads 20 / 30 / 60 Hz and defaults to 30; "Edit safe
  zone" sets the side margins with Left/Right and the top and bottom ones
  with Up/Down (two settings, default 15 px); the prefetch tally counts up
  live and drops to zero on clear; the Input test is gone. Settings
  repaints no longer stat every screenshot on the card.
- Details ignore A for half a second after opening, so a double tap in the
  list cannot launch.
- Screenshot decoding and downloads pause while a key is held; the menu
  button probe runs off the drawing loop (it was costing a few frames a
  second).
- Details: Up/Down pick the launch target; row browsing from details is
  gone (go back to the list). No B legend.
- "no connection" instead of "offline", and no label at all in the first
  seconds after a cold boot while the clock is still unset (the check
  retries every 15 s until it is).
- Pane card status fits its column: "current build", "older: 20240601",
  "on card, undated", "not on card". Date rules between rows are gone.
- Settings help gets four lines. A "MisterZine is loading" line shows on the
  console while the app starts, and the launcher warms the cache at boot.
- Scroll speed setting (normal / fast / turbo), help text under every
  setting, coloured button hints, Input test screen, Quit item; B never
  exits.

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
