# MisterZine v1.0.36

Arcade games now warn when a required ROM archive is missing, instead of handing a known incomplete installation to MiSTer.

- Details shows the missing archive for the selected game version, such as **Missing game ROM: jpark.zip**.
- Start checks again before launching, including requirements in later ROM sections and alternative game versions.
- Checks follow MiSTer's storage search order and accept alternative archive names.
- Launch logs describe the request handed to MiSTer, rather than claiming the game loaded.

Also corrects the MiSTercade/JAMMA display-timing instructions and log hint for cabinets affected by a rolling picture.

These checks detect missing archives; they do not validate archive contents or guarantee playability. Card status continues to describe installation and core version.

Update through Update All, then quit and reopen MisterZine. Settings and favorites are preserved.
