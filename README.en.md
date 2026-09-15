[简体中文](README.md) | English

# PocketLink

One repository for connected pocket-device firmware, relay services and a web console,
with future expansion into multiplayer games and other applications.

**Current release: P2 trial with device management, room permissions, text sending, offline delivery and per-device receipts.**
A new **P3 device development build** implements QR hotspot provisioning, HTTPS pairing
and text reception, pending physical-device acceptance. Voice is not implemented.
Use `pocketlink` firmware for device features; `board-check` remains hardware diagnostics.
[Device provisioning and flashing guide](docs/development/device-provisioning.en.md)

[Using the console](#using-the-console) · [Deploying](#first-deployment-on-a-new-server) · [Maintenance](#maintenance-and-upgrades) · [Backups](#backup-and-recovery) · [Troubleshooting](#troubleshooting) · [Development](#local-development-and-repository-layout)

## Existing deployment

Open **[PocketLink](https://pocketlink.mcloc.cn)**. The following records the deployment on
2026-09-11, rechecked on 2026-09-14; server configuration files are authoritative after later changes.

| Item | Configuration |
| --- | --- |
| Entry point | 1Panel HTTPS reverse proxy → `http://127.0.0.1:3002` |
| Container / Compose project | `pocketlink-prod-relay-1` / `pocketlink-prod` |
| Active configuration | `/opt/pocketlink/compose.yaml`, `compose.auth.yaml`, `.env` |
| Persistent root | `/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/` |
| Database | `data/relay.db` under that root, plus runtime SQLite WAL files |
| Administrator password | `secrets/admin_password` under that root |
| Backups | `backups/` under that root; no scheduled backup currently exists |
| Application commit | `b7c5618c77b003e0323bfee8ec6a1774c126ab96` |
| Resource limits | 192 MB memory, 0.5 CPU, 128 processes; non-root, read-only root filesystem |

The image includes the web console; no separate frontend or MySQL deployment is needed.
SQLite persists in a dedicated directory. HTTPS login/logout, restart session invalidation,
backup recovery and database integrity were verified. Physical device pairing is unverified.

## Using the console

1. In **1Panel → Files**, open the full server path of `secrets/admin_password` above and copy the password.
   There is one administrator, with no username, registration or default password. Never commit the password or post it publicly.
2. Open the console over HTTPS and log in. Sessions last up to 12 hours; restarting the service requires another login.
3. Enter a device name and generate a pairing code. Names allow 40 characters. Codes expire after 10 minutes and work once.
4. With the pocketlink development firmware, scan the device hotspot QR and enter Wi-Fi
   settings and the pairing code on its local page. See [device provisioning](docs/development/device-provisioning.en.md).
   **Generating a code alone does not add a device**; it appears after successful pairing.
   Developers can also use the [authentication API](docs/development/authentication.en.md).
5. Refresh after successful pairing. An active entry means its credential is valid, **not that the device is online**.
6. Revoke the credential when a device is lost or needs rotation. Old tokens stop working immediately;
   revoke an active SN before pairing it again.
7. Log out when finished. If a pairing response is lost, the device secret cannot be recovered; revoke and pair again.

SN is an identifier, not a password. Pairing codes and device credentials are random secrets;
SQLite stores only their digests. Limits: 32 unexpired codes, 256 active devices, 1024 retained
serials and a shared 20 login/pairing attempts per minute. There is no web password editor,
automatic device-token expiry or multi-administrator account system yet.

### Rooms and text

Create a room, check paired devices as members, then send up to 200 characters. Messages persist before delivery;
view/refresh receipts in history for pending, received, read or withdrawn status. New members do not receive history;
removing members or archiving withdraws delivery rights. Physical reception still needs upcoming firmware.
Retry identical content after network failure; check history before resending after reload. See the [text/WSS contract](docs/development/messaging.en.md).

## First deployment on a new server

**The current server is already deployed. Do not repeat these installation steps there.**
This example reproduces the trial release. Other domains/paths require corresponding changes
to the proxy, environment and mounts.

Prepare Docker Engine, Docker Compose, Git, Python 3 and 1Panel with OpenResty. Work as the
server administrator; inspect available resources, port 3002 and existing services first.

### 1. Prepare configuration and private directories

```bash
# First installation only. Do not repeat on the existing deployment.
set -e
ss -lnt | grep ':3002 ' || true
free -m
df -h /opt
# If 3002 is occupied, stop here and identify its owner; do not stop another service.
test ! -e /opt/pocketlink
test ! -e /opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink
git clone https://github.com/androidmumo/pocketlink.git /opt/pocketlink-src
cd /opt/pocketlink-src
git checkout b7c5618c77b003e0323bfee8ec6a1774c126ab96
install -d -m 700 /opt/pocketlink
cp deploy/compose.yaml deploy/compose.auth.yaml /opt/pocketlink/
base=/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink
install -d -m 700 "$base" "$base/secrets" "$base/backups"
install -d -m 700 -o 65532 -g 65532 "$base/data"
python3 - <<'PASSWORD'
from pathlib import Path
import os, secrets
p = Path('/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/secrets/admin_password')
with p.open('x') as f:
    f.write(secrets.token_urlsafe(32) + '\n')
os.chown(p, 65532, 65532)
p.chmod(0o400)
PASSWORD
```

Write the following to `/opt/pocketlink/.env` and set its permissions to `600`.
The pinned digest comes from the [successful CI run](https://github.com/androidmumo/pocketlink/actions/runs/34574560808).
Do not treat the floating `edge` tag as a fixed release.

```dotenv
POCKETLINK_IMAGE=ghcr.io/androidmumo/pocketlink-relay@sha256:b2c473220481cafd4b028d2b4257ccaebd54c3c0cb31377e8feaa7cc7e09696c
POCKETLINK_HTTP_PORT=3002
POCKETLINK_LOG_LEVEL=info
POCKETLINK_PUBLIC_ORIGIN=https://pocketlink.mcloc.cn
POCKETLINK_ADMIN_SECRET_SOURCE=/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/secrets/admin_password
```

Edit `/opt/pocketlink/compose.yaml`: replace the service's `- data:/data` entry with the
bind mount below, then remove the unused final top-level `volumes: / data:` declaration.
Preserve the other non-root, port and resource restrictions.

```yaml
    volumes:
      - type: bind
        source: /opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/data
        target: /data
        bind:
          create_host_path: false
```

**The repository template uses a named volume; this site uses the bind mount above.
Do not overwrite the server file with the template on upgrades.** Keep the private parent
at `0700`, data owned by `65532:65532` and the password at `0400`. Do not recursively change
permissions on the whole site. The password is mounted read-only, not stored in environment
values or the image. Servers with SELinux enabled also need appropriate container labels
on dedicated directories; do not disable SELinux globally.

### 2. Start the container

Define this helper, scoped to the project and both configuration files, for subsequent operations:

```bash
# Run in Bash on the server; scoped to PocketLink.
pl() {
  docker compose --env-file /opt/pocketlink/.env -p pocketlink-prod \
    -f /opt/pocketlink/compose.yaml -f /opt/pocketlink/compose.auth.yaml "$@"
}
```

For the first start, run these commands. Check public HTTPS health after configuring the proxy:

```bash
pl config --quiet
pl pull
pl up -d --wait --wait-timeout 90
pl ps
curl -fsS http://127.0.0.1:3002/health/ready
```

The expected response is `{"status":"ready"}`. A GHCR permission error requires checking
package visibility; private images need registry credentials with read access. A Git SSH
key is not an image-registry credential.

### 3. Configure the 1Panel proxy and HTTPS

Create a reverse-proxy website in 1Panel: domain `pocketlink.mcloc.cn`, upstream
`http://127.0.0.1:3002`, a valid certificate and HTTP-to-HTTPS redirection.
See the [official instructions](https://1panel.cn/docs/v2/user_manual/websites/website_create/).

The current OpenResty uses host networking and can reach loopback. Bridge networking requires
a separately configured reachable private network; container loopback is not host loopback.
Do not publish 3002 to the internet or open UDP ports. `POCKETLINK_PUBLIC_ORIGIN` must exactly
match the browser's HTTPS origin, without a trailing slash or path.

Persistent files are inside the site directory. **Keep a full reverse proxy; never turn this
into a public static-file website.** Check the homepage and `/health/ready`, and confirm
`/pocketlink/data/relay.db` and `/pocketlink/secrets/admin_password` cannot be downloaded.
The current configuration returns 404.

## Maintenance and upgrades

- Status: `pl ps`; recent logs: `pl logs --tail 100 relay`.
- Stop only this service: `pl stop relay`; resume: `pl up -d --wait --wait-timeout 90`.
- Change/reset the administrator password: edit its file in 1Panel, preserving permissions and
  one line of 16..256 bytes, then run `pl up -d --force-recreate --wait --wait-timeout 90 relay`.
  Recreating ensures a file replaced by an editor is remounted. All admin sessions expire;
  device credentials remain unchanged. This also handles a forgotten password.
- Upgrade: read the changelog, back up data, both Compose files and `.env`, and record the old
  image digest. Change only `POCKETLINK_IMAGE` to the target CI digest, then run `pl config --quiet`,
  `pl pull`, `pl up -d --wait --wait-timeout 90` and verify health/login.
- Use `up -d` after Compose/environment changes; `restart` alone does not apply new container configuration.
- GitHub Actions tests, publishes images and packages firmware; **it does not update the server**.
  Main pushes create `edge` and commit tags; `relay/v*` tags publish versioned images.

Database migrations may prevent an older image from starting. Check compatibility before
rollback; incompatible databases require restoring a matching backup, not just switching images.
Restoration loses writes made after that snapshot, so stop usage first. Never run global Docker
cleanup, restart Docker/1Panel or modify unrelated services. Do not use `down -v` or delete data.

## Backup and recovery

The pre-text-upgrade backup is `backups/before-text-20260911T073934Z.tar.gz`; its database was restored to a temporary directory and passed integrity checks.
The initial backup is `backups/initial-20260911T011224Z.tar.gz`. It represents only that moment,
not an ongoing backup. SQLite may have WAL files: **never copy only a live `relay.db`**.

A manual consistent backup:

```bash
# Define pl above first. Only PocketLink is briefly stopped for this backup.
(
  set -e
  base=/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink
  stamp=$(date -u +%Y%m%dT%H%M%SZ)
  umask 077
  pl stop relay
  trap 'pl up -d --wait --wait-timeout 90' EXIT
  tar -czf "$base/backups/data-$stamp.tar.gz" -C "$base" data
  tar -tzf "$base/backups/data-$stamp.tar.gz" >/dev/null
)
```

Listing the archive only checks readability, not database recoverability. For a restore drill,
extract your own backup into a private temporary directory, run SQLite `PRAGMA quick_check`
and validate with a matching application version. For an actual restore, stop this service,
move the current `data/` to a retained directory, restore the selected complete `data/`, check
ownership `65532:65532` and permissions, then force-recreate the container and verify health,
login and the device list. Do not overwrite a live database or delete current data first.

Separately protect `/opt/pocketlink/` configuration and `secrets/admin_password`; they are needed
to restore administrator access. The archive command contains only database files. Encrypt
backups/secrets and keep a copy on another machine. Same-disk backups do not protect against
disk failure. No automatic backup or cleanup policy is configured; schedule it and monitor space.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| 502 | Check `pl ps`, local health, proxy port 3002 and OpenResty networking |
| Homepage/admin API 404 | Both auth settings are required; otherwise only foundation endpoints are enabled |
| Origin mismatch / 403 | Use the exact configured HTTPS domain; check slash, port and proxy; do not log in over HTTP/IP |
| 401 / expired login | Wrong password, 12-hour expiry, logout or restart; revoked device credentials also return 401 |
| 429 | Wait about a minute for the shared rate limit; the 32-session cap may require logging out older sessions or waiting for expiry |
| 409 | Check code/device/retained-serial limits; expired codes are pruned when creating another code |
| Cannot read password/database | Check UID/GID, permissions and mounts; do not open the whole site's permissions |
| Expired/used pairing code | Generate another; revoke first if the device exists but its response was lost |
| Missing device text/PTT controls | Server text transport is implemented; device firmware and audio remain pending |

## Local development and repository layout

| Directory | Purpose |
| --- | --- |
| `services/relay` | Go service, authentication, SQLite and protocol parsing |
| `apps/console` | Embedded console without third-party frontend dependencies |
| `firmware/apps/board-check` | Independent ESP-IDF hardware baseline |
| `firmware/apps/pocketlink` | QR provisioning, pairing and text reception development firmware |
| `firmware/components/pocketlink_config` | Configuration validation and host-testable inbox state |
| `firmware/boards/ai_passport` | BSP, licenses and provenance |
| `packages/protocol` | Protocol documentation and shared vectors |
| `deploy` | Compose templates; server-specific mount described above |
| `tools`, `tests`, `.github/workflows` | Validation, tests and CI |

Tools: Go 1.26.8, Node.js 24, Python 3.10+, a C compiler and actionlint 1.7.12; firmware also
requires ESP-IDF 5.5.3. See [environment setup](docs/development/setup.en.md).

```bash
source tools/env.sh
(cd services/relay && go run ./cmd/server)
# In another terminal; without auth configuration there is no console.
curl -fsS http://127.0.0.1:8080/health/ready
./tools/validate.sh --static
# Activate ESP-IDF 5.5.3 before building/verifying firmware:
./tools/validate.sh --firmware
```

Local HTTP defaults to 8080; server Docker maps port 3002. Merged firmware is in
`dist/firmware/board-check/` and `dist/firmware/pocketlink/`; Actions uploads separate firmware and SHA256 files. Successful builds
are not device validation. Preserve 8 MB Flash, the 3 MB application limit and identity at
`0x356000`; do not erase the entire chip or publish device identity data.

Maintained documents use Simplified Chinese at `.md` and English at sibling `.en.md` paths,
with reciprocal links at the top. Update both languages and distinguish plans from implemented features.

- [Architecture and stages](docs/architecture/overview.en.md)
- [Device authentication and API](docs/development/authentication.en.md)
- [Protocol format](packages/protocol/README.en.md)
- [Deployment record](docs/operations/deployment.en.md)
- [Hardware constraints and provenance](firmware/boards/ai_passport/README.en.md)
- [Changelog](docs/CHANGELOG.en.md) · [Third-party notices](docs/THIRD_PARTY.en.md)

No project-wide license has been selected. Imported AI Passport source retains its MIT license;
dependency licenses remain applicable.
