# Troubleshooting

## Display setup

`fb_terminal=1` must be enabled in MiSTer.ini. It lets Main open the
framebuffer console used by MisterZine and other scripts.

HDMI uses Main's framebuffer output. A CRT connected through an HDMI-to-VGA
DAC in direct-video mode already receives that output. For the analog board's
VGA/YPbPr port with no HDMI display attached, configure:

```ini
[Menu]
direct_video=1
```

For simultaneous HDMI and analog CRT output, the analog port needs the scaler
and a suitable 15 kHz mode. One example is:

```ini
[Menu]
vga_scaler=1
video_mode=640,54,56,106,224,16,0,28,13764
```

Put these settings in the existing Menu section rather than creating competing
sections. They configure Menu/script output, not individual game cores. Adjust
for your display; MisterZine does not rewrite MiSTer.ini.

If Main displays its message about `vga_scaler=1`, review this routing setup.
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
