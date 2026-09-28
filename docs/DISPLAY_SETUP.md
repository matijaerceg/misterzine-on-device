# Display setup

Choose the connection you use before opening MisterZine for the first time.
Games can display correctly while MisterZine still needs a display setting:
its interface uses MiSTer's Linux screen output, which takes a different path
from a game core's picture. MisterZine does not rewrite MiSTer.ini.

| Your connection | Setup |
|---|---|
| Normal HDMI monitor or TV | [HDMI](#normal-hdmi-display) |
| RGB or component CRT on an analog board | [Analog-board CRT](#rgb-or-component-crt-through-an-analog-board) |
| CRT on a supported HDMI-to-VGA adapter | [Direct-video adapter](#crt-through-an-hdmi-to-vga-adapter) |
| MiSTercade or another JAMMA cabinet | [JAMMA cabinets](#jamma-cabinets-mistercade) |
| S-Video or composite TV | [Y/C output and its colour limitation](#s-video-and-composite-yc-output) |
| Both HDMI and a CRT | [Choose which screen shows MisterZine](#hdmi-for-the-menu-crt-for-misterzine), or [show it on both](#can-hdmi-and-crt-show-misterzine-together) |

## Before editing MiSTer.ini

Back up the active MiSTer.ini, quit MisterZine, and restart MiSTer after saving
your changes. Keep your existing sync and RGB/YPbPr settings.

The `[Menu]` examples below configure the menu, scripts and MisterZine; game
cores keep their own settings. Edit your existing `[Menu]` section if you have
one. To change only MisterZine when opened from its main-menu entry, use
`[MisterZine]` instead and put it at the end of the file, below `[Menu]`.
The [dual-display example](#hdmi-for-the-menu-crt-for-misterzine) shows both.

MisterZine needs `fb_terminal=1`, which is MiSTer's default even if the line is
absent. Only change it if you previously disabled it with `fb_terminal=0`.
Do not disable it in response to MiSTer's generic console warning.

## Normal HDMI display

Use these settings for the menu and MisterZine:

```ini
[Menu]
direct_video=0
vga_scaler=0
```

## RGB or component CRT through an analog board

For a 15 kHz CRT on the analog board's VGA/YPbPr port:

```ini
[Menu]
direct_video=1
vga_scaler=0
```

Direct video ignores `video_mode` for the menu core and runs it at 240 lines,
or 288 lines at 50 Hz with `menu_pal=1`. MisterZine fills either mode, at
320x240 or 320x288, doubled across the line. Before v1.1.4 the 288-line mode
got a 384-wide canvas that MiSTer could only place once, so the picture sat
centred at 60% of the screen's width; on that version, `menu_pal=0` was the
workaround for a TV that holds 60 Hz. The report's `INI:` line shows
`menu_pal`.

Keep the sync and RGB/YPbPr settings your CRT needs.

## CRT through an HDMI-to-VGA adapter

For a supported HDMI-to-VGA direct-video adapter, use:

```ini
[Menu]
direct_video=1
vga_scaler=0
```

The adapter receives the direct-video output, at 240 lines or 288 lines with
`menu_pal=1`. Keep the sync and RGB/YPbPr settings your CRT needs. Power off
before changing between that adapter and a normal HDMI display.

## JAMMA cabinets (MiSTercade)

MiSTercade's MiSTer.ini ships with `vga_scaler=0` and `direct_video=2`.
Setting 2 is Main's auto mode: it only switches direct video on when an HDMI
DAC is plugged in. With nothing on HDMI, as in a cabinet, the framebuffer
stays on HDMI, so the cabinet monitor never shows MisterZine or any other
script. Choosing MisterZine then looks like a blink and a return to the
menu, and the first run says "JAMMA cab?" while the log warns about
`direct_video=2`. `fb_terminal` is not involved; leave it at 1.

Add this section at the end of MiSTer.ini, then reboot:

```ini
[Menu]
video_mode=320,16,32,16,240,4,3,16,6048
vga_scaler=1
```

That routes the menu core's scaler output to the cabinet at 240p, so Update
All and the other scripts become visible there too. The long `video_mode`
line is the 15.7 kHz, 60 Hz timing the arcade cores themselves send, taken
from the timings MiSTercade lists in its INI. MiSTercade's README suggests
`video_mode=320,240,60` for this section instead; Main computes that form
into a 15.0 kHz, 59 Hz signal, which some arcade monitors will not hold, and
the picture rolls in the menu and in MisterZine. Keep `vga_scaler=0` in the
main `[MiSTer]` section: games must bypass the scaler, or they reach the
cabinet at the HDMI resolution and lose sync.

If the monitor is rotated, add the `osd_rotate` value for it under the same
section. Use `[MisterZine]` instead of `[Menu]` only if an HDMI display also
serves the menu and should keep it. Confirmed on a MiSTercade v1 on
September 15, 2026 with exactly the section above under `[Menu]`.

## HDMI for the menu, CRT for MisterZine

If an HDMI display serves the MiSTer menu, the terminal and Update All, and
only MisterZine should go to the CRT, leave `[Menu]` on the HDMI settings and
add a `[MisterZine]` section **below it** with the CRT settings:

```ini
[Menu]
direct_video=0
vga_scaler=0

[MisterZine]
direct_video=1
vga_scaler=0
```

The main-menu entry, `MisterZine Arcade.mgl`, loads the menu core under the
name `misterzine` (the section follows that name, not the entry's file name),
and Main applies every section matching either name in file order, so
`[MisterZine]` overrides `[Menu]` only while MisterZine is open; quitting
restores the menu on HDMI. Put `osd_rotate` there as well if only MisterZine
runs on a rotated CRT: Follow INI rotation reads the same sections Main does.
The section must come after `[Menu]`, or `[Menu]` wins. It only applies to
launches through the menu entry: the Scripts entry runs under the plain menu
core and keeps the `[Menu]` output.

## S-Video and composite (Y/C) output

MisterZine cannot be shown in colour through MiSTer's native Y/C output
(`vga_mode=svideo` or `vga_mode=cvbs`), and no MiSTer.ini setting changes
that. The Y/C encoder lives in the FPGA framework and is fed from the core's
own picture, after the OSD. MisterZine draws into Linux's framebuffer, which
reaches the analog pins only through the scaler path, and the framework's
output selector takes that path instead of the Y/C one whenever the
framebuffer is routed there. The direct-video output is wired the same way.
Making the framebuffer Y/C-capable would be a change to MiSTer's framework,
upstream of this project.

What a Y/C configuration shows today:

- With `vga_scaler=0` and `direct_video=0`, the analog-board recipe, the
  framebuffer never reaches the analog port. MisterZine runs but nothing
  appears; the log warns and the first run shows the CRT notice.
- With `direct_video=1`, the pins carry plain RGB while MisterZine is open.
  The Y/C encoding is off, so the adapter receives the green channel as luma
  with no colour burst, and the CRT shows MisterZine **in black and white**.

Game cores keep their Y/C output either way, as long as `vga_scaler` stays 0
in the main `[MiSTer]` section. To use MisterZine on the same CRT, in
monochrome, give it the direct-video output only while it is open, with a
section at the end of the file (below `[Menu]` if there is one):

```ini
[MisterZine]
direct_video=1
vga_scaler=0
```

A global `vga_scaler=1` gets a picture too, but turns Y/C off for the game
cores as well. Confirmed on September 20, 2026 on a DE10-Nano with an analog
board, `vga_mode=svideo` and exactly the section above: MisterZine appears in
black and white. The MiSTer.ini `vga_mode` and `ntsc_mode` values are not read
by MisterZine and do not appear on its log's `ini:` line.

## Can HDMI and CRT show MisterZine together?

The current interface cannot provide normal HDMI resolution and native 240p CRT
output simultaneously. Unlike a game core's native picture, Linux's framebuffer
enters through MiSTer's scaler. Enabling `vga_scaler=1` copies that scaler output
to the analog port at the same timing; it does not separately convert 1080p to
240p. See MiSTer's [video settings](https://mister-devel.github.io/MkDocs_MiSTer/advanced/ini/#general-video-settings).

An advanced shared-timing setup may work if **both displays accept the same
CRT-compatible mode**. Ordinary HDMI TVs often do not accept that signal. Do not
enable `vga_scaler=1` with a 720p/1080p mode on a 15 kHz CRT.

## MiSTer.ini warning or picture does not fit

Main's CRT message suggesting `fb_terminal=0` or `vga_scaler=1` is generic console
help. Disabling the console prevents MisterZine's launcher from working; choose
the output configuration above instead. For frequent switching, MiSTer supports
[named alternate INI files](https://mister-devel.github.io/MkDocs_MiSTer/advanced/ini/#alternate-mister-ini-naming-option);
MisterZine reads whichever one is active, and its log's `ini:` line names the file
and the `osd_rotate` it found.

Use Options -> Edit safe zone if text reaches outside the visible CRT picture.
