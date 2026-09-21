[简体中文](CHANGELOG.md) | English

# Changelog

## Unreleased

- Add administrator-issued registration invites, isolated device ownership and room invitation/leave/removal. Migrate existing data to the administrator and redesign the responsive console with role-specific navigation, senders and invite management.

- Fix provisioning access rejection on dual-stack HTTP sockets by recognizing IPv4-mapped addresses and explicit default port 80, while retaining hotspot-interface, Host, Origin and token checks.

- Fix missing Chinese glyphs on the provisioning screen with complete UI text and common GB2312 Han coverage, checked by host tests.

- Add signed OTA packages, A/B device updates with physical confirmation and boot rollback, console release management, SQLite migration 004 and manual GitHub signing. Device acceptance remains pending.

- Add independent PocketLink development firmware with QR hotspot provisioning, HTTPS pairing, a durable single-message inbox and receipts; add configuration/inbox host tests and dual-firmware CI. Physical-device acceptance is pending.

- Add room permissions, idempotent text sending, offline queues, per-device received/read receipts and WSS transport with console history.

- Default documentation to Chinese with English switching; expand the README with deployment, usage, upgrades, backup/recovery and troubleshooting.

- Add the PocketLink monorepo foundation and pinned AI Passport board-check demo.
- Add validated relay configuration, atomic SQLite migrations, liveness/readiness,
  capability reporting and graceful shutdown.
- Add protocol v1 parsers, shared vectors and host tests.
- Add isolated container configuration and gated image/firmware CI artifacts.
- Add optional administrator login, single-use device pairing, persistent revocable credentials
  and an embedded device management console. Voice intercom remains pending.
