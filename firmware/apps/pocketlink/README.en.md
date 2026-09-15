[简体中文](README.md) | English

# PocketLink device firmware (P3 development build)

For the AI Passport ESP32-C3. Implements QR hotspot provisioning, HTTPS device pairing,
text reception and read receipts. Voice, a WSS client, raw TCP/UDP clients and OTA are
not implemented. Builds and host tests do not establish real-device acceptance.
See [device provisioning and reception](../../../docs/development/device-provisioning.en.md)
for usage, protocol, storage and flashing constraints.

Built separately from `board-check`, using the pinned shared BSP without upstream
working-tree changes. Application and merged image retain the `FoloToy-AI-Passport`
name for the existing layout verifier. Choose the `pocketlink-<commit>` CI artifact;
the `board-check` artifact is not a substitute.
