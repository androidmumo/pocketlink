English | [简体中文](deployment.zh_CN.md)

# Deployment boundaries

The user authorized a P2a trial deployment with login and device management only.
The complete intercom service remains a later milestone. The user manages the
domain and HTTPS reverse proxy in 1Panel.

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
TLS and future WebSocket upgrade handling belong to the dedicated reverse-proxy
site. P2 must define trusted proxies before using forwarded headers for security.

## Backup and rollback

The named `pocketlink-prod_data` volume holds SQLite and WAL files. Do not copy
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

## Current trial environment (2026-09-11)

- URL: `https://pocketlink.mcloc.cn`, proxied by 1Panel to `127.0.0.1:3002`.
- Application commit: `a204762f96e519fd1315a180d7af0287e8852ac8`.
- Image: `ghcr.io/androidmumo/pocketlink-relay@sha256:d9c02a8e2ba751b290ae2cee369eddb2a949c5812bde342281f72c1ad909997f`.
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
