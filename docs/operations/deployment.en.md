[简体中文](deployment.md) | English

# Deployment boundaries

The user authorized deployment of invite registration and shared rooms, with isolated accounts,
device management, room permissions, text sending, offline delivery and receipts.
The source includes development half-duplex intercom; physical audio quality and latency await acceptance. The deployment records below identify the production version.
The user manages the domain and HTTPS reverse proxy in 1Panel.

## Isolated deployment configuration

Use `/opt/pocketlink`, the explicit `deploy/compose.yaml` file and project name
`pocketlink-prod`. Never invoke Compose from an ambiguous directory. Inspect
existing containers, listeners, resources and proxy network mode before changes.

Set `POCKETLINK_IMAGE` to a published version **digest**, not a floating edge tag.
An example configuration is in `deploy/env.example` (the example is not a release).

```bash
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml -f /opt/pocketlink/compose.auth.yaml config --quiet
# Execute only during an authorized deployment milestone:
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml -f /opt/pocketlink/compose.auth.yaml pull
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml -f /opt/pocketlink/compose.auth.yaml up -d --wait
```

The container is nonroot, read-only except its own data volume and small tmpfs,
with capabilities dropped, no new privileges, bounded logs, memory/CPU/PIDs and a
readiness healthcheck. The default application port is published only on host
loopback `127.0.0.1:3002`; verify availability first. No UDP port is published
until the authenticated media service exists. Do not change other applications,
Docker daemon settings or shared 1Panel networks. Do not run global prune or
`down -v`. Docker isolation does not isolate kernel, total disk or host failures.

If OpenResty uses host networking, its upstream can be the loopback application
port. A containerized proxy on a bridge requires a separately reviewed network
attachment instead. The application port stays off public security-group rules.
TLS and WebSocket upgrade handling belong to the dedicated reverse-proxy
site. P2 must define trusted proxies before using forwarded headers for security.

## Backup and rollback

The template uses the named `pocketlink-prod_data` volume; this deployment uses
the `data/` bind directory recorded below for SQLite and WAL files. Do not copy
only the live main DB file. Before a migration, either use a tested SQLite online
backup command or stop only this service and snapshot its entire data volume.
Keep backups outside the live volume; protect them as private message data.
Backup/restore automation must be implemented and exercised before production.

Keep the previous image digest and database schema version. A compatible image
can be redeployed; a schema downgrade requires a documented restore procedure.
Restoring a snapshot loses writes after that snapshot, so do not silently restore
while users are sending data. The startup migration guard rejects newer schemas.

GHCR can require a registry login for private packages. Do not put registry or
server credentials in the repository or logs. Publishing via GITHUB_TOKEN and
production pulling are distinct permissions. CI performs no production updates.

## Current trial environment (upgraded and verified 2026-09-21)

- URL: `https://pocketlink.mcloc.cn`, proxied by 1Panel to `127.0.0.1:3002`.
- Application commit: `1243e27b688d8e2a3b991d1b97d31269e300080a`.
- Image: `ghcr.io/androidmumo/pocketlink-relay@sha256:ff711150690689b0bab273894ce2b8bece6e3fa4e4b9bb12fcabb226f1f93908`.
- Active configuration: `/opt/pocketlink/compose.yaml`, `compose.auth.yaml` and `.env`.
- Persistent files live under `/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/`:
  `data/` holds SQLite, `secrets/admin_password` the administrator secret and `backups/` consistent backups.

The server Compose replaces `/data` with a bind mount of the directory above instead
of the template's named volume. Do not overwrite this mapping with the repository template
on upgrades. The private parent is mode 0700; database ownership is UID/GID 65532 and
the password file is mode 0400. Keep the site as a full reverse proxy; never expose
persistent files through a static-file website.

HTTPS login/logout, restart session invalidation and health checks passed. A stopped-service
backup was restored into a temporary directory and passed SQLite integrity checks.
Existing bing, chat and OpenResty containers were not restarted. The initial backup is
`backups/initial-20260911T011224Z.tar.gz`; no scheduled backup has been configured.
Administrators can read the password through 1Panel file management. It is never committed
or printed in deployment logs.

The pre-text-upgrade backup is `backups/before-text-20260911T073934Z.tar.gz`.
Checks on 2026-09-14 passed: new container health, loopback port 3002 and data mount,
live database integrity, pre-upgrade backup database restore, HTTPS text capabilities,
login, room/device list reads and logout. Existing bing, chat and OpenResty start
times match the pre-upgrade baseline. WebSocket delivery, reconnect replay and receipts
passed host and CI tests; real devices behind the production proxy and physical device reception remain unverified.

Signed OTA management was deployed on 2026-09-20. The pre-upgrade backup `backups/before-ota-20260920T021947Z.tar.gz` contains data and deployment configuration; its extracted database passed integrity checks. Migration 004, HTTPS login/firmware listing/logout and health checks passed; the firmware channel is empty. Other containers retained their start times and restart counts. Returning to the old text release requires restoring its matching database, not just switching images.

On 2026-09-21, invite registration, account isolation, shared rooms and the redesigned console were deployed. The pre-upgrade backup `backups/before-accounts-20260921T060225Z.tar.gz` contains data and deployment configuration; its extracted SQLite database passed integrity checks. Migration 005, legacy ownership, existing row counts, HTTPS administrator login/identity/device/room/firmware/invitation reads and logout passed. Other containers retained their start times and restart counts. Isolated local browser tests covered registration, two-account room joining/leaving, text and receipts, and desktop/mobile layouts; no production test accounts were created. Rollback requires the pre-upgrade database and may discard later writes; never restore it silently.

A subsequent deployment on 2026-09-21 fixes console scrolling: the document and navigation remain stationary while content scrolls independently. Browser checks passed at five desktop, mobile and landscape sizes. The backup is `backups/before-scroll-20260921T064947Z.tar.gz`; the schema is unchanged. Production CSS matches the verified file, HTTPS login/API/logout and container health checks passed, and other containers were not restarted.

On 2026-09-21 the bounded message-history region was deployed: at most 420px and half the viewport, with internal scrolling. Backup `backups/before-history-20260921T115006Z.tar.gz` contains data/configuration and passed extracted-database integrity checks. Production CSS matches the browser-verified file; HTTPS login/API/logout passed and other containers were not restarted. OTA 0.4.2 (sequence 2) remains published; the user confirmed online installation, battery display and loading/input locking.

## Invitation recovery and release dates

A recovery key is automatically stored beside the database with the `.invitation-key` suffix (default `data/relay.db.invitation-key`, mode 0600). Back up and restore the **entire data directory**, including the matching key, not SQLite alone. Startup refuses to replace a missing key when encrypted invitations exist; restore the matching backup instead. The key is independent of administrator password changes. Access to both the database and key exposes active codes; protect both and their backups as credentials.


The custom room picker supports arrow keys, Enter and Escape. New active invitations can be viewed and copied again; lost legacy codes must be revoked and replaced. The release library shows upload time and latest publication time in the browser time zone. Historical missing publication dates display “Not recorded”, never the upload time. Repeating publication of the active release preserves its timestamp; withdrawal retains it, and republication updates it.

On 2026-09-21, deployed the custom room picker, encrypted invitation recovery and publication dates. The pre-upgrade backup `backups/before-console-refinements-20260921T125516Z.tar.gz` passed extraction and SQLite integrity checks. Migration 006, retained record counts, key-file permissions, HTTPS login and API checks passed; other containers retained their start times and restart counts. A single verification invitation was recovered and then revoked; no account or message was created. All three deployed UI assets match the locally browser-tested files by SHA256. OTA 0.4.4 is published; the user confirmed 0.4.4 installation and correct display. Future backups must preserve the matching `.invitation-key` file.

## 2026-09-22 intercom and room actions deployment

Deployed source `78a5348`. CI [35703429952](https://github.com/androidmumo/pocketlink/actions/runs/35703429952) passed host checks, firmware builds, amd64/arm64 container smoke tests and image publication. Backup `backups/before-intercom-20260922T082059Z.tar.gz` includes all data and configuration; extracted SQLite integrity checks passed. No database migration was introduced. Container health, retained data counts, key permissions, six UI asset hashes, HTTPS APIs and the WSS subprotocol handshake through the production proxy passed. The verification connection sent no production audio or message; its test invitation was revoked. Other containers retained their start times and restart counts.

The listener remains `127.0.0.1:3002` with the same data path, no new UDP port and no shared-proxy restart. OTA 0.5.0 is published and requires manual confirmation on the device. The user chose publication first and deferred physical acceptance. Registration invitations remain on the administrator page; room invitations are in the selected room's Invite friends action panel.

## 2026-09-23 administrator user management deployment

Deployed source `b07f03a`. CI [35826339327](https://github.com/androidmumo/pocketlink/actions/runs/35826339327) passed host, firmware and both Docker architecture checks. Backup `backups/before-user-management-20260923T063232Z.tar.gz` contains complete data, secrets and deployment configuration; extracted SQLite integrity passed. Migration 007, retained user/business counts, the file-based administrator password boundary, invitation key permissions and container health were verified. User listing, HTTPS login/logout, WSS handshake and six deployed asset hashes passed. No production account was disabled or reset, and no test user was created. Other containers retained their start times and restart counts.

Administrators can search users, inspect registration dates and active device/owned-room counts, disable/enable, force logout and reset ordinary passwords. Permanent deletion is not provided; the built-in administrator remains protected. See [account management](../development/accounts.en.md). Rollback requires the matching pre-migration database and configuration, not only the old image. Firmware and OTA remain 0.5.0; device upgrades are unnecessary.

## User management style fix (2026-09-23)

Deployed `1243e27`: align sidebar text and show compact blue administrator, green active and red disabled badges. Desktop/mobile browser checks, user-management regression, the complete static gate and CI [35854733820](https://github.com/androidmumo/pocketlink/actions/runs/35854733820) passed. Backup `backups/before-user-badges-20260923T113810Z.tar.gz` was checked for SQLite integrity before deployment; the database remains at migration 007. Production login, user listing, WSS handshake and asset hashes passed; other containers were not restarted. Firmware is unchanged.
