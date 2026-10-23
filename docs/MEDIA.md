> Video previews are shelved. The app currently offers screenshots and flyers only. Video preparation instructions below are retained for future experiments.

# Optional videos and flyers

MisterZine can advertise an optional gameplay preview and flyer for a game. A video uses the compact `MZV1` frame container; the MiSTer player decodes its JPEG frames directly. A flyer is a PNG. Both are separate from the existing screenshot fields, so older app versions ignore the new catalogue fields.

## Rights and source notes

Downloading a file does not grant permission to redistribute it. The reviewed collections do not provide a general licence that clearly permits MisterZine to mirror their arcade videos or scans. Get a permission or licence that covers public hosting, conversion/cropping, and the intended distribution before adding an asset to a public catalogue. Footage captured from a game can still contain copyrighted game material.

- [Progetto-SNAPS snapshots](https://www.progettosnaps.net/snaps_en.html) offers per-category MAME snapshot collections. Its [DAT index](https://www.progettosnaps.net/dats/) lists Flyers at 0.266 and VideoSnaps at 0.258, while Snap and Titles are at 0.289; video and flyer coverage may lag. Its [FAQ](https://www.progettosnaps.net/faq/) explains category folders and DAT-based checking/renaming. These are practical download and matching references, not proof of redistribution rights.
- [ScreenScraper WebAPI](https://www.screenscraper.fr/webapi2.php) documents image and video media endpoints, including flyer media. It says API integration is for free distributed apps unless ScreenScraper gives prior authorization, requires developers to present their software and obtain API credentials, and requires quota management. API access does not itself establish permission to mirror media; ask about the rights that apply to each asset.
- [EmuMovies downloads](https://emumovies.com/files/) offers artwork and video packs. Its [terms](https://emumovies.com/terms) say not to repost downloaded content to other sites, so treat it as a private user-library source unless you obtain separate written permission.
- [Flyercade](https://www.flyercade.com/about/) describes a curated physical flyer collection and says game artwork, logos, trademarks, and copyrights remain with their respective holders. No general reuse grant is stated there.
- [ArcadeDB terms](https://arcadedb.com/terms.html) reserve rights in site material, limit downloads to personal use, and restrict separating artwork from accompanying text. It is not a self-hosting source under those terms.

The project's catalogue licence also says screenshots are outside the catalogue and remain their owners' property ([LICENSE-CATALOGUE](../LICENSE-CATALOGUE)). Track the source and permission for each media file independently. The manifest's `permission` text is provenance supplied by the operator, not a legal certification or automated rights check.

## Local acquisition and hosting

There are already two hosting paths: catalogue screenshots live under the
site's `/images/`, while the [image service](https://github.com/matijaerceg/misterzine/tree/main/images)
uses the existing `misterzine-images` Cloudflare R2 bucket for per-setname
screenshots. That bucket is already seeded from Progetto-SNAPS snap/title packs.
Flyers use that image service by default, at `/flyer/<key>.png`, with PNG
objects stored in R2. The catalogue resolves game families and advertises only
available assets. The app caches downloads locally for offline use.

`internal/fetch.MediaService` can still be overridden at build time for a
different host. An empty value selects the site's `/images/` directory.
This repository's preparation tools do not publish objects or change hosting.

Acquire files from a source you are allowed to use, keeping the source files and permission records outside the generated asset directories. The tools below process local files; they do not download collections or publish files.

Prepare a gameplay capture with the local-only converter (it does not download or publish anything):

```sh
python tools/prepare_media.py video /path/to/input.mp4 --output-dir build/media --key pacman
```

This creates `build/media/video/pacman.mzv`: silent, looping JPEG frames at 15 fps, up to 20 seconds and 320 pixels on either side, capped at 8 MiB, in the `MZV1` container. The converter uses local `ffmpeg`; it does not contact an asset site. For a flyer, convert a permitted local image to a bounded PNG:

```sh
python tools/prepare_media.py flyer /path/to/flyer-source.jpg --output-dir build/media --key pacman
```

This creates `build/media/flyer/pacman.png`, scaled within 640×640 while preserving its aspect ratio. The catalogue tool and player reject flyers larger than 4 MiB.

Create a reviewed JSON manifest. `setnames` are exact MAME setnames; include a family root explicitly when the same media should be the fallback for its clones. An exact `sn` match takes precedence over the row's `family` match. Title matching is deliberately not used.

```json
[
  {
    "key": "pacman",
    "setnames": ["pacman"],
    "slots": ["video", "flyer"],
    "source_url": "https://example.org/asset-page",
    "credit": "Creator or collection credit",
    "permission": "Written permission, licence name, or other reviewed basis"
  }
]
```

The required manifest fields document source, credit, and the operator's permission record. `media_catalog.py` checks that the local output exists, but it cannot determine whether permission is valid. It rejects unsafe keys, unknown slots, duplicate setname assignments, and missing outputs. A family is the catalogue's explicit MAME `family` field, not a fuzzy title or inferred clone relationship.

Apply the manifest to a local catalogue JSON array:

```sh
python tools/media_catalog.py \
  --catalog build/data.json \
  --manifest media-manifest.json \
  --images build/media \
  --output build/data-with-media.json
```

The script validates the MZV1 frame table and JPEG dimensions, and the PNG chunk structure, checksums, and dimensions. It rejects files outside the format bounds and manifest entries that match no catalogue row. It writes only the output JSON and refuses to overwrite either input. It adds `media_key` and `media_slots` to matched rows and removes stale values from unmatched rows; existing `img` and `img_slots` are preserved. The same key may have a video, a flyer, or both. The catalogue advertises the files at `/video/<media_key>.mzv` and `/flyer/<media_key>.png` under the existing image base. On the device, files are cached as `shots/video/<media_key>.mzv` and `shots/flyer/<media_key>.png`.

Hosting is a separate release step: copy only reviewed files to those URL paths and publish the matching catalogue output through the normal catalogue release process. Regenerate `meta.json` with the output's SHA-256 and row count, otherwise the app rejects the changed catalogue. Publish the required source credits with the assets. The local scripts do not upload files, change the live service, or confirm rights.
