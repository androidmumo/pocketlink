[简体中文](CHANGELOG.md) | English

# Changelog

## Unreleased

- Remove duplicate provisioning copy from the empty inbox; add a custom room picker, encrypted invitation recovery with view/copy, and upload/latest publication dates. Migration 006 requires backing up the recovery key with the data directory.

- Keep background polling, battery reads, reconnects and update checks out of interactive loading/locking. Show a single-line wait message with a trailing spinner for manual work, and reject stale keys after message/view changes.

- Show fuel-gauge battery percentage and low-charge color, animate a lightweight loading indicator independently of the worker, reject overlapping operations, and bound web message history with internal scrolling.

- Allow long-OK cancellation of device provisioning with saved-network recovery; replace message scrolling with fixed pages, clamp short messages and preserve the page across receipt updates.

- Fix unnecessary login-page scrolling and sidebar rubber-banding with a viewport-bound layout and independent content scrolling; keep forms accessible on small screens.

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
