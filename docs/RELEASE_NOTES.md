Automatic INI rotation, optional matching-game filtering, and faster filter selection.

- **Follow INI rotation:** enabled by default, including after upgrading.
  Each startup reads MiSTer.ini orientation. You can still rotate the current
  session manually; turn off Options -> Follow INI rotation to retain your
  chosen orientation across launches. MisterZine never edits the INI.
- **Strict orientation filter:** Options -> Filter by INI rotation is a separate,
  default-off option. It shows only entries with a known orientation matching
  the INI, hiding unknown and opposite orientations, including in Favorites.
  It acts independently of interface rotation. Turning it off restores your
  manual filters; search and other filter choices remain intact.
- **Y only/all:** press Y on a filter value to isolate it within its section.
  Press Y again on that value to enable all values in the section. Other
  sections are unchanged. A still toggles individual values, and the legend
  shows the Y shortcut.
- **Clearer last-look feedback:** the marker says No changes in top 200 when
  only that window was checked, or No changes in this view when the complete
  view was checked. The Since last look filter still checks the full catalogue.

Existing installations can update through Update All/Downloader. Favorites,
settings and filter choices are preserved.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
