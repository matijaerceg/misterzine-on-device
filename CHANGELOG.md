# Changelog

## v0.2.2

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
