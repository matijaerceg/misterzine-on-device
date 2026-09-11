Narrow list titles, a remembered version per game, a Year order and a tidier
Options page.

- **Narrow titles:** list titles use a narrow proportional font (scientifica,
  SIL Open Font License), so a row fits about a third more of a title. The
  default "narrow tall" face matches the body font's height; Options -> Title
  font also offers the original "narrow" face and "normal". The status and
  date columns use the narrow font at the titles' height.
- **Beta cores:** Patreon beta cores carry a yellow β after their title, and
  the pane's "beta" chip is yellow to match. Filters -> Type -> Arcade opens
  into Stable and Beta, so beta cores can be hidden or shown alone.
- **Date format and list shots:** Options -> Date format chooses MM-DD,
  DD-MM, Mon D, D Mon or YYMMDD for the list dates; Options -> List shots
  chooses a gameplay shot (default) or the title screen for the pane.
- **Remembered version:** the version chosen in Details is remembered for
  that game across restarts. Details opens on it and Start launches it from
  the list and from artwork. Choosing the main version again forgets the
  choice; an alternative no longer on the card falls back to the main one.
- **Details:** Left/Right choose the version and Up/Down page the
  information, which slides by the pixel with a green scrollbar thumb. A
  version name too long for its line scrolls back and forth. The legend names
  Left/Right only for games with more than one version and Up/Down only when
  the information scrolls.
- **Year order:** Y now cycles core updated, MiSTer debut, original year, A-Z
  and Favorites A-Z, and the top bar names the order. Original year lists the
  newest release year first, titles alphabetical within a year and unknown
  years last; L/R jump between years.
- **List paging:** Left/Right move a screen of rows at a time and keep the
  selected row centered, like stepping.
- **Options and Filters:** Options is grouped under Data, Display and
  Operation headings, scrolls with the selection centered, and shows small
  edge arrows only while entries sit out of view; the help text is framed and
  the build version and catalogue date sit in grey above the hint bar.
  Filters always lists Clear all filters at the top, greyed out until a filter
  is active, and A on a heading opens or closes it.

New settings (title font, list shots, date format) start at their defaults;
existing settings, favorites and filter choices are preserved. The remembered
versions are kept in state.json.

Existing installations can update through Update All/Downloader. Reopen
MisterZine after updating.

For a new installation, follow the
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
Run **MisterZine-Setup** from Scripts once, then open MisterZine from the main menu.
