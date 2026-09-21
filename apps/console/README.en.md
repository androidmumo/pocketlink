[简体中文](README.md) | English

# Device management console

The console provides administrator login/logout, one-time pairing code creation,
device listing and credential revocation. It is a small embedded HTML/CSS/JavaScript
application with no third-party frontend dependencies or external asset requests.
The local Go module embeds source assets directly; relay builds include it through
an explicit relative module replacement. No generated asset copies are checked in.

The UI is served at `/` only when authentication is configured. It uses same-origin
requests and an HttpOnly session cookie; credentials are never stored in localStorage.
Device names and serials are rendered as text. A strict CSP disallows inline scripts.
See [authentication](../../docs/development/authentication.en.md) for setup and API contracts.

Rooms, membership, text composition, paginated history and per-device receipts are implemented. Audio remains pending. The pairing
code must later be transferred through the device configuration flow, not typed with
three device buttons. No pairing firmware is included in this increment.

The responsive workspace includes overview, devices, rooms/messages, invitations and administrator-only firmware management. See [accounts and invitations](../../docs/development/accounts.en.md).
