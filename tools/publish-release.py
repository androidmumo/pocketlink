#!/usr/bin/env python3
"""Publish four verified firmware assets; existing releases are never overwritten."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import urllib.error
import urllib.parse
import urllib.request

spec = importlib.util.spec_from_file_location("check_release", Path(__file__).with_name("check-release.py"))
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)


def publish(folder, source, expected_version):
    metadata = checker.verify(folder)
    assert metadata["version"] == expected_version, "Tag/input version does not match signed metadata"
    assert re.fullmatch(r"[0-9a-f]{40}", source)
    token, repo = os.environ["GH_TOKEN"], os.environ["GITHUB_REPOSITORY"]
    assert re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repo)
    api = "https://api.github.com/repos/" + repo
    def call(url, method="GET", data=None, content_type="application/json"):
        request = urllib.request.Request(url, data=data, method=method, headers={"Authorization": "Bearer " + token, "Accept": "application/vnd.github+json", "Content-Type": content_type})
        with urllib.request.urlopen(request, timeout=120) as response:
            return json.load(response)
    tag = "firmware/pocketlink/v" + metadata["version"]
    try:
        call(api + "/releases/tags/" + urllib.parse.quote(tag, safe=""))
    except urllib.error.HTTPError as error:
        if error.code != 404:
            raise
    else:
        raise RuntimeError("Release exists; published assets must not be overwritten")
    body = ("## 下载 / Downloads\n\n"
            "- `pocketlink-full.bin`：首次 USB 刷机，地址 `0x0`；不要清除设备数据。 / Initial USB image at `0x0`; do not erase device data.\n"
            "- `pocketlink-ota.bin` + `manifest.json`：一起上传到固件管理。 / Upload together for OTA.\n"
            "- `SHA256SUMS`：下载校验 / Integrity checks.\n\n"
            "[中文教程](https://github.com/" + repo + "/blob/main/docs/guides/device.md) · "
            "[English guide](https://github.com/" + repo + "/blob/main/docs/guides/device.en.md)\n\n"
            "开发版：真机音质、延迟和手机兼容性待验收。 / Development build: physical audio quality, latency and mobile compatibility await acceptance.\n\n"
            "Firmware source: `" + source + "` · Sequence: " + str(metadata["sequence"]) + "\n")
    release = call(api + "/releases", "POST", json.dumps({"tag_name": tag, "target_commitish": source, "name": "PocketLink " + metadata["version"] + " · 开发版 / Development", "body": body, "draft": True, "prerelease": False}).encode())
    upload = release["upload_url"].split("{")[0]
    assert upload.startswith("https://uploads.github.com/repos/" + repo + "/releases/")
    for name in checker.FILES + ["SHA256SUMS"]:
        data = (folder / name).read_bytes()
        asset = call(upload + "?name=" + urllib.parse.quote(name), "POST", data, "application/octet-stream")
        assert asset["state"] == "uploaded" and asset["size"] == len(data)
        if asset.get("digest"):
            assert asset["digest"] == "sha256:" + hashlib.sha256(data).hexdigest()
    result = call(api + "/releases/" + str(release["id"]), "PATCH", json.dumps({"draft": False, "make_latest": "true"}).encode())
    assert not result["draft"]
    print("Published:", result["html_url"])


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("folder", type=Path)
    parser.add_argument("--source", required=True)
    parser.add_argument("--version", required=True)
    args = parser.parse_args()
    publish(args.folder, args.source, args.version)
