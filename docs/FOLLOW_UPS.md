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
  batch before implementation proceeds to the next. Batch 1 only fixes
  image-worker shutdown with pending downloads or unreadable cached images.
  Its regression tests failed before the fix and now pass on the Windows
  host, MiSTer Pi and DE10-Nano. User checks:
  browse while pictures load, launch an installed game, reopen MisterZine,
  and quit through Options; confirm prompt transitions and a stable CRT.
  User acceptance is still pending. Other review fixes remain unstarted.
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
