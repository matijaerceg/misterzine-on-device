Accurate card status: compared against the shipped core build, with Jotego cores matched by checksum.

- **Shipped-build comparison:** the on-card status now compares each core file
  with the build the catalogue actually ships, using its build date and
  checksum, instead of the catalogue's last-updated date. A core that only
  received an MRA fix, or whose release landed after midnight UTC, no longer
  shows as an older build when Update All has nothing newer to install.
- **Undated cores resolved:** cores whose filenames carry no date (Jotego's,
  for example) are matched by checksum against the shipped build, using the
  record Update All keeps on the card. A match shows as current. A mismatch
  shows as “older build likely”, the same evidence Update All acts on, and
  counts with older builds in the card summary and the install filter.
  “Date unknown” now only appears when the catalogue has no comparison data.
- **Folder-aware matching:** arcade games only match cores in `_Arcade/cores`,
  and console/computer entries only match their own folders, so a shared core
  name (Astrocade ships both) no longer reports the wrong file.

The catalogue at misterzine.fyi already publishes the new build fields; no
action is needed beyond updating.

Existing installations can update through Update All/Downloader. Reopen
MisterZine after updating. Favorites, settings and filter choices are preserved.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
