# Release procedure

Normal installations follow the latest stable GitHub release. Development
builds identify themselves with a `-dev` suffix; the release workflow stamps the
actual tag, commit and build date.

## Before publishing a release

- Recheck physical HDMI after the final UI changes: menu entry, list/details/artwork,
  rotation, safe zone, screensaver, launch, quit and reopening. The September 9
  HDMI walkthrough passed before the screensaver was added.
- Accept the final CRT/controller changes, including version selection,
  information paging and hold delay.
- Exercise packaged fresh installation, upgrade, both uninstall choices and
  reinstall on test cards/directories. Confirm saved favorites, preferences and
  filters survive an upgrade and the keep-data removal path.
- Check a normal launch with no debug flag/argument: no debug listener, input
  event logging or detailed frame measurements.
- Check an initial launch without network access against the embedded catalogue.
- Refresh the embedded snapshot shortly before the final build, then rerun checks.

Record the completed device checks with the release handoff. Final UI acceptance
and release approval are required before announcing.

## Candidate and stable releases

Use `vX.Y.Z-rc.N` for a candidate and `vX.Y.Z` for a stable release. A candidate
is explicitly marked prerelease and is never promoted to GitHub's latest release.
The downloader database ID stays `misterzine` for every version.

Update the changelog and `docs/RELEASE_NOTES.md` before tagging, following the
release card rules below. Run the full CI checks,
including the ARM build and deterministic render comparisons. The
release workflow calls the same CI workflow, packages exact staged assets, and
uploads a draft. A subsequent job downloads the draft assets and verifies their
database hashes, sizes and SHA256 checksums before publishing.

Only create/push a version tag when that release is approved. A normal code push
does not publish a release. Failed verification leaves a draft for investigation.

To test a candidate through Downloader, temporarily point the single existing
MisterZine database entry at that candidate's versioned database URL. Do not add
a duplicate database. Restore the stable URL afterward.

## The changelog and the release card

The changelog is the complete record: every change, the reason for a fix and
the mechanism behind it, in the `## vX.Y.Z — YYYY-MM-DD` section for the
release. Write it first. Even there, write for a reader of the repository,
not of the code: it is the screensaver, never the saver.

`docs/RELEASE_NOTES.md` becomes the body of the GitHub release, and the
releases page shows every body in full, one after another. Its reader is a
MiSTer user scanning that feed, often one who has never installed
MisterZine, deciding in a few lines whether this is worth having. The card
is a pitch and a short success story, not a ledger of changes: it says what
they get and why they would want it, in the words they would use
themselves. It stays short, about 150 words with the footer.

- **Words:** everyday ones. Screensaver, slide, game, card, Options, the
  bottom bar. Never the code's or the guide's internal names: saver, pane,
  tab, strip, hue, legend, glyph, quantiser. A phrase that needs the app in
  front of you to make sense ("the pane's lines") is left out or replaced
  by what it means to the reader ("game details"). Never a term a new
  reader has no context for.
- **Headline:** a short title naming the feature as a benefit, the way a
  product update is announced: "More info on Screenshots Screensaver". Not
  a sentence of mechanics, and never a fix. A release without a feature
  names its most visible fix and skips the paragraph.
- **The paragraph:** three or four sentences that sell the feature. What it
  looks like in use, what it does for them, how to reach it and how to set
  it, with the choices in plain words ("full details, the title only, or
  nothing but the picture"). Paint the scene ("leave MisterZine idle and it
  becomes a tour of the arcade catalogue"); do not list mechanisms.
- **What counts as a feature:** something a user could not do or see
  before and would update for on its own: a new view, screen, control,
  option, source, or a new visual. A control that now works in one view as
  it already did in others is consistency, not a feature. Wording, labels,
  help text, spacing, colours, timings, defaults and any tuning or reshaping
  of something that already existed are adjustments. When in doubt it is an
  adjustment.
- **`Also in this release:`** everything else, one plain line each:
  bugfixes, behaviour tweaks, consistency across views, label and copy
  changes. Copy changes collapse to a single line unless one changes what a
  setting does. Omit the list when empty.
- **Not on the card:** credits and early adopter names, timings and pixel
  sizes, internal mechanisms, test changes, catalogue refreshes, board
  measurements. They belong in the changelog.
- **Footer, always the same three lines:** a link to the changelog section
  (anchor `#v1028--2026-09-13` for `## v1.0.28 — 2026-09-13`, lowercase with
  dots and the dash removed), the Update All line, and the installation guide
  link. Add a sentence about stored data only when the release migrates it.

The v1.0.28 body is the reference shape. Its first version was rejected as
"too LLMy": it opened with "The screenshots saver types the pane's details
about the game onto each shot", listed the Options L and R section jump as a
feature, and read as facts rather than a reason to update.

```markdown
More info on Screenshots Screensaver

The screenshots screensaver now types out game details onto each slide, one character at a time behind a blinking cursor: the title, whether the game is on your card, the core, the year and maker, the controls. Leave MisterZine idle and it becomes a tour of the arcade catalogue, a new game sliding in every few seconds with its details written underneath. Hold Start on any slide to play that game. Choose how much it says in Options: full details, the title only, or nothing but the picture.

Also in this release:

- The lettering screensaver has a retro low-colour look, dithered like an old display, and it draws faster.
- L and R jump between sections in Options, as they already do in the list and in Filters.
- Options rows carry their full names, dependent rows sit indented under their parent, and the bottom bar says what each button does on the selected row.

Full details in the [changelog](https://github.com/matijaerceg/misterzine-on-device/blob/main/CHANGELOG.md#v1028--2026-09-13).
Existing installations update through Update All. New installation:
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
```

## Package contents

`tools/make-db.py TAG BINARY OUTPUT.json.zip` stages the launch wrapper,
one-time Setup and Uninstall Scripts entries, removal helper, MGL and licenses
alongside the binary. It also writes `SHA256SUMS`. The drop-in INI is supplied
as a release asset for users to copy; Downloader does not install its own
root-level configuration.

Favorites, preferences, filters, logs, tokens and debug flags are never package
contents. Developer test binaries and render outputs are also excluded.
