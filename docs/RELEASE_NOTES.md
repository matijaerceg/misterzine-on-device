The picture fills a 1080p screen, the pad's Menu button is MisterZine's,
and Select gives you quick toggles.

- **Fit display:** Options -> Canvas, on by default. Where MiSTer's integer
  scaling of a 320x240 picture left bars above and below, MisterZine now
  draws at a size that fills the screen height: 360x270 on 1080p, 340x256
  on 1024x768, 400x300 on 1600x900. Text keeps its size; the list gains
  rows. CRT modes, 720p and 1440p already fit and stay 320x240, and the
  320x240 choice keeps the classic size everywhere. Applies at the next
  start.
- **Menu button:** a pad defined in MiSTer whose A and B MisterZine can read
  is held while MisterZine runs, so MiSTer sees nothing from it. The button
  you defined as MiSTer's menu (OSD) button is now MisterZine's Menu
  button: it opens Options from any screen, or closes it, and Options ->
  Menu button can make it leave to the MiSTer menu instead, as before.
  Keyboard F12 and the board's button still leave.
- **Partial definitions:** a pad whose A or B MisterZine cannot read is left
  to MiSTer's translation, as it was before v1.0.17, so an incomplete
  definition can no longer lose the back button. The pad tester says which
  pads are held and shows the Menu button.
- **Quick toggles:** hold Select on the list and Y cycles List layout while
  X switches List shots; the legend names the chords while Select is held.
- The main list legend now reads A B X Y.

Nothing changes in MiSTer.ini, the menu core or your MiSTer controller
definitions; MisterZine only reads the map files MiSTer already saves and
lets go of every pad when it closes. Existing settings, favorites, filter
choices, remembered versions and the launch history are preserved.

Existing installations can update through Update All/Downloader. Reopen
MisterZine after updating.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
