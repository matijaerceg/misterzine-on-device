A `[MisterZine]` section in MiSTer.ini can give MisterZine its own display
settings.

- **HDMI for the menu, CRT for MisterZine:** MisterZine's menu entry loads
  the menu core under the name `misterzine`, and Main applies INI sections
  named after either that name or `Menu`, in file order. A `[MisterZine]`
  section placed below `[Menu]` therefore changes `direct_video`,
  `vga_scaler` and `osd_rotate` only while MisterZine is open, and quitting
  restores the `[Menu]` output. An HDMI display can keep the menu, the
  terminal and Update All while the CRT shows MisterZine alone. The
  [troubleshooting guide](https://github.com/matijaerceg/misterzine-on-device/blob/main/docs/TROUBLESHOOTING.md#hdmi-for-the-menu-crt-for-misterzine)
  has the exact lines.
- **Follow INI rotation** now reads `[MisterZine]` the way Main does, so an
  `osd_rotate` placed there is honoured, and the CRT-only warning in the log
  and on the first run no longer fires for that layout. The first-run notice
  names both sections.
- The section must come after `[Menu]`, or `[Menu]` wins, and it applies to
  launches through the menu entry: the Scripts entries run under the plain
  menu core and keep the `[Menu]` output.

MisterZine still never edits MiSTer.ini. Existing settings, favorites, filter
choices, remembered versions and the launch history are preserved, and a card
without a `[MisterZine]` section behaves exactly as before.

Existing installations can update through Update All/Downloader. Reopen
MisterZine after updating.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
