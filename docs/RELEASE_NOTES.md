The card scan no longer reports itself incomplete over alternative MRAs it
cannot read.

- **Unreadable alternatives:** an MRA under `_Arcade/_alternatives` that
  holds no readable header (an empty file, one cut short, one without
  `<rbf>`) is skipped: it is left out of the version picker, as MiSTer
  could not load it either, named in the log the first time its folder is
  read, and counted on every scan (`N unreadable MRAs skipped`). The scan
  itself completes and the result screen shows its totals.
- **Seibu SPI sets:** MRAs with `--` inside an XML comment (Raiden
  Fighters, Senkyu, Viper Phase 1) are now read, so their alternatives
  appear in the picker.
- **Real read failures:** "Card scan incomplete" is kept for a folder or
  file that could not be read at all; the result screen then keeps its
  totals with the problem under them, and the log names the path.
- `watch.log` rotates at 1 MiB, as `log.txt` already did.

Nothing changes on the card outside `misterzine/`. Existing settings,
favorites, filter choices, remembered versions and the launch history are
preserved.

Existing installations can update through Update All/Downloader. Reopen
MisterZine after updating.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
