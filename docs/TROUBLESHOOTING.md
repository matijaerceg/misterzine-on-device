# Troubleshooting

## Display setup

MisterZine draws into Linux's framebuffer: a memory area containing the screen's
pixels. Its launcher needs Main's framebuffer console to display those pixels.
`fb_terminal=1` is already the MiSTer default, even if the line is absent from
MiSTer.ini. Only change it if you previously disabled it with `fb_terminal=0`.

Choose the output for Menu and MisterZine in the existing `[Menu]` section of
MiSTer.ini. Save a backup before editing, quit MisterZine, and restart MiSTer
afterward. These settings apply to Menu/script output; game cores use their
own settings. MisterZine does not rewrite MiSTer.ini.

For a normal HDMI display:

```ini
[Menu]
direct_video=0
vga_scaler=0
```

For a 15 kHz CRT on the analog board's VGA/YPbPr port:

```ini
[Menu]
direct_video=1
vga_scaler=0
```

A CRT connected through a supported HDMI-to-VGA direct-video adapter already
receives the direct-video output. Power off before changing between that adapter
and a normal HDMI display. Keep the sync and RGB/YPbPr settings your CRT needs.

### Can HDMI and CRT show MisterZine together?

The current interface cannot provide normal HDMI resolution and native 240p CRT
output simultaneously. Unlike a game core's native picture, Linux's framebuffer
enters through MiSTer's scaler. Enabling `vga_scaler=1` copies that scaler output
to the analog port at the same timing; it does not separately convert 1080p to
240p. See MiSTer's [video settings](https://mister-devel.github.io/MkDocs_MiSTer/advanced/ini/#general-video-settings).

An advanced shared-timing setup may work if **both displays accept the same
CRT-compatible mode**. Ordinary HDMI TVs often do not accept that signal. Do not
enable `vga_scaler=1` with a 720p/1080p mode on a 15 kHz CRT.

Main's CRT message suggesting `fb_terminal=0` or `vga_scaler=1` is generic console
help. Disabling the console prevents MisterZine's launcher from working; choose
the output configuration above instead. For frequent switching, MiSTer supports
[named alternate INI files](https://mister-devel.github.io/MkDocs_MiSTer/advanced/ini/#alternate-mister-ini-naming-option).

Use Options -> Edit safe zone if text reaches outside the visible CRT picture.

The September 7, 2026 MiSTer Linux framebuffer driver lacked the usual mapping
operation. MisterZine includes a fallback for that driver and uses normal mapping
when available. The [kernel fix](https://github.com/MiSTer-devel/Linux-Kernel_MiSTer/commit/ea2212221ad137cf26bf5caa7ad3dab7216435a6)
is also available upstream. Downgrading Main or Linux is not an installation step.

## Main-menu entry missing or not opening

Run **MisterZine Setup** in Scripts. It creates the menu entry and enables the
startup helper. Setup is repeatable. If files are missing, run Downloader first.
After a successful setup, return to the main menu and select MisterZine.

An external updater can temporarily prevent the launcher from opening. Let it
finish before trying again. Errors from the app wrapper show recent log lines
before returning; the logs are `misterzine/log.txt` and `misterzine/watch.log`.

## Favorites unreadable

The app keeps the original favorites file on read errors and disables favorite
edits for that session. Restore or repair `misterzine/favorites.json`, then
reopen the app. Do not delete the file as a generic troubleshooting step.

## A game or an alternative is missing

Use Rescan card after an external update. Alternatives must be installed:
Downloader filters can intentionally exclude them. MisterZine leaves those
preferences unchanged. A missing or invalid launch target reports an error.

## No connection or an old catalogue

Cached data remains usable. Check the connection, then use Refresh data now.
Pictures load as you browse and failed downloads retry with increasing delays.
The bundled catalogue is a starting point for a first launch without a connection.

## Reporting a problem

Include the app version and data date from Options, your board, display
connection, rotation, and the steps that reproduce it. A photo helps for physical
display problems; F12 captures what the app draws. Remote debugging is optional
and described in the development guide.
