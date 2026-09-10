English | [简体中文](deployment.zh_CN.md)

# Deployment boundaries

P0/P1 produces a foundation image only. Do not expose it as a working intercom
service. Production deployment is deferred. The user manages the domain/site in
1Panel after the proxy target has been verified.

## Isolated deployment recipe for a later milestone

Use `/opt/pocketlink`, the explicit `deploy/compose.yaml` file and project name
`pocketlink-prod`. Never invoke Compose from an ambiguous directory. Inspect
existing containers, listeners, resources and proxy network mode before changes.

Set `POCKETLINK_IMAGE` to a published version **digest**, not a floating edge tag.
An example configuration is in `deploy/env.example` (the example is not a release).

```bash
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml config --quiet
# Execute only during an authorized deployment milestone:
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml pull
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml up -d --wait
```

The container is nonroot, read-only except its own data volume and small tmpfs,
with capabilities dropped, no new privileges, bounded logs, memory/CPU/PIDs and a
readiness healthcheck. The default application port is published only on host
loopback `127.0.0.1:18080`; verify availability first. No UDP port is published
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
