Rotation follows the INI you actually use.

- **Active INI:** Follow INI rotation now reads the MiSTer INI that is
  currently active, the main MiSTer.ini or the alternative `MiSTer_*.ini` you
  selected in the MiSTer OSD, resolved the same way Main resolves it. Until
  now only MiSTer.ini was read, so an `osd_rotate` set in an alternative INI
  was ignored. The startup log names the file it used.
- **Clearer rotation options:** Options lists Follow INI rotation first, then
  Rotation, then the filter. Rotation is greyed out and locked while
  following is on; turn following off and Rotation becomes your saved manual
  choice.
- **Filter by rotation:** the strict orientation filter now follows the
  interface's current rotation, whatever set it, INI or manual, and updates
  the moment you rotate. Your saved on/off choice carries over.

Existing installations can update through Update All/Downloader. Reopen
MisterZine after updating. Favorites, settings and filter choices are preserved.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
