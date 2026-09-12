MisterZine can come back by itself: after a game, and after a boot.

- **Return after game:** Options -> Return after game, off by default. On,
  when a game you started from MisterZine exits to the MiSTer menu (Reset or
  Exit in the OSD), the menu launcher picks the MisterZine entry again, the
  same way you would, and the list opens on that game. The MiSTer menu shows
  for a moment in between. Games started from the MiSTer menu itself, and
  leaving MisterZine with the pad's Menu button, do not bring it back. Needs
  the Main menu launcher (Setup enables it).
- **Open at boot:** Options -> Open at boot, off by default. On, once the
  MiSTer menu is up after a power-on or reboot, the launcher picks the
  MisterZine entry for you. A `bootcore` in your INI takes precedence and
  nothing happens. Needs the Main menu launcher.
- **Faster Y:** each list order is sorted once per loaded catalogue and
  reused, and the Year order no longer re-parses years while sorting, so
  cycling orders, typing a search and card-status updates skip the sort.

Both new switches start off; existing settings, favorites, filter choices,
remembered versions and the launch history are preserved. Neither switch
changes MiSTer.ini, the menu core or any MiSTer file: both are carried out by
the same background helper that already runs the main-menu entry.

Existing installations can update through Update All/Downloader. Reopen
MisterZine after updating.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
