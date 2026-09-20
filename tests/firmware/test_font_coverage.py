#!/usr/bin/env python3
"""Reject missing UI glyphs and missing common Chinese message characters."""
import importlib.util
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('font_generator', ROOT / 'tools/generate-font.py')
generator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(generator)

class FontCoverage(unittest.TestCase):
    def test_displayed_characters_have_glyphs(self):
        generated = generator.FONT.read_text()
        glyphs = {int(code, 16) for code in re.findall(r'/\* U\+([0-9a-fA-F]+)', generated)}
        missing = sorted(ord(c) for c in generator.required_characters() if ord(c) not in glyphs)
        self.assertFalse(missing, 'Missing glyphs: ' + ', '.join(f'U+{c:04X}' for c in missing))
        self.assertGreaterEqual(len(glyphs), 6763 + 95)

if __name__ == '__main__':
    unittest.main()
