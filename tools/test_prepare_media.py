import importlib.util
from pathlib import Path
import struct
import tempfile
import unittest
from unittest import mock


SPEC = importlib.util.spec_from_file_location("prepare_media", Path(__file__).with_name("prepare_media.py"))
prepare_media = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(prepare_media)


def fake_jpeg(width=4, height=3):
    # The preparation tool only needs the standard SOF dimensions; device-side
    # parsing performs full JPEG validation before a clip can be displayed.
    return b"\xff\xd8\xff\xc0" + struct.pack(">H B H H B", 8, 8, height, width, 1) + b"\xff\xd9"


def fake_png(width=10, height=20):
    return b"\x89PNG\r\n\x1a\n" + struct.pack(">I", 13) + b"IHDR" + struct.pack(">II", width, height) + b"x" * 20


class PrepareMediaTests(unittest.TestCase):
    def test_mzv_header_and_records(self):
        frame = fake_jpeg()
        data = prepare_media._encode_mzv([frame, frame])
        self.assertEqual(data[:4], b"MZV1")
        self.assertEqual(struct.unpack_from("<HHHH", data, 4), (4, 3, 15, 2))
        self.assertEqual(struct.unpack_from("<I", data, 12)[0], len(frame))

    def test_video_preparation_uses_local_frames_and_slot_directory(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "source.mp4"
            source.write_bytes(b"local")

            def fake_run(command):
                Path(command[-1].replace("%03d", "001")).write_bytes(fake_jpeg())

            with mock.patch.object(prepare_media, "_run", side_effect=fake_run):
                output = prepare_media.prepare_video(source, root / "out", "demo")
            self.assertEqual(output, root / "out" / "video" / "demo.mzv")
            self.assertTrue(output.read_bytes().startswith(b"MZV1"))

    def test_flyer_preparation_preserves_png_output(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "source.jpg"
            source.write_bytes(b"local")

            def fake_run(command):
                Path(command[-1]).write_bytes(fake_png())

            with mock.patch.object(prepare_media, "_run", side_effect=fake_run):
                output = prepare_media.prepare_flyer(source, root / "out", "cover")
            self.assertEqual(output, root / "out" / "flyer" / "cover.png")
            self.assertEqual(prepare_media._png_size(output.read_bytes()), (10, 20))

    def test_oversize_flyer_is_not_written(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "source.jpg"
            source.write_bytes(b"local")

            def fake_run(command):
                data = fake_png() + b"x" * prepare_media.MAX_FLYER_BYTES
                Path(command[-1]).write_bytes(data)

            with mock.patch.object(prepare_media, "_run", side_effect=fake_run):
                with self.assertRaises(ValueError):
                    prepare_media.prepare_flyer(source, root / "out", "cover")
            self.assertFalse((root / "out" / "flyer" / "cover.png").exists())

    def test_cli_rejects_remote_input_and_path_keys(self):
        with self.assertRaises(SystemExit):
            prepare_media.main(["video", "https://example.test/a.mp4", "--output-dir", ".", "--key", "ok"])
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "a.mp4"
            source.write_bytes(b"x")
            for key in ("../bad", "Uppercase", "has.dot"):
                with self.subTest(key=key), self.assertRaises(SystemExit):
                    prepare_media.main(["video", str(source), "--output-dir", temporary, "--key", key])


if __name__ == "__main__":
    unittest.main()
