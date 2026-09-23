#!/usr/bin/env python3
"""Validate a distributable firmware bundle without using a private key."""
import base64
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from verify_firmware import verify_protected_layout

FILES = ["pocketlink-full.bin", "pocketlink-ota.bin", "manifest.json"]

def verify(folder):
    envelope = json.loads((folder / "manifest.json").read_bytes())
    payload = base64.b64decode(envelope["payload"], validate=True)
    signature = base64.b64decode(envelope["signature"], validate=True)
    metadata = json.loads(payload)
    assert re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,31}", metadata["version"])
    assert metadata["format"] == 1 and metadata["board"] == "ai-passport-esp32c3"
    assert metadata["app"] == "pocketlink" and 1 <= metadata["sequence"] <= 2147483647
    app = (folder / "pocketlink-ota.bin").read_bytes()
    full = (folder / "pocketlink-full.bin").read_bytes()
    assert len(app) == metadata["size"] and hashlib.sha256(app).hexdigest() == metadata["sha256"]
    assert full[0x10000:0x10000 + len(app)] == app
    # Never distribute a full image reaching cardid or pocketcfg.
    assert len(full) == 0x312000 and full[0] == 0xE9 and full[0x310000:] == b"\xff" * 0x2000
    with tempfile.TemporaryDirectory() as temp:
        t = Path(temp)
        (t / "FoloToy-AI-Passport.bin").write_bytes(app)
        (t / "payload").write_bytes(payload)
        (t / "signature").write_bytes(signature)
        trust = Path(__file__).resolve().parents[1] / "services/relay/internal/firmware/trust.pem"
        subprocess.run(["openssl", "dgst", "-sha256", "-verify", str(trust), "-signature", str(t / "signature"), str(t / "payload")], check=True)
        verify_protected_layout(full, t)
    expected = "".join(hashlib.sha256((folder / n).read_bytes()).hexdigest() + "  " + n + "\n" for n in FILES)
    assert (folder / "SHA256SUMS").read_text() == expected, "SHA256SUMS mismatch"
    return metadata

if __name__ == "__main__":
    m = verify(Path(sys.argv[1]))
    print("Verified release", m["version"], "sequence", m["sequence"])
