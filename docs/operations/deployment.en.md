[简体中文](deployment.md) | English

# Deployment boundaries

The user authorized deployment of invite registration and shared rooms, with isolated accounts,
device management, room permissions, text sending, offline delivery and receipts.
Device reception firmware awaits physical acceptance; voice intercom is not implemented.
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
- Application commit: `b4ab64ae4e8e4f6029b19011d09d877c3f462ace`.
- Image: `ghcr.io/androidmumo/pocketlink-relay@sha256:5a2e62e4a837d323ae2946f52cf48ba3d393acc36801a0ea8e46ca23dc2c0270`.
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
