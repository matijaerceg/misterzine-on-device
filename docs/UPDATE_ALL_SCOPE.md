# Update All screen: implementation and handoff

Implemented September 8, 2026. This replaces the earlier scope proposal.

## User experience

From the main list: B → Options → Run Update All → A. A dedicated screen
blocks normal browsing and launching until the run ends. It shows a
five-stage bar (Prepare, Check, Update, Extras, Finish), current activity,
elapsed time, age of the last output, and a scrollable live log. The bar
advances from recognized upstream announcements; it is deliberately not
an overall percentage. Optional stages may be skipped. The spinner is
the supervisor's activity indicator, not proof a remote download is moving.
If status stops arriving, the header shows “no status”.

Up/Down scroll the log; L/R page it. Hold B for two seconds to request
cancellation. Ordinary work is terminated as a process group, escalating
after three seconds if necessary. During a detected Linux or Pocket
firmware update, the request waits for that phase to finish. A process
check additionally protects known bootloader/flash writers. Cancellation
does not roll back files already changed.

Both the initial termination signal and force-stop escalation use the same
phase and process checks. If a protected operation appears after the first
signal, escalation waits and the screen says it is finishing a system update.
Known writers are checked before briefly pausing the process group; the check
is repeated with the group paused before either signal is sent. The group is
resumed on every exit from that check. Protection is not overridden by a time
limit: cancellation remains pending until the detected operation clears or
the updater exits.

A persistent **Restart expected** ribbon appears on the first Linux-update
announcement or detected reboot requirement. This can precede the actual
kernel write. Update All's saved configuration, including automatic
reboot, is retained. There is no per-run restart override or silent change
to those preferences. A required restart that upstream only announces
late cannot be predicted earlier.

Completion, partial update errors, failure, cancellation and interruption
are separate outcomes. B returns to Options after the run, and the host
rescans the card. Decorative separator lines remain in the saved log but
are omitted from the small on-screen log.

## Ownership and recovery

- `internal/updater/runner_linux.go` starts a copied supervisor executable
  in its own session, holding an inherited exclusive lock. The updater
  child has a separate process group and no controlling terminal. It
  never opens another agetty session or writes Main's `/tmp/script`.
- The supervisor owns stdout/stderr drainage independently of the UI.
  Closing MisterZine through Main's menu button or quitting its process
  leaves the run alive. The next launch reconnects to that run ID. A
  duplicate managed start returns the existing run rather than starting
  another updater. Recognized external updater processes block new starts;
  the MGL watcher also checks before taking the script console.
- The full installed `/media/fat/Scripts/update_all.sh` wrapper runs with
  `UPDATE_ALL_NON_INTERACTIVE=true` and unbuffered Python output. Its
  self-update/`--continue` sequence is preserved. The copied supervisor
  survives replacement of MisterZine's installed executable; replacing
  the UI with an older release could still remove reattachment support.
- Status is published atomically in `/tmp`, with a card checkpoint every
  five seconds and at important state changes. The last summary and log
  are `/media/fat/misterzine/update-all/state.json` and `output.log`.
  The live tail is limited to 160 lines, individual lines to 512 characters,
  and the file log rotates by truncating after approximately 2 MiB.
- Terminal escape sequences are stripped. Common credentials, secret URL
  parameters and HTTP URL passwords are redacted on a best-effort basis.
  The integration captures ordinary output, not the upstream debug log.
- PID, run ID and boot ID validate a live worker. On a later boot,
  persisted success is described as “reported success before restart”,
  not a newly verified successful run. Missing workers without that
  evidence show interruption. Nothing restarts Update All automatically.

The modal protects navigation inside MisterZine. It cannot block the
physical power switch, Main's own menu button, or a separate root/Remote
session starting scripts. External launchers do not honour MisterZine's
lock, so races with them cannot be eliminated here. For a normal external
Scripts/Remote update, first quit MisterZine and wait for the updater to
finish before reopening. Prefer the Options action for this supervised
workflow.

## Upstream behaviour verified

Checked against Update All implementation `b845085`, distributed as
2.10 b84 on the Pi (repository head `35a3a0e`). Non-interactive mode skips
settings/countdown and final interactive log/timeline, while retaining
ordinary output and the saved sequence. `DLP1` events in Update All cover
RetroAccount sync, not the full updater pipeline.

Sources: [configuration](https://github.com/theypsilon/Update_All_MiSTer/blob/b845085/src/update_all/config_reader.py),
[execution/outcomes/reboot](https://github.com/theypsilon/Update_All_MiSTer/blob/b845085/src/update_all/update_all_service.py),
[child output](https://github.com/theypsilon/Update_All_MiSTer/blob/b845085/src/update_all/os_utils.py),
[structured events](https://github.com/theypsilon/Update_All_MiSTer/blob/b845085/src/update_all/update_output.py).

Known reboot markers are `/tmp/MiSTer_downloader_needs_reboot` and
`/tmp/downloader_needs_reboot_after_linux_update`. The nested Downloader's
`ALLOW_REBOOT=0` does not suppress Update All's final configured reboot.
Unknown future output wording remains visible in the log; it may not
advance the stage bar or identify a protected phase. The phase parser and
writer checks are conservative compatibility guards, not an upstream
transaction/cancellation API.

## Validation

Automated tests exercise streaming/carriage returns, partial lines split
across heartbeats, bounded/sanitized output, stage advancement, early
restart flag, modal navigation, short versus long B, duplicate start,
wrong-run cancellation, TERM-resistant process groups, protected-phase
deferral, a writer without a preceding announcement, detached parent exit,
partial updater errors, an unconfirmed zero exit, and recovery after a
simulated reboot. The detached tests run real shell processes against
temporary fake card directories, with no firmware writes.

The cancellation escalation regression starts a simulated protected phase or
fake writer only after the initial termination signal. It checks that the
operation survives past the three-second escalation deadline, that the UI
summary explains the wait, and that cancellation completes after the operation.
A separate writer check verifies that an already detected writer is not paused
and resumed repeatedly while cancellation waits. The updated updater suite
and Update screen tests passed on both DE10-Nano and MiSTer Pi in review
batch 6; these tests did not run the installed Update All or change firmware.

The updater suite and Update screen/input tests passed directly on the
MiSTer Pi's ARM CPU. This caught and fixed an epoch-time conversion overflow
in the spinner; a 32-bit UI regression check is now included in CI.
Harness renders cover horizontal and tate layouts, including the early
restart ribbon and cancellation state. Reproduce them with:

```sh
go run ./cmd/mzharness -update-state testdata/update-running.json -out out/update-h -script "shot running; hold back 2200; shot cancel"
go run ./cmd/mzharness -update-state testdata/update-running.json -rot left -logical -out out/update-t -script "shot running; hold back 2200; shot cancel"
```

Real Pi checks on official September 7 Main/Menu and Linux 6.18.38:

- 19:57 UTC: Options launched Update All 2.10 b84; completed successfully
  in 11 seconds, no new installs/errors and no restart required.
- 19:58 UTC: holding B for 2.2 seconds requested cancellation; the run
  stopped, showed Cancelled, and triggered a card rescan.
- 19:59 UTC: quit MisterZine during a fresh run, confirmed the supervisor
  continued with parent PID 1 and no terminal, reopened through the MGL,
  and reconnected to run `dla7986rn7nz`, which completed successfully.
- The user confirmed the Update All result screen was **readable and
  stable on the CRT**.

No real Linux/Pocket write or automatic reboot was required by these runs;
those paths have simulated coverage, not a new firmware-write hardware
test. DE10-Nano was left untouched during those updater tests. No new
public release was published.

## Final installation on both devices

At the user's subsequent request, build **update-all (924c3a9, 2026-09-08)**
was installed on both the MiSTer Pi and DE10-Nano. Both reported that
version through the running app. The installed binaries have identical
SHA-256:

`d821a5f8d429dbc66d8130db030017019ba03eb4d3c60f6a1a96e3d2bb0372e8`

The DE10-Nano opened the new Filters and Options screens correctly at
320×240; the Run Update All entry is present. Its existing Linux 5.15.1
was retained. No Update All run or firmware change was performed on the
DE10 for this installation. Both devices were left in Options.

Previous executable backups:

- Pi: `/media/fat/misterzine/misterzine.before-update-all`
- DE10: `/media/fat/misterzine/misterzine.before-update-all-20260908`

Full CI passed for the installed revision, including vet/tests, a 32-bit
UI test, ARM cross-build, downloader package validation and render artifacts.
