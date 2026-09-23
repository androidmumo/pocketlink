import base64
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "tools"))
spec = importlib.util.spec_from_file_location("bundle", ROOT / "tools/check-release.py")
bundle = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bundle)

class BundleTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.folder = Path(self.temp.name)
        for name in bundle.FILES + ["SHA256SUMS"]:
            shutil.copyfile(ROOT / "firmware/releases/0.5.0" / name, self.folder / name)
    def verify(self):
        with contextlib.redirect_stdout(io.StringIO()):
            return bundle.verify(self.folder)
    def test_published_original(self):
        self.assertEqual(self.verify()["version"], "0.5.0")
    def test_corrupt_ota_rejected(self):
        p = self.folder / "pocketlink-ota.bin"
        b = bytearray(p.read_bytes()); b[-1] ^= 1; p.write_bytes(b)
        with self.assertRaises(AssertionError): self.verify()
    def test_identity_region_rejected(self):
        p = self.folder / "pocketlink-full.bin"
        p.write_bytes(p.read_bytes() + bytes(0x40000))
        with self.assertRaises(AssertionError): self.verify()
    def test_mismatched_full_app_rejected(self):
        p = self.folder / "pocketlink-full.bin"
        b = bytearray(p.read_bytes()); b[0x10010] ^= 1; p.write_bytes(b)
        with self.assertRaises(AssertionError): self.verify()
    def test_invalid_signature_rejected(self):
        p = self.folder / "manifest.json"; v = json.loads(p.read_bytes())
        b = bytearray(base64.b64decode(v["signature"])); b[-1] ^= 1
        v["signature"] = base64.b64encode(b).decode(); p.write_text(json.dumps(v))
        with self.assertRaises(subprocess.CalledProcessError): self.verify()
    def test_wrong_checksums_rejected(self):
        (self.folder / "SHA256SUMS").write_text("invalid")
        with self.assertRaises(AssertionError): self.verify()

if __name__ == "__main__": unittest.main()
