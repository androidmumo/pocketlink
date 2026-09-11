English | [简体中文](authentication.zh_CN.md)

# Device authentication increment (P2a)

P0/P1 artifacts have passed CI. P2a adds a single administrator and independent device
credentials; it does not complete P2 rooms, text delivery, receipts or WebSocket sessions.
The board-check image is unchanged. Production rollout remains deferred.

## Configuration

Both settings are required to enable business endpoints and the embedded console:

| Setting | Contract |
| --- | --- |
| `POCKETLINK_PUBLIC_ORIGIN` | Exact HTTPS origin, e.g. `https://pocketlink.example`; no trailing slash/path/query |
| `POCKETLINK_ADMIN_PASSWORD_FILE` | Mounted file containing one 16..256-byte password line; optional final LF/CRLF |

Without both settings the service retains foundation mode. Partial or malformed
configuration fails startup. The password file is read at startup only, never included
in logs or passed in command arguments. Use a long unique random password. Rotate the
file and restart only this service to change it and invalidate every admin session.

The HTTP backend must remain on the loopback interface or an isolated proxy network.
Terminate TLS at the dedicated reverse proxy. Do not expose the backend port publicly.
No forwarded headers are trusted. The configured Origin is checked exactly on browser
mutations, even when the reverse proxy rewrites Host. No CORS access is enabled.
Device clients must validate the TLS certificate and hostname. This increment does
not establish or modify the proxy, certificate, DNS or production server.

`deploy/compose.auth.yaml` is an optional Compose override. Set `POCKETLINK_ADMIN_SECRET_SOURCE`
to a file outside the checkout that container UID 65532 can read, and supply the HTTPS
origin. Compose mounts the file read-only; source file permissions must be set explicitly
because local bind-backed secrets do not remap ownership. Keep parent directories private.
Validate the merged configuration before any later deployment:

```bash
docker compose --env-file /opt/pocketlink/.env -p pocketlink-prod \
  -f deploy/compose.yaml -f deploy/compose.auth.yaml config --quiet
```

## API

JSON request bodies require `Content-Type: application/json` and are capped at 4096 bytes.
Errors contain stable generic codes, never SQL details or credential values.

| Method/path | Credential | Request/result |
| --- | --- | --- |
| `POST /api/v1/auth/login` | Password + exact Origin | `{password}`; sets session cookie |
| `GET /api/v1/auth/me` | Admin cookie | Login state |
| `POST /api/v1/auth/logout` | Admin cookie + Origin | Invalidates current session |
| `POST /api/v1/pairings` | Admin cookie + Origin | `{name}`; returns `{code, expires_at}` |
| `GET /api/v1/devices` | Admin cookie | `{devices}` including revoked status; no hashes/secrets |
| `DELETE /api/v1/devices/{id}` | Admin cookie + Origin | Immediately revokes credential; idempotent for known ID |
| `POST /api/v1/device/pair` | One-time code; no browser Origin | `{code, sn}`; returns `{device, credential}` once |
| `GET /api/v1/device/me` | `Authorization: Bearer <credential>` | Current active device identity |

An administrator session cannot authorize a device endpoint; a device token cannot
administer devices. Session cookies are Secure, HttpOnly, SameSite=Strict and host-only,
with a 12-hour lifetime. At most 32 sessions are retained in memory; restart logs them out.
PBKDF2-HMAC-SHA256 uses a random startup salt and 600,000 iterations. Password verification
is limited to one concurrent operation. Login and device pairing share a global limit
of 20 attempts/minute, independent of spoofable proxy/client-IP headers. This deliberately
favors a small private deployment; an attacker can temporarily exhaust this shared quota.

Pairing codes and device tokens contain 256 random bits. SQLite stores their SHA-256
digests only. Pairing codes expire after 10 minutes and are consumed atomically. Maximum
32 unexpired codes, 256 active devices and 1024 retained distinct serials. Device names
are 1..40 Unicode characters without controls. SN is an identifier, not a secret:
1..64 ASCII letters/digits plus dot, underscore, colon or hyphen, starting with a letter/digit.

An active SN cannot be silently replaced. Revoke it before re-pairing; re-pairing retains
its ID and replaces the credential. If the pair response is lost, the consumed code cannot
recover the secret: revoke the listed device and generate another pairing code. Admins
must protect displayed pairing codes. Device tokens have no automatic expiry in this
increment; revoke and re-pair to rotate them. No credential is placed in a URL.

## Verification and remaining work

Host tests cover origin checks, session expiry/logout, role isolation, bounded requests,
concurrent single-use exchange, expiry, quotas, revocation, re-pairing and SQLite reopen.
Process smoke checks also exercise configured mode and restart behavior. UI assets are
embedded and checked by HTTP tests and JavaScript syntax validation. These checks do not
prove browser interaction, TLS proxy setup, physical device storage or end-to-end pairing.
Migration 002 adds devices/pairings; an older image rejects the newer database. Retain a
consistent pre-upgrade backup before a future production migration.
