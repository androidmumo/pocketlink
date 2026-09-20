[简体中文](firmware-ota.md) | English

# Signed OTA firmware updates

Implemented in source; device upgrades, power-loss rollback and peak memory remain unverified.
Packages update the PocketLink application only, not other games, bootloaders or partition tables.
The server needs migration 004; provisioning and text reception still work with the older server.

## Initial migration and layout

The existing ESP32-C3 with 8 MB Flash can hold two 3 MiB application slots without additional hardware.
Migration from factory firmware requires a first wired installation of the new bootloader, partition table,
application and blank otadata. Subsequent application updates can use Wi-Fi.

| Partition | Address | Size |
| --- | --- | --- |
| ota_0 | 0x10000 | 0x300000 |
| otadata | 0x310000 | 0x2000 |
| cardid (protected) | 0x356000 | 0x4000 |
| pocketcfg (configuration and update state) | 0x35a000 | 0x10000 |
| ota_1 | 0x370000 | 0x300000 |

Original nvs and phy_init remain. Before migration, privately back up cardid and pocketcfg and inspect
actual partitions. Build and verify the merged image, then use that build's segmented flash_args.
**Never erase the entire Flash or include identity backups in source or release packages.** Initial wired installation, boot and identity preservation passed on one device on 2026-09-20; OTA update/rollback still needs physical-device acceptance. Returning to diagnostic firmware also requires wired migration.

## Building and signing

Activate ESP-IDF 5.5.3 and run `./tools/validate.sh --firmware` at the repository root.
`dist/firmware/pocketlink/pocketlink-ota.bin` is application-only.
`FoloToy-AI-Passport-full.bin` is a merged wired-install image and cannot be uploaded for OTA.

The two source `trust.pem` files embed an identical public key in server and device.
The matching private key is outside this repository at `~/.config/pocketlink/firmware-signing-key.pem`
on the development Mac. Keep a private backup; never publish it or put it on the relay server.
Key replacement requires trusted wired migration; online key rotation is not implemented.

Local signing (choose the actual version and increasing sequence):

```bash
source tools/env.sh
cd services/relay
go run ./cmd/fwtool \
  -key "$HOME/.config/pocketlink/firmware-signing-key.pem" \
  -image ../../dist/firmware/pocketlink/pocketlink-ota.bin \
  -version 0.4.0 -sequence 1 -out ../../dist/signed
```

This produces `manifest.json` and `pocketlink-ota.bin`. Sequence must be 1..2147483647 and greater
than every sequence previously accepted by the server and the device's confirmed sequence. Deleting
releases does not free their sequences. Version accepts ASCII letters, digits, dot, underscore and dash,
starts with a letter or digit and is at most 32 characters. Identical application binaries share a hash;
retry uploads with the original signed package rather than signing identical bytes again.

For GitHub packaging, open Settings → Secrets and variables → Actions → New repository secret.
Name it `POCKETLINK_FIRMWARE_SIGNING_KEY` and paste the complete private key into that Secret field,
never into chat or an Issue. This repository configured the Secret on 2026-09-17; the signing workflow still needs a verification run.
Open Actions → Signed firmware package → Run workflow, select main and enter version and sequence.
Download `signed-pocketlink-<run_id>` after build, checks and signing succeed. The workflow does not
publish to devices or deploy the server automatically.

## Console and device operation

1. Log in as administrator. Select the matching manifest and application under firmware upgrades and save.
2. The server validates signature, target, size, hash and sequence. Up to eight releases are stored; saving is not publishing.
3. Publish a release. There is one global channel for PocketLink on AI Passport ESP32-C3.
4. A paired online device checks after about 30 seconds when idle, then every 30 minutes. Unread messages
   defer automatic checks. Hold Up outside provisioning to check manually.
5. The device verifies the signature and displays the version. Hold Down to install or press OK to cancel.
   Text polling pauses while the prompt is shown.
6. Download writes the inactive slot, verifies SHA256 and ESP image validity, switches the boot slot and restarts.
   Maintain stable power and avoid repeated key presses. Failure retains the current application.
7. The new application confirms boot after storage, display and Wi-Fi initialization, without requiring internet
   connectivity. A crash or restart before confirmation triggers bootloader rollback. Later business failures do not.

Published means available, not installed. Fleet progress, forced remote installation, resumable downloads,
delta updates and automatic low-battery blocking are not implemented. Withdraw stops new downloads but
an existing transfer may finish. Only unpublished releases can be deleted. To restore older application behavior,
rebuild older code with a new increasing sequence; devices reject previously confirmed or lower sequences.

## Security and recovery boundaries

HTTPS validates the server certificate; device credentials authorize downloads and redirects are disabled.
ECDSA P-256 / SHA256 binds target, version, sequence, size and image hash. The relay never receives the private key.
This is application verification, without Secure Boot or hardware anti-rollback eFuse provisioning. It does not
resist physical reflashing of malicious firmware. Sequence lives in pocketcfg; do not erase it to bypass checks.
Interrupted downloads do not overwrite the running application.

Migration 004 stores manifests and images in existing SQLite, with approximately 24 MiB maximum image data.
No new Docker volume or public port is needed; the reverse proxy still uses 3002. Allow at least 4 MiB uploads
and preferably 180-second upload/download timeouts in 1Panel. Database backups include packages. Back up before
migration: older server images reject the newer schema, so server rollback must restore a matching old database.
See [deployment](../operations/deployment.en.md).

## Device acceptance checklist

- Safely migrate two devices, preserving identity, Wi-Fi, binding and unread messages.
- Update A→B→A; persist confirmed sequences and reject old sequences.
- Reject bad signatures, hashes, targets, merged images and truncated packages without changing boot selection.
- Test power loss during download, failed downloads and failure before new-application confirmation.
- Verify text, receipts and provisioning afterward; measure free heap and main-task stack headroom.
