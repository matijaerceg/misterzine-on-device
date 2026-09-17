# Changelog

## Unreleased

## v1.0.40 — 2026-09-17

- Animate layout changes made with Select+Y in the main view, including the list and artwork regions and their divider, in both orientations. Keep the divider visible throughout each transition.
- Simplify Options: put Sources first in List and remove the Layout setting; Select+Y remains the way to change layouts.
- Rename Edit safe zone to Safe zone and mark rows that open another screen with an arrow: Views, Safe zone, Screensaver, Run Update All, Rescan card and Last Update All result.
- Indent Clear image cache and Last Update All result beneath their related options.

## v1.0.39 — 2026-09-16

- Smooth held list scrolling at 20 or 30 Hz with evenly spaced intermediate frames. Add a saved Smooth scrolling option below Scroll speed and animated comparisons; releasing finishes the current step without advancing the selection.
- Keep Options hints and previews at a fixed four-line height in both orientations, avoiding a layout jump when selecting Smooth scrolling.
- Add full-display HDMI layouts, apply picture changes live with framebuffer recovery, and improve widescreen artwork sizing and portrait captions.
- Offer to restart MisterZine after Update All replaces the running program.
- Add Penetron to the early-adopter credits.

- Order HDMI picture choices as Full display, 320x240 and Fit 4:3. New installations default to Full display; existing installations retain their saved picture mode.

- Make Options sections collapsible, with only Data expanded on startup. Match Filters' expansion arrows and controls, remember open sections during the session, and keep Troubleshooting, Credits and Quit outside the collapsible groups.

## v1.0.38 — 2026-09-15

- Remove the pause when navigating away from the Scroll speed and Hold delay previews in Options. Draw the new selection before leaving the animation loop.
- Prevent the screensaver from starting while Update All is active. Restart the normal idle countdown when the update finishes so its result stays visible first.
- Label the finished Update All screen's return action Back.

## v1.0.37 — 2026-09-15

- Restore missing arcade alternatives, including Galaga's Namco versions and hacks. Match explicit MAME game families and clone setnames from the catalogue, with installed MRA parent metadata as a fallback. Recognize directory-qualified HBMAME ZIP references while keeping the compatible-core requirement; shared BIOS files and folder names do not establish a game family. Thanks to ac3 for reporting the Galaga issue.
- Build the alternatives lookup during the background scan, preserve remembered launch paths, and discard results for an obsolete catalogue. Rebuild old alternative caches with parent metadata and notice in-place MRA changes. Read parent-only headers before large embedded ROM bodies.
- Clear routine data-age and checking text from the top bar. Options now shows the catalogue publication time and the last successful check this session, including checks that find no new data. Connection failures and app-update notices remain visible. Compact timestamps fit narrow displays; unknown times are labeled explicitly.
- Make screenshot fixtures independent of the host time zone and refresh the embedded catalogue snapshot.

## v1.0.36 — 2026-09-15

- Show missing arcade ROM archives in Details and check again before launching. Check every ROM section and the selected alternative, following MiSTer's storage search order. Archive contents remain unchecked. Launch logs now describe a request handed to MiSTer, rather than claiming the game loaded.

- Correct the MiSTercade/JAMMA setup instructions and log hint to use explicit arcade display timing, preventing rolling pictures on affected cabinets.

## v1.0.35 — 2026-09-15

- Fix: on a MiSTercade, choosing MisterZine flashed the cabinet monitor and returned to the menu. MiSTercade's shipped MiSTer.ini sets `direct_video=2`, Main's auto mode, which only switches direct video on when an HDMI DAC is attached; with nothing on HDMI the framebuffer stays there, so the cabinet never shows MisterZine or any other script. MisterZine treated 2 like 1 and kept quiet. It now counts only `vga_scaler=1` or `direct_video=1` as reaching the analog port, and with `direct_video=2` the first-run notice and the log warning name the cure: `vga_scaler=1` and `video_mode=320,240,60` under `[Menu]`, the section MiSTercade documents for seeing scripts. The troubleshooting guide has a JAMMA cabinets section with that configuration, confirmed on a MiSTercade v1, and the README points to it. Thanks to ctrain_1985 for reporting it.
- Refresh the embedded catalogue snapshot.

## v1.0.34 - 2026-09-14

- Allow the screensaver to start while the arcade-only welcome notice is open. Waking returns to the notice; a fresh hold of A dismisses it.

## v1.0.33 - 2026-09-14

- Start with an arcade-only catalogue. Options > List > Show non-arcade cores restores console, computer and other cores; hidden favorites are kept. Upgrades explain the new default once; hold A for two seconds to dismiss, with a progress indicator beside the legend. Filters show Stable/Beta directly, and catalogue counts and refresh announcements follow the enabled catalogue. Clear all filters leaves the preference alone; Update All keeps its normal scope.

- Legends in Filters, Views, the main list, Details and the updater log now follow the available actions, including disabled rows, empty lists and scrolling boundaries.

- Hide deprecated cores by default, with a remembered Options toggle to show them.
- Add a dissolving startup logo and page transitions, with a remembered Page transitions switch in Options.
- Keep Details scrolling paced to display refresh and within the frame budget.
- Animate gentle light waves in the lettering background. Thank retrofan01 in the early adopters credits.

## v1.0.32 — 2026-09-14

- Options previews now live inside the existing hint box. Layout shows three diagrams; Title font shows the three fonts. Scroll speed and Hold delay show a selection moving over fixed game names, with three rows horizontally and four in tate. Animation follows display frames, matching held-button scrolling.
- Remember last view is now a child of Views. Turn it off to choose a Default view from the enabled views, used each time MisterZine starts, including after a game. Changing these settings leaves the current view alone.
- Add a dim screensaver that keeps the current screen visible at 33% or 66% brightness. Screensaver Enabled is separate from its saved delay; brightness choices use filled indicators.
- Recognize the theypsilon unofficial distribution source, including Nemesis in the bundled catalogue.
- Shorten the list option labels to Art type and Layout. Thank ac3 and Fallon in the early adopters credits.

## v1.0.31 — 2026-09-14

- Highlight List layout in Options to compare diagrams of List, Split and Picture side by side. The previews follow horizontal or tate orientation, with list lines, an artwork rectangle and colored metadata lines. Left/Right changes the layout and selection border; moving to another option closes the previews.

## v1.0.30 — 2026-09-14

- Screensaver settings now have their own Options subpage, with an immediate Preview.
- Screenshot filters combine games on the card, current rotation, favorites and original resolution. They are independent of list filters and appear only with Screenshots selected, alongside Brightness and Info. Switching styles keeps their values.
- With no matching screenshots, lettering runs instead; Preview explains the empty selection.
- Button hints in Filters and while holding Select follow A, B, X, Y order.

## v1.0.29 — 2026-09-14

- Fix: games whose core file carries a longer name than their MRA says
  read as not found on the card although they were installed and launched
  fine (issues misterzine#9 and #10: Black Heart, Captain America and The
  Avengers, Double Wings, Diet Go Go, Mania Challenge, Mat Mania, Thunder
  Dragon and Zero Wing from the Coin-Op Collection, which ships
  `blkheart_mister_20260909.rbf` for `<rbf>blkheart</rbf>`). MiSTer's MRA
  loader resolves that by prefix, accepting any rbf whose filename starts
  with the tag followed by `_` or `.`, also behind `Arcade-`, the greatest
  filename winning when several match; the scan only tried the exact,
  date-stripped and `arcade-` names. The scan now applies MiSTer's rule as
  its last fallback, after the attempts that matched before, so every row
  that matched keeps its file; a name without the separator (`tdrago` for
  `tdragon_mister`) still does not match. The launch path resolves through
  the same lookup. The launch picker's alternatives treat two core names as
  the same core when one is the other followed by `_`, with or without the
  `arcade-` prefix and their own date suffixes, so a row that says
  `blkheart_mister` (what the catalogue now exports for those games) lists
  the alternative MRAs that say `blkheart`, and a cached catalogue that
  still says `blkheart` lists MRAs that say `blkheart_mister`. The guide's
  card status section says how a core file is matched.
- The latest-update view's last-look line follows the dates instead of a
  200-row window: it sits after the last unseen row dated on or after the
  newest stamp the baseline holds, so a core catalogued late under an old
  date keeps its own unseen mark but never drags the line down, and the line
  sits on top reading Nothing new since your last look only when a scan of
  the whole view finds nothing unseen, shortening to the age alone or to
  Nothing new where the list is too narrow for the sentence. No changes in
  top 200 and No changes in this view are gone.
- ItsDanik joins the early adopters in Credits.

## v1.0.28 — 2026-09-13

- The screenshots screensaver types what the main view's pane says about the
  game onto its shot instead of cutting the title onto a tab. Once a shot
  has wiped in, the title on one line in the shot's hue (cut short with
  an ellipsis when it is too long for the width, where the pane wraps it
  over two), the card answer, the kind, the core, the year and maker, the
  rotation, players and controls and the badges appear a character a
  frame with a fifth of a second's beat at each line end, in a bottom
  corner of the safe zone, the other corner at each shot with a little
  wander, each line on an opaque black strip that reaches only as far as
  its letters have been typed plus the cursor's cell, behind an underline
  cursor a cell wide that is solid while it types and blinks once a
  second once the block is complete. A wipe hides the caption and a wipe
  turned back by a hold brings it back where it was. The hold's line and
  the not-on-the-card word sit under the block on the same black, and the
  caption dims and comes up with the shot as the tab did. The pane's line
  list is shared with the main view, so the two say the same things.
- Options -> Screensaver info, with the screenshots only: full (default)
  types the pane's lines, title only types just the title, none leaves
  the picture alone. Holding Start still brings the picture up and fills
  its line whichever is set, and with none it puts the title up whole for
  the hold, so the hold always shows what it is about to play. The choice
  is saved as screensaver_info; any other word reads as full.
- L and R jump between the Options sections as they do between the list's
  letters and the Filters headings: to the first row of the previous or
  next section (Data, List, Display, Controls, Operation), from inside a
  section back to the previous one, stopping at either end. Keyboard Home
  and End still go to the first and last row. The Filters jump and this
  one are one routine, landing on the heading there and on the first row
  here since the Options headings are not selectable.
- The lettering screensaver's whole frame is posterised to three bits a channel
  and dithered in a 4x4 Bayer cell, the sharp first picture, the blur
  levels, the fade and the glints alike, so the pattern reads as a display
  with few colours instead of hiding in the last bit. The quantiser
  scales to the steps so a cell averages to the value it stands for,
  folds the shade into its one multiply and mixes two blur levels with
  one more, so a fade frame writes the screen in 13 ms on the boards
  against 17 before: a divide by 255 and a second multiply on the same
  chain had cost them 15 ms a frame. The tuner gains Bits and Cell, and a
  benchmark of the ground write runs on a board from the cross-compiled
  test binary.
- Options names its rows in full and steps the dependent ones in:
  Screensaver delay, Screensaver style and Screensaver brightness replace
  the truncated Saver rows, Main menu launcher becomes Main menu shortcut,
  and Rotation, the screensaver rows, Open at boot and Return after game
  sit a step in behind a branch mark as children of the row above, the
  mark in the section headings' grey with the name in the row's own
  colour so it reads as structure. A child whose full name does not fit
  at the widest tate safe zone falls back to Style, Brightness or Info
  behind the mark instead of losing its tail.
- The Options legend follows the row under the cursor instead of always
  reading Change, A Open, B Back: a choice row names the arrow that can
  still move, both in the middle and one at either end; a row A acts on
  says what A does there (Preview on the screensaver rows, Open on Views,
  Last update result, Troubleshooting and Credits, Edit on the safe zone,
  Run on the update, Refresh, Rescan, Clear and Quit); a greyed row leaves
  only Back.

## v1.0.27 — 2026-09-13

- Options -> Saver style chooses what the screensaver shows. Lettering
  (default) is the MISTERZINE sweep. Screenshots shows one arcade game's
  gameplay shot after another, every arcade game in the catalogue in a
  shuffled cycle, each new one wiping in from the side behind a soft
  dithered edge that wanders and changes shape as it goes, with the
  game's title on a black tab that moves from corner to corner in the
  shot's own hue. Only the gameplay shot is used: not the title screens,
  and not the third slot, which is often a game-over screen. Online, a
  shot the card lacks is downloaded when its turn comes; offline, only
  the shots on the card show, and with nothing to show at all the
  lettering runs. Start held for two seconds on a shot plays that game:
  the picture comes up to full brightness with a line filling under the
  title, and a hold let go early fades it back to half brightness. A hold
  that catches a wipe half way turns the wipe back, so the shot you were
  looking at is the one that plays and the incoming one waits its turn.
  A game not on the card only says so. Every other button wakes. The
  title tab, its hold line and the not-on-the-card word are drawn at the
  shot's brightness, so a half-bright shot carries a half-bright title
  and the hold brings both up together.
- Options -> Saver brightness, with the screenshots only: half (default)
  shows the shots at half brightness, kinder to a CRT, and the Start hold
  brings one up to full; full shows them at full brightness throughout.
- Options -> OK button: which face button confirms, per pad. MiSTer's
  define buttons screen ends by asking which pad button is MENU OK and
  which MENU BACK, and stores both in the pad's map file (slot 23,
  SYS_BTN_MENU_FUNC, back in the high half and OK in the low half). Auto
  from MiSTer (default) follows it: on a pad whose MENU OK is the button
  defined as B, B confirms and A goes back, Enter and back trading places
  as the event arrives, and the legends name the buttons that way round
  (the label sets apply after the swap, so an Xbox-lettered pad defined
  by position reads "A Open"). MENU OK unset, on A, or on any other
  button leaves A confirming, and the hint says which it saw; one button
  chosen for both counts as unset. A or B overrides it for one pad, kept
  in settings.json under ok_buttons by the pad's vendor and product as in
  MiSTer's map file name, so two pads of one model share it. The row
  shows and edits the pad that last pressed a button, or the only defined
  pad connected when none has (a pad that is several event nodes with one
  name, as the Xbox 360 pad is, counts once), and it is muted with the
  reason before any pad has pressed, for a pad that comes through
  MiSTer's translation, and for a pad whose A and B are the same button,
  which has no back button and is called out in the pad tester too. The
  pad tester names each pad's OK button and where it came from, and the
  input log line prints each pad's MENU OK and BACK choice.
- A pad MiSTer has not defined is held and read directly when it reports
  the standard Linux gamepad layout, as nearly every pad does, instead of
  staying with Main, whose default menu button opened its OSD and took
  the screen from the app, which then left at once; and while any held
  pad was connected such a pad's face buttons were dropped altogether.
  By position, the right button is A, the bottom B, the top X and the
  left Y, the shoulders L and R, Select and Start themselves, the home or
  guide button is Menu with the tap and hold behaviour, and the hat or the
  d-pad buttons and the left stick move. The bottom button is its OK, as
  in Main's own default, so Auto from Linux (B) confirms with it until
  the pad is defined in MiSTer, which always wins. A pad without the face
  buttons (joystick-class codes) gives up Start alone as before, and a
  virtual device is never held: the Zaparoo pad claims a USB bus type, so
  a virtual device is told by its empty physical path, where a real pad
  reports its USB port or Bluetooth address.
- Options is in five sections. Controls, before Operation, holds Button
  labels, OK button, Menu button, Scroll speed and Hold delay, so the rows
  changed with the pad in hand sit together above the rarely touched
  ones. Each section heading carries a one-cell mark and a rule to the
  right instead of a colon, as the list's group markers: a diskette for
  Data, stacked lines for List, a monitor on a stand for Display, an
  arcade stick for Controls and two sliders for Operation, five pixels
  wide and seven tall on the baseline in every font. Credits and Filters
  headings are unchanged.
- Credits: shad00m and washaa join the early adopters.

## v1.0.26 — 2026-09-13

- Fix: a stop signal that arrived while the screensaver was up was not
  acted on until a key woke it (v1.0.25). The screensaver's frame loop,
  new in v1.0.25, served keys, downloads, scans and the debug API between
  frames but not the process signals, so a SIGTERM from an updater, a
  reboot or a shutdown sat in its channel. The loop now hands a signal
  back to the main loop, which stops the app as before.

## v1.0.25 — 2026-09-13

- The screensaver blurs and dims the picture into a soft glow instead of
  cutting to a dark copy of it. The fade takes a second and climbs through
  four blur levels that a background worker computes from the picture
  frozen at the first saver frame, each level with a larger radius and
  more bloom, so the picture softens progressively rather than crossing
  from sharp to blurred; a slow board waits for a level rather than
  jumping. A key runs the fade back in a quarter of a second, lettering
  gone at once, and only then repaints the screen. The light parts bloom:
  channels above a knee of 171 are raised six times their excess in
  16-bit channels, so nothing clamps before the blur spreads them, and the
  blurred picture shows at 57%. The levels keep eight bits of fraction
  and a static 8x8 ordered dither at the final write stops the dark
  gradients banding on a CRT. The look was chosen on a CRT through a
  debug endpoint (POST /api/saver, docs/DEVELOPMENT.md). One level costs
  the boards a couple of hundred milliseconds, so the picture under the
  saver stays frozen until a wake; Update All and downloads carry on
  behind it.
- The saver's word repeats edge to edge once it is in, the seam an
  ordinary letter gap (the letters carry their own side bearings), instead
  of leaving the screen before entering again.
- Fix: the saver's scroll hitched every couple of seconds. Measured on
  both boards over a minute, the lettering never skipped a column, but
  the 30 fps timer that paced the frames was not locked to the 60 Hz
  picture, and 46 frames a minute missed their vertical blank: one shown
  for three refreshes, the next for one. The saver is now paced by the
  blank itself, a frame every second blank and copied at the blank, with
  the lettering moving one pixel per frame by count; after the change 2
  frames a minute miss, at the start of the fade while the worker runs.
  Keys, the debug API, downloads and scans are served between frames.
  The debug state reports the cadence between presents and the saver's
  frame counts.
- Legends in Title Case ("A Details  B Options  X Filters  Y View") with
  "Select +" at the end of the list bar as a reminder of the Select
  chords (Select in the button colour, the plus muted). In tate the bar
  reads "A Open  B Opt.  X Filt.  Y View  Select +". Chunks close up to
  one space before any is dropped when the safe zone leaves no room.
- Manufacturer replaces Maker everywhere on screen: the view name, the
  top bar, the group headers ("Unknown manufacturer"), the Views page and
  the guide; Details labels the credit "Mfr" to keep its column. The
  saved view name in settings is unchanged.
- Contact bounce: a press that follows the same key's release by under
  25 ms is dropped. Worn arcade microswitches deliver a second press 2 to
  8 ms after the release, and the quickest deliberate double tap is
  several times longer. Only the press side is guarded, so nothing waits;
  a bounce during a hold ends the hold, which costs one repeat delay.
- A held Backspace erases a search at 60 ms a step after the Hold delay,
  instead of 200 ms a step after a fixed half second.
- The Options -> Screensaver help mentions the blur.

## v1.0.24 — 2026-09-13

- Fix: turning the last view back on in Options -> Views (Recents,
  typically) did not survive a restart. An empty "views off" set was
  saved as null, which the next start read as a settings file from
  before the Views page and switched Recents off again. Every view on now
  stays on.
- Select + A on the list stars or unstars the game under the cursor, with
  the same notices as Y in Details; in the Favorites view an unstarred
  row drops out and the cursor moves to its neighbour. While Select is
  held only the three chords act (Y layout, X shots, A favorite): nothing
  moves, opens or launches until it is released.
- The pad's Menu button: a tap opens or closes Options as you let go. A
  hold shows "keep holding to quit MisterZine" after 300 ms and leaves
  the screen alone from then on; letting go early does nothing. While the
  hint is up, the line under the top bar fills green from left to right,
  reaching the far edge at the two-second quit. The B holds that cancel
  Update All and leave the pad tester fill the same line from the press,
  and the Update All message drops its whole-second countdown.
- Pressing X on the Filters screen returns to the list, as B does.
  Expanding and collapsing a decade's years stays on Right and Left.
- Options copy: "Remember sort order" is "Remember last view", "Canvas" is
  "HDMI picture" (it changes nothing on CRT modes, 720p or 1440p), the
  Menu button choice reads "quit MisterZine" and the hold notice and pad
  tester say quit too. Credits sits above Quit MisterZine, the last row,
  and neither carries a hint. The Main menu launcher hint no longer
  claims Setup is needed to restore the entry: turning the option back on
  does that, and Scripts -> MisterZine-Run opens MisterZine while it is
  off. The Return after game, HDMI picture, Button labels and Screensaver
  hints were too long for the help box on a 320x240 tate screen with a
  15 px safe zone and now fit; a test keeps every hint inside the box.
- Credits: Porieux joins the early adopters.

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
