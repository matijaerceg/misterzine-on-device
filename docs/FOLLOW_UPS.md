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

- **Code review fixes: one small batch at a time.** The user must test each
  batch before implementation proceeds to the next. Batch 1 (`efd13f9`)
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
  Batch 7 flushes recovery checkpoints before replacing the previous record
  and moves card saves outside the output-reader lock. The full updater and
  Update screen tests passed on both devices; checkpoint/recovery tests also
  passed using temporary directories on each device's actual SD card.
  User acceptance is
  pending: let Update All finish normally, check the live log stays responsive,
  and return to Options with B. Do not start batch 8 before this test.
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
