# DE10-Nano display corruption investigation — 2026-09-08

Companion to the code review prepared for Claude. This records the hardware
investigation so a later session can distinguish observations from hypotheses.

## Target and current status

MisterZine must work with the latest official MiSTer Main. As checked on
2026-09-08, that is Main 20260907; Menu 20260907 is also the latest release.
The DE10 is now running both September 7 releases, with the original core and
Main backed up. The user confirmed stable CRT output on entry to MisterZine and
after three exit/re-entry cycles. **Recovery is confirmed; prevention of the
original delayed idle failure is still pending.**
No MisterZine application source or binary was changed during these diagnostics.

The user also wants the MiSTer Pi / DE10 Menu wallpaper difference investigated
**after** the corruption issue. That follow-up is deferred, not forgotten.

## Observations

1. After leaving MisterZine idle, the CRT showed fixed, full-height vertical
   stripes, resembling one image row repeated down the screen. Ignore the CRT
   photograph's scan-line noise; the repeated image content is the failure.
2. The app's logical screenshot and a separately decoded read of `/dev/fb0`
   both showed the correct menu while the physical CRT showed stripes.
   CPU-visible framebuffer content was therefore intact at capture time.
   Neither capture verifies the actual FPGA scaler output.
3. Linux reported 320 x 240, 32 bpp, stride 1280; frame interrupts continued
   at approximately 60 Hz. Reapplying that mode, then briefly requesting
   640 x 240 before restoring 320 x 240 and repainting, did not clear the stripes.
4. A diagnostic Main screenshot command displayed a transient “Scaler not
   compatible” notification. The command could not capture this framebuffer
   path. This notification was caused by the diagnostic, not an app warning.
5. Warm-loading Menu 20260907 changed the visible failure to moving digital
   corruption: ribbons/rows, a scrolling diagonal line, and periodic whole-image
   vertical jitter. The same corruption appeared in both Menu wallpaper and
   MisterZine.
6. Restoring and warm-loading the original June Menu core did **not** clear
   that moving corruption. It is not established that the new release caused
   it; the earlier suggestion of a release regression was premature.
7. A full Linux reboot with the original June core cleared the corruption.
   The user confirmed that the Menu wallpaper was stable.
8. Installing Menu 20260907 and rebooting again also produced stable Menu
   wallpaper, confirmed by the user, with the original August Main still
   installed. Warm reload and full reboot gave different results.
9. Installed Main 20260907 alongside Menu 20260907 and rebooted. Both installed
   hashes matched the official downloads. Opened MisterZine through its normal
   MGL launcher: the user confirmed a stable picture.
10. Completed three app exits to Menu and MGL re-entries. Each exit reached
    `MENU`; each re-entry reached the list with 320 x 240 / stride 1280. The
    user confirmed the physical CRT was still stable after these transitions.
11. Left MisterZine running idle after the last entry at device wall-clock
    11:30:54 (device clock, not a claim about the host's timezone). The delayed
    failure has not yet been ruled out by a comparable idle interval.

## Working interpretation

The failure is downstream of the CPU-visible framebuffer and affects a display
path shared with Menu wallpaper. A full reboot cleared state that reapplying
framebuffer dimensions and warm core reloads did not. This narrows the search;
it does not prove a specific DDR, clock, or FPGA state-machine failure.

Menu's September 7 update contains a relevant scaler change: it resets the
output read/copy state at vertical sync to recover when an output-clock pause
during PLL reconfiguration leaves the reader stuck. That is a plausible match
for the original repeated-row symptom, not a confirmed reproduction of its
trigger. An app screenshot alone cannot establish that this fixes the CRT.

Primary sources:

- [Menu scaler change, b19b9fd](https://github.com/MiSTer-devel/Menu_MiSTer/commit/b19b9fd1ab61c74f3f11593f8ddef6115331e839)
- [Official Main releases](https://github.com/MiSTer-devel/Main_MiSTer/tree/master/releases)
- [Main changes between the installed August and latest September releases](https://github.com/MiSTer-devel/Main_MiSTer/compare/dfb4791bea126afea66025be806651a995f9cfd6...f8dc68e3dcf4694f5593e6552aea56cd852982af)

## Versions and recovery information

| Component | Original | Test target |
|---|---|---|
| Main | Official 20260823 | Official 20260907 |
| Menu | Official 20260603 | Official 20260907 |
| MisterZine | `v0.2.1-13-g6dada3b-dirty` | Same installed binary |

SHA-256 values:

```text
Main 20260823  ef0dbfde3744ccb160444932d8a7d31f298b16a95db506852c99ece018321596
Main 20260907  aa201f19da90d7bd130ae128620a3e8e627a99a2a2dda36c6ff23a75b514ec59
Menu 20260603  821bcf66181a00ff550e4a4110dc11c9fa8e68d38e9cb5558b3ddb99ca938934
Menu 20260907  25d5461b55e4d45e79c876a02d69f32b22f414b64e600a1adc930eefea6ea4a7
```

DE10 backups:

```text
/media/fat/MiSTer.codex-before-20260908
/media/fat/menu.rbf.codex-before-20260908
```

The relevant display configuration has `[Menu] direct_video=1`, global
`vga_scaler=0`, `fb_terminal=1`, and `video_mode=8`. These diagnostics did not
change MiSTer.ini. The Menu framebuffer is 640 x 240 at boot; MisterZine requests
320 x 240 on entry and restores its original geometry on exit.

## Remaining validation

- [x] Verify both latest binaries are active after reboot.
- [x] Enter MisterZine using its regular MGL launcher and inspect the physical CRT.
- [x] Exercise app exit/re-entry and inspect the physical CRT again.
- [ ] Leave the app idle long enough to cover the original delayed failure.
- If corruption recurs, capture Linux framebuffer data and diagnostic state
  before resetting it; compare the physical symptom with the two distinct
  failures above. Avoid claiming that correct framebuffer bytes prove stable
  scanout.
