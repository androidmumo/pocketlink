import importlib.util
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
import urllib.error

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "tools"))
spec = importlib.util.spec_from_file_location("publisher", ROOT / "tools/publish-release.py")
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)

class PublishTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(); self.addCleanup(self.temp.cleanup)
        self.folder = Path(self.temp.name)
        for name in publisher.checker.FILES + ["SHA256SUMS"]:
            (self.folder / name).write_bytes(b"test")
        self.calls = []; self.existing = False; self.fail_upload = False
    def request(self, request, timeout):
        self.calls.append(request)
        url, method = request.full_url, request.method
        if method == "GET" and not self.existing:
            raise urllib.error.HTTPError(url, 404, "not found", None, None)
        if method == "GET": data = {"id": 1}
        elif url.startswith("https://uploads.github.com/"):
            if self.fail_upload: raise urllib.error.HTTPError(url, 500, "upload failed", None, None)
            data = {"state": "uploaded", "size": 4}
        elif method == "PATCH": data = {"draft": False, "html_url": "https://github.com/test/repo/releases/tag/test"}
        else: data = {"id": 1, "upload_url": "https://uploads.github.com/repos/test/repo/releases/1/assets{?name}"}
        return io.BytesIO(json.dumps(data).encode())
    def run_publish(self):
        with patch.dict(os.environ, {"GH_TOKEN": "synthetic-test-only", "GITHUB_REPOSITORY": "test/repo"}), patch.object(publisher.checker, "verify", return_value={"version": "0.5.0", "sequence": 5}), patch.object(publisher.urllib.request, "urlopen", side_effect=self.request):
            publisher.publish(self.folder, "a" * 40, "0.5.0")
    def test_existing_release_is_never_overwritten(self):
        self.existing = True
        with self.assertRaises(RuntimeError): self.run_publish()
        self.assertEqual([r.method for r in self.calls], ["GET"])
    def test_failed_upload_stays_draft(self):
        self.fail_upload = True
        with self.assertRaises(urllib.error.HTTPError): self.run_publish()
        self.assertNotIn("PATCH", [r.method for r in self.calls])
        create = json.loads(self.calls[1].data)
        self.assertTrue(create["draft"])
    def test_all_assets_upload_before_publication(self):
        self.run_publish()
        uploads = [r for r in self.calls if r.full_url.startswith("https://uploads.github.com/")]
        self.assertEqual(len(uploads), 4)
        self.assertEqual(self.calls[-1].method, "PATCH")
        self.assertFalse(json.loads(self.calls[-1].data)["draft"])

if __name__ == "__main__": unittest.main()
