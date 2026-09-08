# MiSTer Pi: MisterZine will not open after Update All

Date: September 8, 2026. Handoff for Claude.

## Outcome and remaining verification

The Pi's immediate launch failure is traced to the framebuffer driver in
the official September 7 Linux release. A MisterZine compatibility build
is installed on the Pi and reaches the app successfully. Two normal
quit-to-Menu and reopen cycles passed. Physical CRT confirmation is still
pending; a working HTTP screenshot or process does not establish that the
analog picture is correct.

The Pi retains official Main 20260907, Menu 20260907 and Linux
6.18.38-MiSTer. No kernel, Main, Menu or ini changes were made in this
investigation. The DE10 was left untouched for its separate idle test.
The Pi wallpaper question remains deferred.

## Evidence and cause

The user ran Update All in the background, returned to Menu, then found
that selecting MisterZine produced its initial flicker but no app.

Device logs establish this sequence (times are device UTC):

- 18:31:35: the old app detected Main taking the screen and shut down.
- Update All 2.10 (b84, b845085) completed successfully at 18:35:56, after
  updating Main, Menu and Linux. Its final message scheduled a reboot in
  30 seconds. The Linux update was from 250402 to 260907.
- At inspection, Linux was 6.18.38-MiSTer, built September 7, and Update
  All was no longer running.
- 18:39:30: MisterZine requested 320×240 successfully in 14 ms, then
  logged `fb: mmap: no such device` and exited with status 3.
- Kernel diagnostics from that same PID contain
  `fb0: fb_WARN_ON_ONCE(!info->fbops->fb_mmap)` in `fb_chrdev.c`.

This failure happens before the first frame. It is different from the
DE10's long-idle repeated-row corruption, where CPU framebuffer contents
were intact and the physical picture was wrong.

Linux removed implicit framebuffer file operations in 6.8. The MiSTer
driver in the shipped image does not supply the required mapping
operation. An upstream fix was merged September 8 at 15:16:57 UTC:
[MiSTer kernel commit ea22122](https://github.com/MiSTer-devel/Linux-Kernel_MiSTer/commit/ea2212221ad137cf26bf5caa7ad3dab7216435a6).
It adds explicit read/write/mmap operations and selects their helpers.
At the time checked, Distribution still offered
`linux_release_20260907.7z`, uploaded September 7, which predates that fix.

## App compatibility change

`internal/platform/mister/fb.go` still tries ordinary `/dev/fb0` mmap
first. Only when it returns ENODEV for the exact `MiSTer_fb` driver does
the app map the framebuffer's reported pixel-memory range through
`/dev/mem`, opened with O_SYNC. It reads the address from
FBIOGET_FSCREENINFO; it does not guess or hard-code it.

The fallback checks alignment, nonzero address and length, 32-bit range
overflow, framebuffer dimensions, stride and mapping length. It maps no
more than the driver reports and excludes the palette/header page. On
this Pi the range is 0x22001000 for 307200 bytes at 320×240×32 bpp.
Failures such as permissions or memory exhaustion do not trigger it.
Once a fixed kernel is installed, normal mmap takes precedence again.

Startup now attempts to restore the original framebuffer geometry when
mapping fails. Previously that failure closed the file without restoring
the mode it had just requested.

This is an app workaround for the shipped kernel. It does not repair
framebuffer access for other applications or utilities on the Pi.

## Verification and deployed artifact

- Read-only physical mapping probe on the actual Pi succeeded before
  installing the app change.
- `go test ./...` passed on Windows; Linux/ARM `go vet ./...` and the
  static ARMv7 app build passed.
- Both new test groups, with 17 cases, passed as a Linux/ARM executable
  on the Pi. They cover fallback selection, error preservation,
  physical-address validation and framebuffer bounds.
- 18:50:31: new app used the fallback, reached its first frame in 584 ms,
  started its debug service and scanned installed cores.
- Two subsequent exits returned to `MENU`; each reopening reached the
  list at 320×240, rotation left. The final reopening at 18:53:33 reached
  its first frame in 432 ms.
- No real Update All run was started for these tests.

Deployed build label: `kernel-20260907-compat (e556cb2-dirty, 2026-09-08)`.
It was built from the worktree based on e556cb2 with the changes described
above, before the resulting source commit. This is a local device build;
no new public release/tag was published.

SHA256 of `/media/fat/misterzine/misterzine`:
`e21ed98aae72371264d3ddfec55588bbd5a630d835fd7c0b022496dc43b2e355`.

The prior executable is preserved at
`/media/fat/misterzine/misterzine.codex-before-kernel-20260908`, SHA256
`1497384fb28b636469bcc56137415df0809cf16ade77a230dc1836bba7cafa17`.
The resident watcher was restarted using the new binary. The wrapper,
user state, favourites and settings were preserved.

Main SHA256:
`aa201f19da90d7bd130ae128620a3e8e627a99a2a2dda36c6ff23a75b514ec59`.
Menu SHA256:
`25d5461b55e4d45e79c876a02d69f32b22f414b64e600a1adc930eefea6ea4a7`.

## The three Update All questions

The precise launch method used by the user has been asked but is not yet
confirmed. Distinguish a separate SSH/background process from a script
whose console has merely been hidden.

| Question | Established behaviour and limits |
| --- | --- |
| Start Update All while MisterZine is open? | An independent process can run. Normal Remote script launch is unavailable while MisterZine owns the console: the Pi returned `canLaunch: false`. The regular Scripts-menu path requires leaving the app. Do not force a second app onto the same script console. Update All may finish by rebooting, as this run did. |
| Exit MisterZine during Update All: does it freeze the updater? | MisterZine's shutdown and Menu watcher do not send stop signals to Update All. A detached test process continued across two app exits and reopenings. That verifies process independence, not all Update All phases. A terminal-bound process may instead wait for input or be disrupted if another launcher takes its console. |
| Open MisterZine while Update All runs? | A detached updater can continue, subject to updates changing files and possibly rebooting. The ordinary console-bound updater and MisterZine are not coordinated: both use tty2 and `/tmp/script`. MisterZine's MGL watcher currently has no guard against a different running script, so this overlap is not supported. A new agetty session may hang up the previous console session. |

The detached test was a disposable Python process with its own session,
no terminal and redirected standard streams. Its progress counter
advanced through both transitions and it exited normally after 90
seconds. It did not download, modify or update anything on the card.

Review anchors:

- `cmd/misterzine/launcher.go`: `runFromMenu` writes `/tmp/script`, starts
  agetty on tty2, waits for the app, then the watcher reloads Menu.
- `cmd/misterzine/main.go`: shutdown restores the framebuffer/console;
  there is no updater pause/resume protocol.
- [Official Main at the deployed release, menu.cpp](https://github.com/MiSTer-devel/Main_MiSTer/blob/f8dc68e3dcf4694f5593e6552aea56cd852982af/menu.cpp#L7407):
  Scripts launch also uses `/tmp/script` and tty2.
- [Remote script implementation](https://github.com/wizzomafizzo/mrext/blob/main/pkg/mister/scripts.go):
  script launch checks for Menu and no active script, then uses that same
  console. Current upstream was read; the Pi's `canLaunch: false` was
  independently observed.

For normal use: exit MisterZine, run Update All to completion, allow any
requested reboot, then reopen MisterZine. If a deliberately independent
updater ran while the app stayed open, rescan installed cores afterward.
Live concurrent updating and core launching are not validated here.

Potential follow-up: explicit console ownership checks and a user-visible
busy message in the MGL launcher. A full concurrent-updater feature would
also need process supervision, detached output, completion/rescan and
reboot handling; the current app has none of that coordination.
