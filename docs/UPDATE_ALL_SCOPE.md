# Update All from Options: implementation scope

Status: scoped September 8, 2026; not implemented or launched by this work.

## Recommended first version

Options → Run Update All opens a dedicated progress screen. It shows the
current stage, elapsed time, recent output and a scrollable log. Normal
browsing and core launching are unavailable until the run finishes. This
meets the user's requirement for visible feedback; a silent background
launch is explicitly unacceptable.

Use a measured progress bar when a stage supplies a trustworthy total.
Otherwise show an activity indicator and live output, with “No new output
for N seconds” when quiet. Do not manufacture an overall percentage from
elapsed time or count arbitrary log lines. Always distinguish completion,
completion with errors, failure, and restart required.

## Verified upstream support

Update All was refreshed to upstream 35a3a0e (implementation b845085).
`UPDATE_ALL_NON_INTERACTIVE=true` skips the settings countdown, settings
screen and final interactive log viewer. It still produces output. This
allows MisterZine to own the presentation while the updater performs the
work. It must use the user's existing update configuration.

Sources: [configuration](https://github.com/theypsilon/Update_All_MiSTer/blob/b845085/src/update_all/config_reader.py),
[execution flow](https://github.com/theypsilon/Update_All_MiSTer/blob/b845085/src/update_all/update_all_service.py).

The normal updater captures its downloader's stdout/stderr and forwards
them. Capture output from the launcher from the start, not just the final
saved log. The `DLP1` emitter in Update All is used for RetroAccount sync;
it is not an overall progress protocol for a full Update All run.
Per-download counters, if available in the installed Downloader, need a
separate compatibility check before implementing percentage bars.

Sources: [child-process output](https://github.com/theypsilon/Update_All_MiSTer/blob/b845085/src/update_all/os_utils.py),
[structured events](https://github.com/theypsilon/Update_All_MiSTer/blob/b845085/src/update_all/update_output.py).

Reboot policy is read from Update All's saved configuration. Non-interactive
mode does not suppress its final reboot. Downloader's `ALLOW_REBOOT=0`
only stops the nested Downloader from rebooting; Update All can still
reboot afterward. Do not mistake that flag for control over the whole run.

## Work required

1. **Runner and ownership.** A supervised process separate from tty2,
   with its own output capture and run lock. Never use Main/Remote's normal
   Scripts launcher, `/tmp/script` or another agetty session. Refuse a
   duplicate run, including an updater started elsewhere. Check for active
   scripts before opening the MGL app as well.
2. **Visible run screen.** Bounded in-memory log, persistent run summary,
   live stage and elapsed time, scrolling output, and a truthful progress
   indicator. Sanitize terminal control codes; preserve carriage-return
   progress updates. Do not display credentials from verbose diagnostic
   output. Keep logging off the rendering loop.
3. **Lifecycle.** Closing the UI or Main taking its screen must not stop
   the updater or block a full output pipe. A supervisor keeps draining
   output and allows the next app launch to reattach to the running job.
   The modal prevents app navigation; it cannot prevent physical power-off
   or Main's own menu button. No blanket “cancel” during Linux/bootloader
   writes. Show loss of contact as unknown, not success.
4. **Self-updates and reboot.** Preserve the launcher's continuation flow
   when Update All updates itself. Account for MisterZine's executable
   being replaced while its UI and watcher are running; stage a consistent
   runner and avoid mixing old and new watcher protocols. For v1, visibly
   announce and honour the configured reboot. A “Restart now/later” button
   needs a verified per-run upstream override first; do not silently edit
   the user's persistent updater preferences.
5. **Completion.** Read the final outcome, present errors and retained
   output, rescan the card when appropriate, and offer Return to Options.
   A prior success marker alone cannot prove this run succeeded. Record
   the run ID/start time and final result. After a reboot, recover the last
   known outcome without automatically starting another update.

## Validation before shipping

Start with a fake updater that emits lines, partial lines, progress,
errors, silence and large output. Exercise UI exit/re-entry, duplicate
launch, process failure and simulated reboot/self-update continuation.
Then use a controlled real run on the Pi, with hardware observation.
Confirm text fits both CRT orientations, output keeps moving, and no
stage steals tty2 or overwrites the app display. Include Linux-update and
reboot handling in the acceptance matrix before promising full support.

This is a moderate feature involving process supervision and UI, not just
an Options button. A live-log modal is the smallest useful first version.
Free browsing while updating can follow later, after file replacement and
core-launch conflicts are handled.
