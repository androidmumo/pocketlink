[简体中文](docker.md) | English

# Simple guide: deploy the Docker server

**Skip this guide if you already have a server account.** One container includes the console and database; no separate frontend or database installation is needed.

Prepare Linux with Docker / Compose, a domain pointing to it, and an HTTPS reverse proxy. This example uses 1Panel and a root terminal. **Do not initialize an existing PocketLink installation again.** See [operations](../operations/deployment.en.md) for upgrades and backups. These instructions do not change the existing production installation.

## 1. Prepare private directories

Replace the domain below. If port 3002 is occupied, select another unused port and use it consistently in steps 2 and 3. Do not stop other services.

```bash
DOMAIN=pocketlink.example.com
ss -lnt | grep ':3002 ' || true
```

After confirming the port is available:

```bash
set -eu
STORAGE="/opt/1panel/www/sites/$DOMAIN/index/pocketlink"
test ! -e /opt/pocketlink
test ! -e "$STORAGE"
install -d -m 700 /opt/pocketlink "$STORAGE" "$STORAGE/secrets" "$STORAGE/backups"
install -d -m 700 -o 65532 -g 65532 "$STORAGE/data"
curl -fL https://raw.githubusercontent.com/androidmumo/pocketlink/main/deploy/compose.quickstart.yaml -o /opt/pocketlink/compose.yaml
python3 - "$STORAGE/secrets/admin_password" <<'PYTHON'
import os, secrets, sys
with open(sys.argv[1], 'x') as f:
    f.write(secrets.token_urlsafe(32) + '\n')
os.chown(sys.argv[1], 65532, 65532)
os.chmod(sys.argv[1], 0o400)
PYTHON
```


The password is stored in `secrets/admin_password`; there is no default password. Keep the storage directory private.

## 2. Configure and start

Continue in the same terminal:

```bash
cat > /opt/pocketlink/.env <<ENV
POCKETLINK_IMAGE=ghcr.io/androidmumo/pocketlink-relay@sha256:ac2e12e8d191fbfc179335a61df0f24f1cf10df67a3e90eb33a4268cc9c3829a
POCKETLINK_PUBLIC_ORIGIN=https://$DOMAIN
POCKETLINK_HTTP_PORT=3002
POCKETLINK_STORAGE_ROOT=$STORAGE
ENV
chmod 600 /opt/pocketlink/.env
docker compose --env-file /opt/pocketlink/.env -p pocketlink-prod -f /opt/pocketlink/compose.yaml config --quiet
docker compose --env-file /opt/pocketlink/.env -p pocketlink-prod -f /opt/pocketlink/compose.yaml up -d --wait
curl -f http://127.0.0.1:3002/health/ready
```


Continue when the container is healthy and the health endpoint succeeds. The example pins a validated image. Choose future images from release notes and back up before upgrading; never delete the data directory.

## 3. Configure HTTPS in 1Panel

1. Create a **reverse-proxy website** for the domain, targeting `http://127.0.0.1:3002`.
2. Enable an HTTPS certificate and WebSocket forwarding for intercom.
3. Open `https://your-domain` and check the PocketLink login page.

This target assumes OpenResty uses host networking. With an isolated container network, its loopback is not the host; configure a reachable upstream for your 1Panel network instead of exposing port 3002 publicly.

Devices use the domain and HTTPS port **443**. Port 3002 is only for the local proxy and need not be opened in the cloud firewall. Allow website ports 80/443 for certificate issuance and HTTPS.

Although persistence lives beneath the site directory, keep the site as a **reverse proxy**, not a static site exposing that directory.

## 4. Sign in and add devices

In 1Panel file management, read `/opt/1panel/www/sites/your-domain/index/pocketlink/secrets/admin_password`. Select administrator login and use that password.

- For yourself: generate a device pairing code, then follow the [device guide](device.en.md).
- For friends: create a registration invitation; friends register and manage their own devices.
- For messages/voice: create a room and select the receiving devices.
- For OTA: download matching `manifest.json` and `pocketlink-ota.bin` from [Releases](https://github.com/androidmumo/pocketlink/releases), upload in Firmware management and publish.

Back up the **whole data directory**, `.env`, Compose configuration and password file before updates. Keep the database with its matching `.invitation-key`. Stop only this project's container during backup, or use a consistent SQLite backup; do not copy a single live database file. See [backup instructions](../technical.en.md).

[Home](../../README.en.md) · [Technical reference](../technical.en.md)
