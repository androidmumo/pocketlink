[简体中文](README.md) | English

# PocketLink

Turn an AI Passport into a pocket message receiver and intercom. Scan to provision Wi-Fi, receive text, talk in rooms and upgrade online. The web console manages devices and invitations, sends messages and joins voice rooms.

## Start here

| What you need | Open |
| --- | --- |
| Latest firmware | **[Latest release](https://github.com/androidmumo/pocketlink/releases/latest)** |
| Flash, connect and use a device | **[Simple device guide](docs/guides/device.en.md)** |
| Deploy your own Docker server | **[Simple server guide](docs/guides/docker.en.md)** |
| Protocols, source, backups and development | [Technical reference](docs/technical.en.md) |

Already have a PocketLink server address and account? Start with the device guide; you do not need your own server.

## Which firmware file?

Download from **Assets** on the Release page:

- **`pocketlink-full.bin`**: first installation through the [official browser flasher](https://ai-passport.folotoy.cn/tools/web-flasher/).
- **`pocketlink-ota.bin` + `manifest.json`**: administrators upload these under Firmware management for online device upgrades.
- `SHA256SUMS`: download integrity checks.

GitHub's automatic `Source code` archives are not firmware. Do not flash the OTA file as a full image.
[Download the full installation image](https://github.com/androidmumo/pocketlink/releases/latest/download/pocketlink-full.bin). The repository also keeps the [0.5.0 snapshot](firmware/releases/0.5.0/README.en.md); Releases is the source for newer versions.

## Available now

- Invite registration, administrator user management, and separate user devices and rooms.
- Offline text delivery, sender/time display and paged reading.
- Half-duplex room intercom for browsers and devices.
- Battery, Beijing-time clock and signed OTA upgrades.

Intercom is a development build. Builds and synthetic audio tests passed; physical audio quality, latency and mobile compatibility await acceptance.

[Changes](docs/CHANGELOG.en.md) · [Architecture](docs/architecture/overview.en.md) · [Third-party notices](docs/THIRD_PARTY.en.md)

No overall project license has been selected; imported source and dependencies retain their own licenses.
