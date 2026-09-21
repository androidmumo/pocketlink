[简体中文](accounts.md) | English

# Invite registration and shared rooms

## Usage

1. Select Administrator on the login page and use the existing server password file. There is no default password; registration never grants administrator privileges.
2. Open Invitations, create a registration code and share it privately. The full code is displayed only once. It expires after seven days, can be consumed once, and can be revoked before use.
3. The recipient selects Invite registration, supplies a username, password and code, then signs in with the new account.
4. Generate a device pairing code under My devices. The device belongs to the account that generated that code. Device codes still expire after ten minutes and are separate from registration invitations.
5. Create a room and choose Invite under Room partners. A signed-in friend enters that room code under Join a friend's room to accept.
6. Each person chooses their own receiving devices. New users cannot read pre-join history; new devices do not receive old messages.
7. Owners can remove partners; partners can leave. Their devices leave that room and pending delivery is withdrawn. Text already displayed on an offline device cannot be instantly erased.

Usernames are normalized to lowercase, contain 3–32 ASCII letters, digits or underscores, and start with a letter. `admin` is reserved. Passwords require at least 12 characters and at most 128 UTF-8 bytes; whitespace is preserved. Email verification, password recovery, account renaming and account deletion are not available. Keep your password safe.

## Permissions

| Operation | Administrator | Regular user / room partner |
| --- | --- | --- |
| Registration invites | Create, list own records and revoke | Cannot create |
| Devices | Manage all devices | Own devices only |
| Rooms | Manage all rooms | Create rooms; access owned or accepted rooms |
| Room invites, archive, remove partners | Any room | Owner only; partners can leave |
| Add receiving devices | Manage all devices | Add own devices only |
| Messages and receipts | All rooms | Accessible rooms, messages after joining |
| Firmware packages and channel | Upload, publish, withdraw, delete | No management access; devices retain authenticated access to published signed updates |

Administrators have global access; account isolation is not end-to-end encryption against administrators. SN is an identifier, never a credential. Even a revoked SN cannot be claimed using another account's pairing code. Cross-account device transfers are not implemented.

## API

Browser writes retain exact Origin validation, Secure/HttpOnly/SameSite=Strict cookies and no CORS.

| Method and path | Input / result |
| --- | --- |
| `POST /api/v1/auth/login` | `{username,password}`; `admin` uses the password file; omitted username preserves the legacy admin API |
| `POST /api/v1/auth/register` | `{username,password,code}`; atomically consumes a registration invitation; sign in afterward |
| `GET /api/v1/auth/me` | `{authenticated,user:{id,username,created_at},role}` |
| `GET /api/v1/invitations` | Own invitation metadata, without codes or hashes |
| `POST /api/v1/invitations` | `{kind:"registration"}` or `{kind:"room",room_id}`; one-time `{id,code,expires_at}` response |
| `DELETE /api/v1/invitations/{id}` | Revoke an invitation created by the current user |
| `POST /api/v1/rooms/join` | `{code}`; accept a room invitation, returns `{room_id}` |
| `GET /api/v1/rooms/{id}/users` | Room owner and accepted partners |
| `DELETE /api/v1/rooms/{id}/users/{user_id}` | Owner removes a partner, or current user leaves; associated device delivery is withdrawn |

Existing device, room and message paths remain, with server-side account authorization. Client-provided roles and ownership are not accepted. A message-ID boundary prevents pre-join history leakage even within the same second; receipt lookup and idempotent retries enforce that boundary too.

## Storage, limits and upgrades

Migration `005_accounts.sql` introduces users, invitations, room users and ownership of devices/pairings/rooms. Existing records belong to `admin`; device credentials, messages and receipts remain intact. User passwords use independent random salts and 600,000 PBKDF2-HMAC-SHA256 iterations. Invitations contain 256 random bits; only SHA-256 digests are stored, and consumption is atomic with registration or joining.

Limits: 256 accounts including administrator, 1,024 unexpired invitation records, 256 browser sessions with at most 16 per account. Sessions expire after 12 hours or a server restart. Creating invitations removes expired records; the list is not a long-term audit log. Login, registration and device pairing share 20 attempts per minute globally, with one concurrent password computation. Message-related writes retain the global 120/minute limit. Existing device/room/message limits remain.

Back up the database and deployment configuration before upgrading. Old images cannot open the new schema; rollback requires the matching pre-upgrade database and loses subsequent writes. The administrator password still comes from its existing file, never the users table or Git.

## Validation

Host tests cover populated legacy migration, cross-account denial, expiry/revocation/concurrent single consumption, database reopen, pre-join history isolation, removal withdrawing delivery, SN ownership and administrator-only firmware management. `tests/console/browser.cjs` only accepts a disposable loopback HTTPS server. It exercises registration/login, pairing, room messaging/receipts and desktop/mobile layout without production data.
