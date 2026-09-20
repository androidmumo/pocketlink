#!/usr/bin/env python3
"""Generate the committed font from a checksum-verified Source Han Sans release."""
import argparse
import hashlib
from pathlib import Path
import subprocess
import tempfile
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
FONT = ROOT / 'firmware/components/pocketlink_font/pocketlink_font_14.c'
URL = 'https://raw.githubusercontent.com/adobe-fonts/source-han-sans/2.004R/OTF/SimplifiedChinese/SourceHanSansSC-Regular.otf'
SHA256 = '84bbd4ace91d327b3ad1a581c688196278a4e41308520176f419180064e4af2b'

def required_characters():
    characters = set(chr(code) for code in range(32, 127))
    # GB2312 levels 1 and 2 contain 6,763 commonly used Chinese characters.
    for first in range(0xB0, 0xF8):
        for second in range(0xA1, 0xFF):
            try:
                characters.add(bytes((first, second)).decode('gb2312'))
            except UnicodeDecodeError:
                pass
    characters.update('，。！？：；“”‘’（）【】《》、…—·')
    source = (ROOT / 'firmware/apps/pocketlink/main/main.c').read_text()
    characters.update(c for c in source if ord(c) >= 128)
    return characters

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--font', type=Path, help='Use an already downloaded, checksum-verified OTF')
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='pocketlink-font-') as directory:
        source = args.font or Path(directory) / 'source.otf'
        if args.font is None:
            with urllib.request.urlopen(URL, timeout=60) as response:
                source.write_bytes(response.read())
        if hashlib.sha256(source.read_bytes()).hexdigest() != SHA256:
            raise SystemExit('Source font SHA256 mismatch')
        generated = Path(directory) / FONT.name
        subprocess.run(['npx', '--yes', '--package=lv_font_conv@1.5.3', 'lv_font_conv',
                        '--font', str(source), '--symbols', ''.join(sorted(required_characters())),
                        '--size', '14', '--bpp', '4', '--no-compress', '--no-prefilter',
                        '--format', 'lvgl', '--lv-include', 'lvgl.h', '-o', str(generated)], check=True)
        text = generated.read_text()
        # Keep generated output reproducible across temporary paths and workstations.
        text = text.replace(str(source), 'SourceHanSansSC-Regular.otf').replace(str(generated), FONT.name)
        FONT.write_text(text)
        print('Generated', FONT.relative_to(ROOT))

if __name__ == '__main__':
    main()
