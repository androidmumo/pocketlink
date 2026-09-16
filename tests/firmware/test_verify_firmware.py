#!/usr/bin/env python3
"""Host tests for the protected firmware-layout parser."""

from __future__ import annotations

import hashlib
import importlib.util
import struct
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location(
    "verify_firmware", ROOT / "tools" / "verify_firmware.py"
)
assert SPEC and SPEC.loader
VERIFY = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = VERIFY
SPEC.loader.exec_module(VERIFY)


def sample_table(ota: bool = False) -> bytes:
    entries = (
        (1, 2, 0x9000, 0x6000, "nvs"),
        (1, 1, 0xF000, 0x1000, "phy_init"),
        (0, 0, 0x10000, 0x300000, "factory"),
        (1, 2, 0x356000, 0x4000, "cardid"),
    )
    if ota:
        entries = entries[:2] + (
            (0, 0x10, 0x10000, 0x300000, "ota_0"),
            (1, 0, 0x310000, 0x2000, "otadata"),
            (1, 2, 0x356000, 0x4000, "cardid"),
            (1, 2, 0x35a000, 0x10000, "pocketcfg"),
            (0, 0x11, 0x370000, 0x300000, "ota_1"),
        )
    raw = bytearray(b"\xff" * VERIFY.PARTITION_TABLE_SIZE)
    for index, (kind, subtype, offset, size, label) in enumerate(entries):
        VERIFY.ENTRY.pack_into(
            raw,
            index * VERIFY.ENTRY.size,
            0x50AA,
            kind,
            subtype,
            offset,
            size,
            label.encode().ljust(16, b"\0"),
            0,
        )
    marker = len(entries) * VERIFY.ENTRY.size
    struct.pack_into("<H", raw, marker, 0xEBEB)
    raw[marker + 16 : marker + 32] = hashlib.md5(raw[:marker]).digest()
    return bytes(raw)


class PartitionParserTest(unittest.TestCase):
    def test_parses_protected_layout_and_md5(self) -> None:
        partitions, found_md5 = VERIFY.parse_partition_table(sample_table())
        self.assertTrue(found_md5)
        self.assertEqual(partitions[-1].label, "cardid")
        self.assertEqual(partitions[-1].offset, VERIFY.CARDID_OFFSET)

    def test_rejects_bad_md5(self) -> None:
        raw = bytearray(sample_table())
        raw[28] ^= 1
        with self.assertRaisesRegex(ValueError, "MD5"):
            VERIFY.parse_partition_table(bytes(raw))


class ProtectedLayoutTest(unittest.TestCase):
    def test_layout_verification_accepts_current_partition_table(self) -> None:
        merged = bytearray(b"\xff" * (0x10000 + 1))
        merged[
            VERIFY.PARTITION_TABLE_OFFSET :
            VERIFY.PARTITION_TABLE_OFFSET + VERIFY.PARTITION_TABLE_SIZE
        ] = sample_table()
        merged[0x10000] = 0xE9

        with tempfile.TemporaryDirectory() as directory:
            build_dir = Path(directory)
            (build_dir / "FoloToy-AI-Passport.bin").write_bytes(b"\xe9")
            VERIFY.verify_protected_layout(bytes(merged), build_dir)

    def test_ota_layout_preserves_identity_and_configuration(self) -> None:
        merged = bytearray(b"\xff" * (0x10000 + 1))
        merged[0x8000:0x8000 + VERIFY.PARTITION_TABLE_SIZE] = sample_table(True)
        merged[0x10000] = 0xe9
        with tempfile.TemporaryDirectory() as directory:
            build = Path(directory)
            (build / "FoloToy-AI-Passport.bin").write_bytes(b"\xe9")
            VERIFY.verify_protected_layout(bytes(merged), build)
            # A protection-region payload must always be rejected.
            merged.extend(b"\xff" * (VERIFY.CARDID_OFFSET + 1 - len(merged)))
            merged[VERIFY.CARDID_OFFSET] = 0
            with self.assertRaisesRegex(ValueError, "cardid payload"):
                VERIFY.verify_protected_layout(bytes(merged), build)

    def test_rejects_displaced_ota_slots_and_configuration(self) -> None:
        for index in (2, 3, 5, 6):
            with self.subTest(partition=index), tempfile.TemporaryDirectory() as directory:
                table = bytearray(sample_table(True))
                offset = index * VERIFY.ENTRY.size + 4
                struct.pack_into("<I", table, offset, struct.unpack_from("<I", table, offset)[0] + 0x1000)
                marker = 7 * VERIFY.ENTRY.size
                table[marker + 16:marker + 32] = hashlib.md5(table[:marker]).digest()
                merged = bytearray(b"\xff" * (0x10000 + 1))
                merged[0x8000:0x8000 + len(table)] = table
                merged[0x10000] = 0xe9
                build = Path(directory)
                (build / "FoloToy-AI-Passport.bin").write_bytes(b"\xe9")
                with self.assertRaisesRegex(ValueError, "must remain"):
                    VERIFY.verify_protected_layout(bytes(merged), build)


if __name__ == "__main__":
    unittest.main()
