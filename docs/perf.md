# Measured on the device

MiSTer Pi (Cyclone V, Cortex-A9 at 800 MHz), 2026-09-08, build 9590340.

| What | Measured | Budget |
|---|---|---|
| First frame from process start | 420 to 460 ms | 500 ms |
| of which data.json parse (1206 rows, 639 KB) | about 280 ms | (rows.bin cache is the planned fix if this grows) |
| fb_cmd1 320x240 request landing | 13 ms | 500 ms |
| Vsync interval | 16.65 ms | 60 Hz |
| Full 320x240 present (byte shuffle into the mmap) | 3.8 ms | 5 ms |
| Full 640x240 present (fallback scaling) | 7.5 ms | |
| Card scan: 550 cores + 1066 MRA stats | 544 ms, off the UI goroutine | |
| Alternatives: 499 MRAs parsed | 1.2 s cold, cached by directory mtime after | 5 s |
| Framebuffer restore + console restore on quit | under 60 ms | |

Workstation (for reference): a 53x20 text screen paints in 73 us, a 240x320
frame rotate takes 176 us, a 384x224 to 96x56 area-average resample 1.2 ms.

Follow-ups: cache the parsed rows in a binary form to cut the first frame by
about 250 ms; the picture resampler uses float64 and could go fixed-point.
