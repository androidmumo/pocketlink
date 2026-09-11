[简体中文](messaging.md) | English

# Rooms, reliable text and receipts (P2)

The console creates rooms, adds/removes paired devices, sends text, pages history and shows
per-device receipts. Devices receive over WSS; an HTTPS inbox supports debugging/polling.
Intercom firmware and audio remain unimplemented.

## Delivery and authorization

- The administrator is the only sender. Devices can only retrieve assigned messages and acknowledge their own receipts.
- Each message and its active-member snapshot commit in one SQLite transaction. New members do not receive history.
- `request_id` is unique within a room. Retrying the same key/body returns the original message; a changed body returns 409.
  After network failure, the console retains the retry key for that content until reload; check history before resending after reload.
- Delivery is at least once. Devices deduplicate by message `id`, persist before acknowledging `received`, and send `read`
  only after displaying it. Duplicate or late received acknowledgements do not overwrite read status.
- A socket write is not delivery confirmation. Unacknowledged messages replay after reconnect; acknowledged messages do not
  replay automatically. Never acknowledge while the only copy is in volatile memory.
- Removing membership, archiving or revoking a device withdraws delivery rights. Already transmitted content cannot be recalled.
  Re-adding does not restore old messages. Re-pairing retains the device ID but requires new room authorization.
- Archived rooms retain history/receipts but disallow sending and new membership. Previously committed idempotent requests still return their original result.

## Administrator API

Existing cookie, Origin and JSON rules apply. Timestamps are Unix seconds; message IDs are increasing integers.

| Method/path | Request / response |
| --- | --- |
| `POST /api/v1/rooms` | `{name}` → room |
| `GET /api/v1/rooms` | `{rooms}`, including archived rooms |
| `DELETE /api/v1/rooms/{room}` | Archive and withdraw delivery rights |
| `GET /api/v1/rooms/{room}/members` | `{device_ids}` |
| `PUT /api/v1/rooms/{room}/members/{device}` | Add an active device; no body |
| `DELETE /api/v1/rooms/{room}/members/{device}` | Remove membership; no body |
| `POST /api/v1/rooms/{room}/messages` | `{request_id,text}` → `{id,room_id,request_id,text,created_at}` |
| `GET /api/v1/rooms/{room}/messages?before={id}` | `{messages,next_before}`, newest first, 20 per page, 0 ends pagination |
| `GET /api/v1/messages/{id}/receipts` | `{receipts}` with per-device `delivered_at`, `read_at`, `withdrawn_at`; null if absent |

Request keys use 1..64 protocol identifier characters; UUIDs work. Text allows 1..200 Unicode codepoints,
at most 2048 UTF-8 bytes, no whitespace-only content or controls except newline/tab. Empty rooms return `empty_room`.
One database retains at most 64 rooms (including archived) and 10000 messages. Capacity returns 409 without silent deletion.
Room/message mutations share a 120/minute limit. History/receipts refresh on demand and do not imply device online presence.

## Device HTTPS API

Use only `Authorization: Bearer <credential>`, never an admin cookie. Browser Origin headers are rejected.
Credentials belong in headers, never URLs. Devices must validate TLS certificates and hostnames.

- `GET /api/v1/device/inbox`: up to 20 oldest unacknowledged messages, without a client cursor that can skip messages.
- `POST /api/v1/device/ack`: `{message_id,state}`, with state `received` or `read`. Unauthorized or withdrawn receipts fail.

## Device WSS protocol

Connect to `wss://pocketlink.mcloc.cn/api/v1/device/stream` with a Bearer header and `pocketlink.v1`
subprotocol. No query parameters or compression. Maximum 64 streams, one per device; duplicates return 409.
Back off after abnormal disconnects. Capacity is checked before and after upgrade; credentials are continuously revalidated.

Pending queues are checked every second. One unacknowledged message is in flight per connection and retransmitted after 5 seconds.
Writes time out after 5 seconds. Ping every 20 seconds, close after a 5-second timeout. Revocation closes old connections.
Shutdown closes and waits for stream handlers before closing the database.

Server message using the [v1 envelope](../../packages/protocol/README.en.md):

```json
{"version":1,"type":"text.message","request_id":"send-001","namespace":"text","room_id":"r123","payload":{"id":1,"room_id":"r123","request_id":"send-001","text":"Hello","created_at":1789100000}}
```

Device acknowledgement:

```json
{"version":1,"type":"text.ack","request_id":"ack-001","namespace":"text","payload":{"message_id":1,"state":"received"}}
```

Acknowledgements omit `room_id`; the server derives authorization from the message and credential. Clients cannot declare
senders or recipients. After persistence, `ack.result` echoes the request ID with `{message_id,acknowledged:true}`.
Retry lost acknowledgement results safely. At most 120 application acknowledgements/minute/connection, otherwise it closes.
Only text control frames up to 8192 bytes are accepted. Binary frames, unknown types, malformed envelopes or acknowledgements close the stream.

## Operations

Migration 003 adds rooms, members, messages and receipts. Old P2a images reject this database; take a consistent backup before upgrading.
No automatic expiry, deletion or attachments are implemented. Port stays `127.0.0.1:3002` and the data mount is unchanged.
1Panel must preserve WebSocket Upgrade/Connection forwarding. WSS uses HTTPS 443; no UDP port is opened.
Host integration tests simulate reconnect/replay, room isolation, monotonic receipts, revocation and shutdown; they do not replace hardware tests.
