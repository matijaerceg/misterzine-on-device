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

## Beta releases

MisterZine Arcade, the Patreon members' beta, is this source with the members'
extras from the private repository `misterzine-arcade-private`, built with
`-tags arcade` and `internal/beta.Channel=beta`; every beta-only feature shows
itself only when `beta.On()` is true. The members' release there assembles the
two, runs this repository's checks and its own over them, builds, packages
with this repository's `tools/make-db.py` and `tools/verify_package.py`, and
publishes on `matijaerceg/misterzine-arcade-betas`, a public repository that
holds built files only. Its README is the members' guide. The procedure,
the access batch and the matching beta after each free release are in
`misterzine-arcade-private`'s README. This repository's release workflow
refuses `-beta.` tags.

The beta installs as the same Downloader database, `misterzine`, so it
replaces the free files in place and Update All keeps working. Members'
entries follow fixed addresses on the betas repository's `main` branch:

- `https://raw.githubusercontent.com/matijaerceg/misterzine-arcade-betas/main/beta.json.zip`,
  the newest beta's database (`buildinfo.BetaDBURL`);
- `.../main/catalogue.json`, its version and batch, which beta builds read for
  their **App update** notice (`buildinfo.BetaCatalogueURL`);
- `.../main/MisterZine-Install-Beta.sh` and `MisterZine-Switch-To-Free.sh`.

A beta package adds `misterzine/channel.py` and
`Scripts/MisterZine-Switch-To-Free.sh` to the database, with
`MisterZine-Install-Beta.sh` and a drop-in pointed at the beta as release
assets only; its files are published under the betas repository's releases.
`make-db.py --distribution` writes the files its `main` branch serves, refuses
a beta older than the one served, and leaves its README alone.
`buildinfo.FreeDBURL` and `BetaDBURL` name the two databases, and a test keeps
`deploy/channel.py`, the free drop-in and `tools/verify_package.py` in step
with them.

A beta build offers only newer betas. A newer free release is not offered to
members: their entry follows the beta database, and taking the free one would
drop the beta features. So every free release gets a matching beta from the
same commit or later, so members are never behind. To offer free releases to
beta builds instead, change `buildinfo.CheckBetaUpdate`.

### Promoting a feature to free

Delete the feature's `beta.On()` check (a members' extra also moves from the
private overlay to this repository), render its free goldens and inspect
them, remove the `-beta` scenarios that only covered it, and ship it in the
next free release. Then cut the next beta from that release or later.

### Between the free version and the beta

`MisterZine-Install-Beta` runs `channel.py beta` and `MisterZine-Switch-To-Free`
runs `channel.py free`: each points the `misterzine` entry at its database
and runs Downloader for it, which replaces the files in place; going free
removes the beta-only files. Both edit only the entry's `db_url`, wherever it
is: the drop-in, another drop-in or a `downloader.ini` section; a card with no
entry gets the standard drop-in. The main-menu entry, `MisterZine Arcade.mgl`
in both builds, stays; one still under its old name, `MisterZine.mgl`, takes
the new name.

The app has no way to the free version of its own. Two guards keep a card
from sliding between the builds by accident, since MiSTer Companion's Install
Center rewrites the `misterzine` entry to the free database
(`cmd/misterzine/channel_guard.go`):

- An unlocked beta whose entry follows the free database, or has none, points
  it back with `channel.py beta --point-only` and says so.
- The free build, on a card that holds a beta unlock (`misterzine/beta-unlocks/`)
  and whose entry follows the free database, asks once whether to go back;
  holding A runs the member's own `Scripts/MisterZine-Install-Beta.sh`
  (`updater.ModeBeta`), which stays in Scripts after going free because no
  database installs it.
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
themselves. The opening stays short, and the lists below it carry the rest
in one line each.

- **Words:** everyday ones. Screensaver, slide, game, card, Options, the
  bottom bar. Never the code's or the guide's internal names: saver, pane,
  tab, strip, hue, legend, glyph, quantiser. A phrase that needs the app in
  front of you to make sense ("the pane's lines") is left out or replaced
  by what it means to the reader ("game details"). Never a term a new
  reader has no context for.
- **Headline:** in bold, a plain name for what got better, the way a
  product update is announced: "Better local file handling", "Update
  MisterZine by itself". Not a slogan or a sweeping promise ("Every version
  your card can play" became "Better local file handling"), not a sentence
  of mechanics, and never a fix. A release without a feature names its most
  visible fix and skips the opening. No release name or tagline.
- **The opening:** one or two sentences saying what is new and what it
  does for them. A concrete case goes in its own short paragraph starting
  "For example:" ("For example: Pleiads plays on the Phoenix core, so its
  Centuri and bootleg sets ... used to be out of reach"). Say how to reach
  something new and how to set it, with the choices in plain words ("full
  details, the title only, or nothing but the picture"); do not re-explain
  controls that already worked that way before (v1.1.2 dropped "Press Left
  and Right to choose, Start to play", which the version picker already
  did). On-screen labels named here are all bold alike: **Update MisterZine
  only** above **Run Update All**. Do not list mechanisms.
- **What counts as a feature:** something a user could not do or see
  before and would update for on its own: a new view, screen, control,
  option, source, or a new visual. A control that now works in one view as
  it already did in others is consistency, not a feature. Wording, labels,
  help text, spacing, colours, timings, defaults and any tuning or reshaping
  of something that already existed are adjustments. When in doubt it is an
  adjustment.
- **Added and Fixed:** everything besides the headline feature, one plain
  line each, under the bold labels `**Added**` and `**Fixed**`. Added holds
  what a user can now do or see that they could not before: other features,
  new controls, sources, help screens.
  Fixed holds bugfixes, behaviour tweaks, consistency across views, and
  label and copy changes. Copy changes collapse to a single line unless one
  changes what a setting does. Leave out a heading whose list is empty.
- **Not on the card:** credits and early adopter names, timings and pixel
  sizes, internal mechanisms, test changes, catalogue refreshes, board
  measurements. They belong in the changelog. The one exception is a thanks
  to whoever reported a fix, one short sentence closing the line it belongs
  to ("Thanks to Mclaneinc for reporting it.").
- **Footer, always the same three lines:** a link to the changelog section
  (anchor `#v112--2026-09-22` for `## v1.1.2 — 2026-09-22`, lowercase with
  dots and the dash removed), the update line, and the installation guide
  link. Add a sentence about stored data only when the release migrates it.

The v1.1.2 body, as published, is the reference shape. Its draft had the
headline "Every version your card can play", the example folded into the
first paragraph, and a sentence on the controls the version picker already
had. Earlier, the first v1.0.28 card was rejected as
"too LLMy": it opened with "The screenshots saver types the pane's details
about the game onto each shot", listed the Options L and R section jump as a
feature, and read as facts rather than a reason to update.

```markdown
**Better local file handling**

Open a game's details and every version of it on your card is there to pick: the regional sets and bootlegs, copies kept in other folders, and now the sets that run on a different core from the game itself.

For example: Pleiads plays on the Phoenix core, so its Centuri and bootleg sets, which need the Pleiads core, used to be out of reach; now they sit in the same list, each marked with the core it runs on.

**Added**

- Send a report, in Options -> Troubleshooting: when a game is missing or something looks wrong, it sends the developer a description of your card and gives you a short code to post where you asked for help.

**Fixed**

- On 1440p and 4K screens MisterZine fills the display instead of a stretched band across the middle.
- Full display keeps text at about the same size on every screen, so 720p gets fewer, larger rows.
- The missing-ROM warning looks inside your zips the way MiSTer does, and spots a mame folder at the top of the SD card that stops every arcade game.
- A playable copy of a game the catalogue has only for a core you don't own sits right under its greyed row in every order. Thanks to Mclaneinc for reporting it.

Full details in the [changelog](https://github.com/matijaerceg/misterzine-on-device/blob/main/CHANGELOG.md#v112--2026-09-22).
Existing installations update through Update All, or through Update MisterZine only. New installation:
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
```

## Package contents

`tools/make-db.py TAG BINARY OUTPUT.json.zip` stages the launch wrapper,
one-time Setup and Uninstall Scripts entries, removal helper, MGL and licenses
alongside the binary. It also writes `SHA256SUMS`. The drop-in INI is supplied
as a release asset for users to copy; Downloader does not install its own
root-level configuration.

A beta tag also packages `channel.py` and `MisterZine-Switch-To-Free.sh`, and
stages `MisterZine-Install-Beta.sh` and a drop-in pointed at the beta database
as release assets, all addressed on the betas repository's releases.
`tools/verify_package.py` refuses a beta database under a free tag and the
reverse, and files addressed on the other build's repository.

Favorites, preferences, filters, logs, tokens and debug flags are never package
contents. Developer test binaries and render outputs are also excluded.
