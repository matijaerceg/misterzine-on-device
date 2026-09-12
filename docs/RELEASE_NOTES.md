MisterZine opens on a card whose main menu is Degauss.

- **Degauss:** Degauss installs itself as MiSTer's `main=` frontend and takes
  over every load of the menu core, and MisterZine's menu entry loads the
  menu core. Choosing MisterZine therefore brought up Degauss, and the
  launcher then waited for a console it could never get until the next
  reboot. The launcher now closes Degauss for the MisterZine session and the
  usual menu restore brings it back when MisterZine exits. Return after game
  and Open at boot work the same way on such a card.
- **MisterZine-Run:** a new Scripts entry that opens MisterZine from a
  Scripts list, since Degauss lists Scripts but not MisterZine's menu entry;
  the frontend returns when MisterZine quits. Opened that way, leave through
  Options -> Quit MisterZine or the pad's Menu button, as keyboard F12 stays
  with the Scripts session. From MiSTer's own menu, the MisterZine entry
  works as before.
- The launcher's console switch now gives up after two seconds with a line
  in `watch.log` instead of waiting for the rest of the boot.

Nothing changes in MiSTer.ini, the menu core or Degauss's own files;
MisterZine only closes Degauss for its own session. Existing settings,
favorites, filter choices, remembered versions and the launch history are
preserved.

Existing installations can update through Update All/Downloader, which also
adds the MisterZine-Run entry. Reopen MisterZine after updating.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
