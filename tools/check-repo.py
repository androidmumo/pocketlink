#!/usr/bin/env python3
"""Small, dependency-free checks shared by CI and local development."""
import hashlib
import json
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
SKIP = {".git", "managed_components", "build", "dist", "node_modules", "__pycache__", ".local"}
errors = []

def files(path):
    for child in path.iterdir():
        if child.name in SKIP:
            continue
        if child.is_dir():
            yield from files(child)
        elif child.is_file():
            yield child

for path in files(ROOT):
    relative = path.relative_to(ROOT)
    if path.suffix == ".md":
        text = path.read_text()
        zh = path.name.endswith(".zh_CN.md")
        pair = path.with_name(path.name.replace(".zh_CN.md", ".md") if zh else path.stem + ".zh_CN.md")
        if not pair.exists() or f"]({pair.name})" not in text:
            errors.append(f"{relative}: missing paired document/language link")
        if not zh and re.search(r"[\u4e00-\u9fff]", text.replace("简体中文", "")):
            errors.append(f"{relative}: Chinese prose in English document")
        for target in re.findall(r"\[[^\]]*\]\(([^)]+)\)", text):
            if "://" not in target and not target.startswith("#"):
                destination = path.parent / target.split("#")[0]
                if not destination.exists():
                    errors.append(f"{relative}: broken link {target}")
    if path.suffix in {".pem", ".key", ".db"} or path.name == ".env":
        errors.append(f"{relative}: private/generated file in source tree")
    if path.suffix in {".go", ".py", ".md", ".yaml", ".yml", ".json", ".sh"}:
        if re.search(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----", path.read_text()):
            errors.append(f"{relative}: private key material")

for path in (ROOT / ".github/workflows").glob("*.yml"):
    for action in re.findall(r"uses:\s*([^\s]+)", path.read_text()):
        if not re.fullmatch(r"[^@]+@[0-9a-f]{40}", action):
            errors.append(f"{path.name}: unpinned action {action}")

manifest = json.loads((ROOT / "firmware/boards/ai_passport/upstream.json").read_text())
for item in manifest["files"]:
    if not item["adapted"] and hashlib.sha256((ROOT/item["path"]).read_bytes()).hexdigest() != item["upstream_sha256"]:
        errors.append(f"{item['path']}: unreviewed change to pinned upstream baseline")

if errors:
    print("\n".join(errors), file=sys.stderr)
    sys.exit(1)
print("Repository docs, imports, action pins and private-file checks: PASS")
