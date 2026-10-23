import shutil
import struct
import json
import tempfile
import unittest
from pathlib import Path
try:
    from tools import media_catalog
except ModuleNotFoundError:
    import media_catalog


MODULE_PATH = Path(__file__).with_name("media_catalog.py")


def manifest_entry(key="asset", setnames=None, slots=None):
    return {
        "key": key,
        "setnames": setnames or ["parent"],
        "slots": slots or ["video"],
        "source_url": "https://example.invalid/source",
        "credit": "Contributor",
        "permission": "Written permission recorded in project archive",
    }


class MediaCatalogTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.images = Path(self.temp.name)
        (self.images / "video").mkdir()
        (self.images / "flyer").mkdir()
        fixtures = MODULE_PATH.parent.parent / "testdata" / "images"
        shutil.copyfile(fixtures / "video" / "media-demo.mzv", self.images / "video" / "asset.mzv")
        shutil.copyfile(fixtures / "video" / "media-demo.mzv", self.images / "video" / "clone.mzv")
        shutil.copyfile(fixtures / "flyer" / "media-demo.png", self.images / "flyer" / "asset.png")

    def test_exact_setname_precedes_family_fallback(self):
        catalog = [
            {"sn": "clone", "family": "parent", "img": "old", "img_slots": ["snap"]},
            {"sn": "other-clone", "family": "parent"},
        ]
        manifest = [manifest_entry("asset", ["parent"]), manifest_entry("clone", ["clone"])]
        got = media_catalog.apply_manifest(catalog, manifest, self.images)
        self.assertEqual((got[0]["media_key"], got[0]["media_slots"]), ("clone", ["video"]))
        self.assertEqual((got[1]["media_key"], got[1]["media_slots"]), ("asset", ["video"]))
        self.assertEqual(got[0]["img"], "old")
        self.assertEqual(got[0]["img_slots"], ["snap"])

    def test_unmatched_rows_keep_legacy_fields_and_drop_stale_media(self):
        catalog = [{"sn": "unknown", "img": "legacy", "media_key": "stale", "media_slots": ["flyer"]}]
        got = media_catalog.apply_manifest(catalog, [], self.images)
        self.assertEqual(got, [{"sn": "unknown", "img": "legacy"}])

    def test_missing_local_asset_is_rejected(self):
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "missing video asset"):
            media_catalog.apply_manifest([], [manifest_entry("missing")], self.images)

    def test_path_like_key_is_rejected(self):
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "unsafe key"):
            media_catalog.apply_manifest([], [manifest_entry("../escape")], self.images)

    def test_unknown_slot_is_rejected(self):
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "unknown slot"):
            media_catalog.apply_manifest([], [manifest_entry(slots=["snap"])], self.images)

    def test_unhashable_unknown_slot_is_rejected_cleanly(self):
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "unknown slot"):
            media_catalog.apply_manifest([], [manifest_entry(slots=[["video"]])], self.images)

    def test_duplicate_setname_is_rejected_as_ambiguous(self):
        entries = [manifest_entry("asset", ["parent"]), manifest_entry("clone", ["parent"])]
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "ambiguous setname 'parent'"):
            media_catalog.apply_manifest([], entries, self.images)

    def test_flyer_slot_uses_png_output_and_is_preserved(self):
        catalog = [{"sn": "parent", "img": "shot", "img_slots": ["title", "snap"]}]
        got = media_catalog.apply_manifest(catalog, [manifest_entry(slots=["flyer"])], self.images)
        self.assertEqual(got[0]["media_slots"], ["flyer"])
        self.assertEqual(got[0]["img"], "shot")
        self.assertEqual(got[0]["img_slots"], ["title", "snap"])

    def test_unused_manifest_entry_is_rejected(self):
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "match no catalogue sn or family"):
            media_catalog.apply_manifest([{"sn": "known", "family": "known"}], [manifest_entry("asset", ["typo"])], self.images)

    def test_cli_refuses_to_overwrite_manifest(self):
        catalog_path = self.images / "catalog.json"
        manifest_path = self.images / "manifest.json"
        catalog_path.write_text(json.dumps([{"sn": "parent"}]), encoding="utf-8")
        manifest_path.write_text(json.dumps([manifest_entry()]), encoding="utf-8")
        with self.assertRaises(SystemExit):
            media_catalog.main([
                "--catalog", str(catalog_path), "--manifest", str(manifest_path),
                "--images", str(self.images), "--output", str(manifest_path),
            ])
        self.assertEqual(json.loads(manifest_path.read_text(encoding="utf-8"))[0]["key"], "asset")

    def test_malformed_mzv_is_rejected(self):
        (self.images / "video" / "asset.mzv").write_bytes(b"MZV1" + struct.pack("<HHHH", 321, 1, 15, 1))
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "invalid MZV1 dimensions"):
            media_catalog.apply_manifest([{"sn": "parent"}], [manifest_entry()], self.images)

    def test_truncated_mzv_frame_is_rejected(self):
        video_path = self.images / "video" / "asset.mzv"
        video_path.write_bytes(video_path.read_bytes()[:-1])
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "truncated or empty MZV1 frame"):
            media_catalog.apply_manifest([{"sn": "parent"}], [manifest_entry()], self.images)

    def test_png_crc_corruption_is_rejected(self):
        flyer_path = self.images / "flyer" / "asset.png"
        data = bytearray(flyer_path.read_bytes())
        data[20] ^= 0x01
        flyer_path.write_bytes(data)
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "bad PNG CRC"):
            media_catalog.apply_manifest([{"sn": "parent"}], [manifest_entry(slots=["flyer"])], self.images)

    def test_truncated_png_is_rejected(self):
        flyer_path = self.images / "flyer" / "asset.png"
        flyer_path.write_bytes(flyer_path.read_bytes()[:-5])
        with self.assertRaisesRegex(media_catalog.MediaCatalogError, "truncated PNG chunk"):
            media_catalog.apply_manifest([{"sn": "parent"}], [manifest_entry(slots=["flyer"])], self.images)


if __name__ == "__main__":
    unittest.main()
