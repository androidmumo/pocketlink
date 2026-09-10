English | [简体中文](README.zh_CN.md)

# PocketLink protocol v1 draft

Status: envelope and real-time frame parsers plus shared vectors are implemented.
Transport handshakes and business handlers are scheduled for P2-P5. Do not treat
this specification as an available relay endpoint. Breaking changes before the
first device release require synchronized vectors and parsers.

## Control envelope

A WSS text message is UTF-8 JSON, at most 8192 bytes and at most 16 nested
containers. Reject duplicate keys, trailing values, unknown envelope fields,
invalid UTF-8, unsupported versions and non-object payloads. Identifiers match
`[a-zA-Z0-9][a-zA-Z0-9._:-]{0,63}`.

```json
{"version":1,"type":"hello","request_id":"r1","namespace":"intercom","payload":{"protocol_versions":[1]}}
```

Required fields are `version`, `type`, `request_id`, `namespace`, `payload`.
`room_id` is optional and required by future room handlers. The session supplies
sender identity, never a client `sender_id`. Payload validation belongs to each
handler; syntactic acceptance is not authorization or a known message type.

Future control types: `hello`, `auth`, `room.join`, `text.send`, `text.ack`,
`text.read`, `ptt.request`, `ptt.granted`, `ptt.release`, `stream.open`, `error`.
Responses echo request IDs; durable messages use server IDs and stable per-sender
idempotency keys in their payload. Reliable traffic has database-backed delivery
state; transient packets never acquire durability by setting a client flag.

Error codes reserved for handlers: `unsupported_version`, `invalid_message`,
`unauthenticated`, `forbidden`, `rate_limited`, `room_busy`, `stream_expired`,
`payload_too_large`, `temporarily_unavailable`. Errors never echo credentials.

Text limits: 1-200 Unicode code points and at most 2048 UTF-8 bytes; reject
C0 control characters except newline/tab, and reject DEL. Glyph availability is a
separate UI constraint. User text is plain text, not HTML.

## Real-time frame

All integers use network byte order. One frame occupies one DTLS application
record or one binary WSS message. The **plaintext** frame is at most 1100 bytes;
implementation must also cap the final UDP payload at 1200 bytes after negotiated
DTLS overhead. No application fragmentation. Drop oversize input before allocation.

| Offset | Bytes | Field |
| --- | --- | --- |
| 0 | 2 | ASCII `PL` |
| 2 | 1 | Version: 1 |
| 3 | 1 | Kind: 1 audio, 2 game state |
| 4 | 4 | Nonzero server-issued stream ID |
| 8 | 4 | Nonzero epoch |
| 12 | 4 | Sequence, starting at 0 |
| 16 | 8 | Audio sample timestamp; game ticks negotiated per stream |
| 24 | 2 | Nonzero TTL in milliseconds |
| 26 | 2 | Payload length, 1-1068 bytes, exact match |
| 28 | 2 | Flags, zero in v1 |
| 30 | 2 | Reserved, zero |
| 32 | variable | Opaque codec or game bytes |

The session/stream registry binds identity, room, codec, epoch, PTT grant and
server-approved TTL ceiling. The parser alone does not enforce these bindings.
Sequences may not wrap within an epoch; allocate a new epoch beforehand. Replays,
old epochs and packets without a valid lease must be rejected by the live handler.
TTL is not a wall-clock timestamp: receiver playback scheduling derives from
sample timestamps and a bounded local monotonic-clock jitter buffer. The sender
and relay drop queued frames older than their own enqueue time plus allowed TTL;
receivers drop frames missing playback deadlines. Remote wall clocks are not
trusted to measure one-way network delay.

The audio codec is negotiated per stream. Each ADPCM packet must carry predictor
and step-index state so losing a packet cannot corrupt all subsequent packets.
The test fixture carries opaque bytes; it does not certify a codec implementation.

See [vectors.json](vectors.json). Go parser tests consume this same file; future
firmware tests must consume equivalent fixtures before protocol changes ship.
