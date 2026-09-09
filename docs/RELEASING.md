# Release procedure

Normal installations follow the latest stable GitHub release. Development
builds identify themselves as `v1.0.0-dev`; the release workflow stamps the
actual tag, commit and build date.

## Before announcing v1.0.0

- Complete physical HDMI testing: menu entry, list/details/artwork, rotation,
  safe-zone behavior, launch, quit and reopening. Discuss any HDMI layout changes
  after that test. A denser layout is not part of this release preparation.
- Accept the final CRT/controller changes, including version selection,
  information paging and hold delay.
- Exercise packaged fresh installation, upgrade, both uninstall choices and
  reinstall on test cards/directories. Confirm saved favorites, preferences and
  filters survive an upgrade and the keep-data removal path.
- Check a normal launch with no debug flag/argument: no debug listener, input
  event logging or detailed frame measurements.
- Check an initial launch without network access against the embedded catalogue.
- Refresh the embedded snapshot shortly before the final build, then rerun checks.

The overnight long-idle check passed on September 9, 2026. HDMI acceptance and
final release approval are still required before the first announcement.

## Candidate and stable releases

Use `vX.Y.Z-rc.N` for a candidate and `vX.Y.Z` for a stable release. A candidate
is explicitly marked prerelease and is never promoted to GitHub's latest release.
The downloader database ID stays `misterzine` for every version.

Update `docs/RELEASE_NOTES.md` and the changelog before tagging, and remove the
README's preparation notice when v1.0.0 is approved. Run the full CI checks,
including the ARM build and deterministic render comparisons. The
release workflow calls the same CI workflow, packages exact staged assets, and
uploads a draft. A subsequent job downloads the draft assets and verifies their
database hashes, sizes and SHA256 checksums before publishing.

Only create/push a version tag when that release is approved. A normal code push
does not publish a release. Failed verification leaves a draft for investigation.

To test a candidate through Downloader, temporarily point the single existing
MisterZine database entry at that candidate's versioned database URL. Do not add
a duplicate database. Restore the stable URL afterward.

## Package contents

`tools/make-db.py TAG BINARY OUTPUT.json.zip` stages the launch wrapper,
one-time Setup and Uninstall Scripts entries, removal helper, MGL and licenses
alongside the binary. It also writes `SHA256SUMS`. The drop-in INI is supplied
as a release asset for users to copy; Downloader does not install its own
root-level configuration.

Favorites, preferences, filters, logs, tokens and debug flags are never package
contents. Developer test binaries and render outputs are also excluded.
