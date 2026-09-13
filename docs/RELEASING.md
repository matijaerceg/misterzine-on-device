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
release. Write it first.

`docs/RELEASE_NOTES.md` becomes the body of the GitHub release, and the
releases page shows every body in full, one after another. Write it for that
feed: a reader scanning the page decides from the headline and a few lines
whether the release matters to them. The target is under about 120 words.

- **Headline:** one sentence naming the one or two features a user would
  update for. Never a fix. A fixes-only release names its most visible fix and
  has no first list.
- **First list, no heading:** features and improvements only, one sentence each,
  no bold sub-heads. What the user will notice, not how it works.
- **Second list under `Fixes and adjustments:`** bugfixes, behaviour tweaks,
  label and copy changes, each one line. Copy changes collapse to a single
  line unless one changes what a setting does. Omit the list when empty.
- **Not on the card:** credits and early adopter names, timings and pixel
  sizes, internal mechanisms, test changes, catalogue refreshes. They belong
  in the changelog.
- **Footer, always the same three lines:** a link to the changelog section
  (anchor `#v1024--2026-09-13` for `## v1.0.24 — 2026-09-13`, lowercase with
  dots and the dash removed), the Update All line, and the installation guide
  link. Add a sentence about stored data only when the release migrates it.

The v1.0.24 body is the reference shape:

```markdown
Select + A stars a game from the list, and the Menu hold shows its
progress on the way to quitting.

- Select + A on the list stars or unstars the game under the cursor.
- The Menu button hold shows a "keep holding to quit" hint and a green
  progress line under the top bar. A tap still opens or closes Options.

Fixes and adjustments:

- The last view turned back on in Options -> Views no longer switches
  itself off at the next start.
- X on the Filters screen returns to the list, as B does.
- Options labels and hints tidied.

Full details in the [changelog](https://github.com/matijaerceg/misterzine-on-device/blob/main/CHANGELOG.md#v1024--2026-09-13).
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
