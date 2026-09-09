This update fixes two installation problems:

- **Scripts launch correctly:** Setup and Uninstall now use the filenames
  `MisterZine-Setup.sh` and `MisterZine-Uninstall.sh`. MiSTer's Scripts launcher
  does not handle the spaces in the previous names. Update All/Downloader
  replaces those files during an update.
- **Global filters:** the installer INI now has an empty `filter =` under
  `[misterzine]`, so global core filters cannot exclude the app's files.

If your INI was recognised but nothing installed, download the current
`downloader_misterzine.ini` again, or add `filter =` to your existing
`[misterzine]` section, then rerun Update All. Downloader does not replace the
installer INI automatically. Your other databases retain their filters.

After installation, run **MisterZine-Setup** from Scripts once, then open
MisterZine from the main menu. See the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
