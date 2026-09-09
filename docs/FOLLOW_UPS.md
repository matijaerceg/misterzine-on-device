# On-device follow-ups

Updated September 8, 2026. This consolidates the on-device project memory;
it is not a list of unrelated website work.

## Closed or superseded

- **Phantom double inputs: controller issue.** The user confirmed one
  controller causes the double inputs, not MisterZine. Remove this from
  the app investigation queue. Do not add debounce or input workarounds
  for this report.
- **Filters/Options split:** X opens Filters, B opens Options from the
  main list; both return directly to the list with B. Filter sections are
  On the card, Favorites, Since last look, then the existing remaining
  sections. The former Settings screen is Options.
- The older proposed two-press B-to-quit flow is superseded by B → Options.
- Full-screen still artwork already exists. The scrolling artwork idea
  below is a separate, unimplemented presentation.

## Active verification / current scope

- **Code review fixes: related batches.** Initially the user tested each batch;
  from batch 16 onward they authorized continuing with automated/device checks
  while unavailable. Keep visual acceptance pending rather than claiming it passed. Batch 1 (`efd13f9`)
  fixes image-worker shutdown. The user confirmed it works on DE10-Nano;
  its regression tests also passed on MiSTer Pi.
  Batch 2 (`2c9389a`) protects favorites after read errors and fixes the
  Options label to "Quit MisterZine". The user confirmed it passes.
  Batch 3 (`967dcdf`) fixes disabling and re-enabling the main-menu launcher.
  Device checks and the user's test passed.
  Batch 4 (`3878f43`) preserves the selected game version and details scroll
  position when returning from artwork. The user confirmed it on MiSTer Pi;
  regression tests passed on both devices for B/A/X returns in all rotations.
  Batch 5 (`3188a00`) clears expired scan messages on the Update All screen, preventing
  a stale timer from forcing continuous redraws. Regression tests reproduced
  the old failure and now pass on both devices, along with the existing
  progress-rendering and long-B cancellation tests. The user confirmed it.
  Batch 6 (`93ec3b5`) applies the same system-write guard to both cancel signals, including
  force-stop escalation; the screen explains when cancellation is waiting.
  A known writer is checked before pausing the group, avoiding repeated pauses
  while a cancellation waits. The updater suite and Update screen tests passed
  on both MiSTers using temporary simulated updaters, with no firmware writes.
  The user confirmed it.
  Batch 7 (`df85a0d`) flushes recovery checkpoints before replacing the previous record
  and moves card saves outside the output-reader lock. The full updater and
  Update screen tests passed on both devices; checkpoint/recovery tests also
  passed using temporary directories on each device's actual SD card.
  The user confirmed it.
  Batch 8 (`a0d8344`) separates warning detection from the shortened display log. Known
  markers are recognized beyond 512 characters and across 4096-byte flushes,
  including split terminal escapes, and are consumed in arrival order.
  A display flush cannot manufacture a line-start success announcement.
  The new regressions reproduced the old failures. The full updater and
  Update screen suites now pass on both devices with simulated updaters.
  The user confirmed it.
  Batch 9 (`0ef953e`) remembers B-dismissal of an interrupted/restarted update
  warning using a separate run-ID record. It preserves the original recovery
  evidence and never suppresses a live run or a different interrupted run.
  The user confirmed it; recovery tests passed on both devices and SD cards.
  The user requested slightly larger related batches from batch 10 onward.
  Batch 10 (`e64903f`) fixes autosave scheduling for settings, favorites and
  data-state changes, retains failed saves for five-second retries, and
  preserves automatic rotation until an explicit rotation choice. Host tests
  passed on Linux and both MiSTers; save/retry/rotation and favorites-protection
  tests passed in temporary directories on both actual SD cards. The complete
  host suite runs on /tmp: its Unix-permissions test cannot run unchanged on FAT.
  The user confirmed batch 10 and approved broader related batches.
  Batch 11 serializes freshness checks and coalesces card scans, snapshots the
  current dataset on the UI thread, and only replaces alternatives on a final
  scan result (including an empty result). Stale positional statuses are discarded
  and rebuilt off the UI thread. New scan/refresh requests wait during Update All;
  already-running work may finish. Worker deliveries can stop when the UI quits.
  Freshness retries are scheduled from completion; manual checks use the installed
  hash. The debug row count now follows that same current dataset.
  Full host tests passed on both devices; new refresh/cache, scan, update-deferral
  and shutdown tests also passed using temporary directories on their SD cards.
  The user confirmed batch 11.
  Batch 12 rebuilds an open Filters/Options panel when new data arrives, keeps
  the final list page filled after the view shrinks, and rejects empty network
  datasets before replacing the working list or its saved cache. Tests reproduced
  stale filter counts and the nearly blank final page before the fixes. Full app
  and host suites passed on both devices, with empty-response cache preservation
  also checked on their SD cards. The user confirmed batch 12 and requested
  a larger next batch.
  Batch 13 addresses six image/display findings: matching thumbnail prewarm
  variants (9), skipping unreadable cached images for the session without deleting
  them (12), retrying delayed prefetch downloads and resetting the cursor on
  reconnect (13), removing unreachable full-screen neighbour requests (38), using
  the Filters font for Clear all filters (43), and copying debug screenshots on
  the UI thread before encoding (48). Rename failures now also enter backoff.
  App/image/debug-server suites passed on both devices; unreadable-file and
  prefetch retry tests passed on their SD cards using temporary fixtures.
  The user confirmed batch 13 and requested another larger batch.
  Batch 14 fixes seven review findings: details/spec/Versions overlap at high
  overscan (39), status-count overflow (40), first-visit Since filtering and its
  empty message (41), calibration text bounds (44), repeated repainting of an
  unchanged terminal Update All result (45), and README control/options/build
  drift (22/23). Artwork shrinks where needed to reserve space for Versions.
  The update poll still checks at 500 ms to detect external changes; unchanged
  states no longer repaint. CI now renders all six screens in both orientations
  (coverage portion of 21; comprehensive golden-image comparison remains deferred).
  New pixel-boundary, launch-region isolation, first-visit and update-idle tests
  pass, as do full app/host suites on both devices. Maximum-overscan renders were
  visually checked. User acceptance is pending: browse Details/Filters/Options,
  open calibration, and quit/reopen. No real Update All is needed for this batch.
  The user confirmed batch 14.
  Batch 15 addresses launcher-hook detection (24), concurrent shutdown (32),
  incomplete alternative-file caching (55), atomic/no-op alternatives cache saves
  (56), and launch preflight/core resolution (5). Versioned cache entries invalidate
  the old potentially incomplete cache. Cache/read failures reach the device log.
  Invalid/missing launch targets and unavailable Main commands keep the app open;
  later marker/send failures return failure and do not log a successful launch.
  Late failures after framebuffer cleanup still return to Menu; a richer recovery
  screen is separate. Core resolution uses the same suffix/alias rules as scanning.
  Host/scan/platform suites passed on both devices; cache recovery and preservation
  tests also passed on their actual SD cards. User acceptance is pending: rescan,
  inspect alternate versions, launch an installed game, reopen, and quit normally.
  The user approved continuing after batch 15. The DE10's missing alternatives
  are explained by its downloader.ini !alternatives filter (and screen_no_tate
  for vertical 1943), not an app scan defect. Leave downloader preferences alone.
  Batch 16 addresses cold-boot clock correction (8) and last-look recovery (47).
  Server time advances from a monotonic sample instead of reapplying an offset
  after NTP catches up; calendar labels and saved dates use that corrected clock.
  Notices, Details guards and update animation use the input/timer clock.
  A clockless return can use the previous snapshot if no baseline exists; an
  existing baseline stays fixed. First clock recovery stamps and autosaves this
  visit without advancing its comparison baseline. Trusted quick-return rules stay.
  Local tests, ARM vet, full host/app suites on both devices, last-look tests and
  recovery persistence on both SD cards passed. Cold clocks were simulated in
  tests; neither device's system clock was changed. User acceptance is pending:
  browse Details/Filters/Options, check date labels and quit/reopen normally.
  The user is unavailable to test and explicitly authorized continuing with
  automated/on-device checks. Batch 16 visual acceptance remains pending.
  Batch 17 addresses Update All startup B-holds (66), protected-write feedback
  (67), stable log reading (68), and button hint accents (69). A hold made before
  the run ID arrives is retained, but cancellation is sent only once an ID exists;
  releasing early still cancels nothing. Protected active runs show a keep-power-on
  ribbon, and queued cancellation explains why it waits. Scrolling back pauses
  the displayed log snapshot; reaching the bottom resumes current output.
  Local tests/ARM vet and full app/host suites on both devices passed, including
  startup handoff/release and log-tail replacement regressions. Protected/restart
  renders were checked in horizontal and tate orientations. No real Update All
  was run for these UI changes. Visual/controller acceptance remains pending.
  Batch 18 addresses unreadable update status (64) and slow supervisor startup
  being mislabeled as failure (65). Status read errors are logged once per repeated
  error and shown as a brief notice; missing history on first install stays quiet.
  Unreadable records are preserved. A known active run keeps its last state while
  the worker lives; a vanished worker becomes an interruption instead of leaving
  the modal stuck. A live supervisor that has not yet published status after the
  startup wait returns its real run ID/PID as starting and reconnects via polling.
  Tests cover delayed startup, supervisor exit before status, corrupt checkpoint
  preservation, deduplicated diagnostics, and leaving a missing-status dead run.
  Cold/failure cases use temporary fixtures, never the devices' actual updater.
  Visual/controller acceptance for batches 16–18 remains pending.
  The user said batch 18 seems fine and asked to continue.
  Batch 19 addresses the old resident launcher after updates (26) and Menu
  restoration while an updater survives the UI (62). Idle launchers detect an
  atomically replaced executable and re-exec with their PID unchanged, only when
  no updater is active and no MisterZine console session is owned. Missing or
  invalid replacements keep the current watcher alive, with retries after errors.
  Menu restoration waits for managed/external updaters and never replaces a
  different selected core. Host suites passed on both devices; replacement
  detection was also tested on both SD cards. Live replacement checks passed on
  both: replacing the executable during an app session left the old watcher in
  place until Menu, then it adopted the new inode with the same PID. Reopening
  passed afterward. Build 514265d is installed on both; its CI passed.
  Batch 20 addresses ellipsis baseline placement (50), background picture
  completion redraws (51), and both fixture regeneration traps (59). The font
  uses its parsed descent; downloads/counts send a separate Options-progress
  notification while decoded images still trigger display updates. The golden
  generator includes arrival batches; snapshot refresh stages downloads privately,
  validates nonempty data, and leaves coupled golden fixtures untouched. New Node
  and mocked-download Python tool tests run in CI. Full host/images/fonts suites
  passed on both devices; local Go tests, ARM vet and both tool tests passed.
  Finding 46 was checked against current site origin/main: its 23h30m rounding
  matches the site exactly. Defer any change until both agree on revised behavior.
  Batch 21 processes ten findings: 4, 5, 11, 15, 27, 33, 35, 46, 63, 71.
  - 4: framebuffer/console restoration now precedes image-worker shutdown. Input
    close remains bounded at 500 ms; cleanup still precedes load_core. SIGKILL
    cannot run in-process cleanup; external geometry recovery remains a limitation.
  - 5: earlier preflight tests still cover missing/invalid launch targets. The new
    wrapper error display also makes late marker/command failures visible instead
    of instantly disappearing; it does not attempt to recreate the app after exit.
  - 11: persistent random debug-token authentication on every endpoint, POST-only
    mutations, Origin rejection, IP/localhost Host validation, no-store responses,
    and a debug-startup notice. tools/dev.sh fetches the token via SSH. Credentials
    are never logged. This remains opt-in private-LAN HTTP, not an Internet service.
  - 15: all terminal results can be acknowledged once; Options has Last update
    result without triggering another run. Run Update All/Rescan remain adjacent.
  - 27/35: wrapper keeps a restore executable in RAM, shows recent log/error status
    for up to 15 seconds after failure, and cleans up its helper on normal exit.
    A mock self-replacing failing executable validated restore and visible reason.
  - 33: navigation no longer schedules state.json rewrites; unused cursor/sort/
    filter fields are omitted. New-visit/clock/data state still autosaves, as do
    settings/favorites independently. Every visit still starts with default view.
  - 46: reviewed, no isolated change: matches current website rounding. Coordinated
    date-ladder behavior remains deferred, not claimed as a fixed defect.
  - 63: verified already fixed by batch 11: initial and periodic requests go through
    updater-aware deferral; existing host tests cover draining queued work afterward.
  - 71: output lines buffer in RAM; the supervisor flushes outside the state mutex.
    A bounded 2 MB queue drops older buffered log text with an explicit marker if
    the card stalls extremely long. Parser/live tail keep updating; a blocked pipe
    test confirms a stuck log flush does not lock output parsing. Heartbeat can
    still be delayed by synchronous card I/O; no claim of asynchronous storage.
  Local tests, wrapper fixture, ARM vet and host/app/updater/debug suites on both
  devices passed. Build 27f8ca2 and its Scripts wrapper are installed on both;
  CI passed. Live tests verified 401 without token, 405 for mutation GET, 403 for
  browser Origin, result review without a new run, unchanged state.json after
  cursor movement, restore-helper removal on exit, persistent token access, and
  no repeat greeting after result acknowledgement. Options layouts were inspected.
  IMPORTANT for future device tools: every debug request now needs the
  X-MisterZine-Token header, read via SSH from /media/fat/misterzine/debug-token.
  Keep that credential out of tool output and logs. Options row 7 is Last update
  result; Refresh/Clear/Quit move to rows 8/9/10, respectively (zero-based).
  Remaining substantive review scope after this batch: 14 (protected latch),
  21 (golden-image assertions), 52 (unconfirmed framebuffer request), 53/54 (input
  loss/probe ownership), 61 (external updater launcher notice), 70 (inherited pipe
  drainage). Plus deferred 46 site parity and the explicitly retained limits above.
  Priorities guide the batches, with related fixes grouped so each remains
  small and testable; this is not a strict traversal of the review's numbering.
- **DE10 long idle stability:** short transitions passed on latest Main
  and Menu; the user's comparable long-idle CRT result is still pending.
- **Pi after Linux update:** the framebuffer compatibility build runs;
  the user confirmed the Update All result screen is readable and stable
  on the CRT. Longer idle observation remains separate from that check.
- **Update All with visible feedback:** implemented as a dedicated screen,
  stage bar, live log, long-B cancellation and early restart ribbon. See
  [UPDATE_ALL_SCOPE.md](UPDATE_ALL_SCOPE.md) for validation and limits.
- **Release:** package the accumulated fixes in the next release, verify
  downloader installation on both devices, and remove debug flags from
  normal installs. No new public release has been requested in this turn.

## Deferred user earmarks

- Why the Pi has no Menu wallpaper while the DE10 does, after display
  corruption is resolved.
- Account/favorites sync with the website.
- Native-size artwork that slowly pans through a cropped viewport in
  details; pause/restart controls and separate landscape/tate layouts.
- Cleaner installation and complete uninstallation.
- Consistent “MisterZine” naming, including the Scripts entry and release
  assets; retain the stable downloader database ID `misterzine`.
- Optional Zaparoo/Remote launch integration.
- More themes and a denser HDMI layout, longer term.

## Earlier UX ideas not yet selected for implementation

- Y = “only this” for a filter value, plus an active-filter summary/count.
- After rescanning, report how many releases became current.
- Distinguish newly seen rows from changed release dates in details.
- Review “nothing new” wording against the 200-row last-look scan limit.
  Do not imply a full-catalogue check if only that window was examined.

Performance notes also mention a binary data cache and fixed-point image
resampling if needed. These are engineering opportunities, not new user
requirements or confirmed performance problems.
