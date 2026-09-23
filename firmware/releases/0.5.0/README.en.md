[简体中文](README.md) | English

# PocketLink 0.5.0 firmware

Prefer [GitHub Releases](https://github.com/androidmumo/pocketlink/releases). These matching files are also kept in the repository for direct access. They contain no private keys, device identity or user settings.

- [pocketlink-full.bin](pocketlink-full.bin): initial USB installation at `0x0`; never erase the whole chip.
- [pocketlink-ota.bin](pocketlink-ota.bin) and [manifest.json](manifest.json): upload together to OTA management.
- [SHA256SUMS](SHA256SUMS): SHA256 checksums.

Source commit `78a53484c3ac57fe986276989bae8332e695950f`, sequence 5. OTA matches the package published on 2026-09-22 without re-signing; the full image embeds the identical application. Builds and host tests passed; physical 0.5.0 acceptance is pending.

[Flashing guide](../../../docs/guides/device.en.md)
