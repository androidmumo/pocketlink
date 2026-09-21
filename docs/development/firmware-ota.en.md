[简体中文](firmware-ota.md) | English

# Signed OTA firmware updates

Implemented in source; the user confirmed the 0.4.2 online upgrade and its display/input feedback. Power-loss rollback and peak memory remain unverified.
Packages update the PocketLink application only, not other games, bootloaders or partition tables.
The server needs migration 004; provisioning and text reception still work with the older server.

## Current OTA release 0.4.3 (2026-09-21)

Published `0.4.3`, sequence `3`, size `2,211,920` bytes, SHA256 `3361ca51a0d645d99008e860e653ae1a010b6604f8894cad7ceddb4939c1afbe`, from source commit `a8afe48`. Background polling, battery reads, reconnects and update checks no longer trigger loading or claim the interaction lock. Manual slow actions show a single-line wait message with a trailing spinner. Immediate page turns do not show loading; queued keys cannot act on a changed message/view.

The full local host gate, both firmware builds and protected-layout checks passed; the server verified and published the package. The user confirmed successful 0.4.3 installation, no idle loading indicator, and correct single-line manual feedback with a trailing spinner. The consistent backup `backups/before-ota-0.4.3-20260921T122133Z.db` passed integrity checks. No service restart or change to other applications was needed.


## Previous OTA release 0.4.2 (2026-09-21)

Published `0.4.2`, sequence `2`, size `2,211,520` bytes, SHA256 `5b058a123415b71497388440408ff6171d7bb8fc0e9da17080add4cf099b7a6d`, from source commit `b4ab64a`. It adds CW2017 battery percentage, independent text loading animation and input locking during work, retaining 0.4.1 cancellation and paging. Local host checks, both firmware builds and protected-layout checks passed. The user confirmed OTA completion, battery display and loading/input-lock behavior on the device. Gauge accuracy, prolonged charge/discharge and power-loss rollback remain unverified.

Before server verification/publication, the consistent online backup `backups/before-ota-0.4.2-20260921T114229Z.db` passed integrity checks. Version 0.4.1 remains stored but unpublished; installation still requires a physical long-Down confirmation.


## Previous OTA release 0.4.1 (2026-09-21)

Published `0.4.1`, sequence `1`, application size `2,189,520` bytes, SHA256 `3fbececb6d9f0aacc84ea03c95eae1e7b9e65cc65bb59e58ca9f32b5f15dfb11`, from source commit `7ed5e63`. The complete local host gate, both firmware builds and protected-layout checks passed. Signing used the private key outside the repository; the server verified the package before publication.

This release adds long-OK provisioning cancellation with saved-network recovery, fixed message pages and reading-position preservation. Bluetooth provisioning is not included. Long-press Up to check, then long-press Down after confirming the version. Publication does not prove successful installation; physical OTA and behavior acceptance await user confirmation.

The consistent online SQLite backup is `backups/before-ota-0.4.1-20260921T111956Z.db`; integrity checks passed. No service container was restarted and other services were unchanged.


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
