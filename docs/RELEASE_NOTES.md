Your pad's A is A, whatever the MiSTer menu calls it, and you can see
exactly what every button does.

- **Pads read as you defined them:** a pad defined in the MiSTer menu is now
  read by MisterZine directly, slot by slot: D-pad, A B X Y, L R, Start and
  the menu stick, including triggers or stick directions used as buttons.
  Before, MisterZine took its A and B from the buttons you chose for the
  MiSTer menu's own OK and back, so a pad with OK on the bottom button and
  back on the right one had them swapped here. Now the button you defined as
  A is A, and each pad follows its own definition, so two pads defined
  differently work side by side. A pad never defined in MiSTer keeps working
  through MiSTer's translation while no defined pad is connected.
- **Button labels:** Options -> Button labels chooses how the legends name
  the face buttons, in MiSTer's A B X Y order: A B X Y (default), B A Y X for
  an Xbox-lettered pad mapped by position, PlayStation symbols by position,
  or 1 2 3 4 in MiSTer's define order. Only the names change.
- **Pad tester:** Troubleshooting -> Test pad buttons lists every pad being
  read with the raw code in each MiSTer slot, then shows each press as it
  happens: which pad, which button or axis, which slot, what MisterZine does
  with it, and the gap since the previous press, which also makes a bouncing
  arcade switch easy to spot. Hold B for two seconds to leave.

Nothing changes in MiSTer.ini, the menu core or your MiSTer controller
definitions; MisterZine only reads the map files MiSTer already saves.
Existing settings, favorites, filter choices, remembered versions and the
launch history are preserved.

Existing installations can update through Update All/Downloader. Reopen
MisterZine after updating.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
