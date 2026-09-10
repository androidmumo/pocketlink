English | [简体中文](overview.zh_CN.md)

# Architecture and delivery stages

One repository contains independently versioned applications. Browser apps live
in `apps`, server processes in `services`, and independent ESP-IDF projects in
`firmware/apps`. Future games follow their execution environment instead of
mixing firmware and server source in a single games folder.

The relay is a modular Go monolith. SQLite uses WAL, FULL synchronous mode,
foreign keys, a five-second busy timeout, and one pooled connection. Migrations
run in an immediate transaction; checksum drift, missing earlier migrations and
unknown future migrations fail startup. A database from a newer release must not
be opened by an older image without a documented compatible rollback.

The current HTTP process exposes liveness, storage-backed readiness and honest
capabilities. It stops readiness before draining on SIGTERM, then closes storage.
No public business endpoints or authentication placeholders are exposed.

## Milestones

| Stage | Deliverable | Acceptance |
| --- | --- | --- |
| P0/P1 | Repository, board baseline, config, storage, protocol, CI/images | Local host/firmware gates and CI image smoke test |
| P2 | Browser login, one-time pairing, per-device credentials, rooms, text, receipts | Idempotency, ACL isolation, offline redelivery, restart recovery |
| P3 | Device provisioning and text reception | Two-board network/message and reboot tests |
| P4 | Half-duplex PTT and WSS PCM audio | Simultaneous PTT, early release, disconnect and lease expiry |
| P5 | DTLS/UDP, ADPCM, jitter buffer, WSS fallback | Packet loss, late drop, latency and memory measurements |
| P6 | Production release, backups, rollback, user guide | Isolated deployment and existing services remain healthy |

## Decisions reserved for later stages

- Identity: SN is a label, not a credential. Exchange a short-lived pairing code
  for a unique revocable device secret over verified TLS.
- Routing: authenticated sessions determine senders. Server ACLs decide rooms
  and namespaces; client fields never grant authority.
- Reliable traffic: WSS text/control, at-least-once delivery with idempotency.
- Real-time traffic: DTLS/UDP preferred; independent binary WSS fallback.
  Late frames are dropped and never replayed after reconnect.
- PTT: one server-issued lease per room. Late grants after button release are
  released immediately. Transport changes allocate a new stream epoch.
- Audio target: 16 kHz mono, 20 ms blocks; PCM first, independent IMA ADPCM blocks
  later. Codec interoperability, voice quality and resource use require hardware tests.
- SQLite stores text and receipts. Live audio is not recorded. Future large
  attachments use separate bounded upload/download storage.
- UI: hold OK to talk on the home page; text reading and settings use separate
  pages. Firmware handles common Chinese glyph coverage explicitly.
- No production deployment in P0/P1. No game engine, E2EE or clustering in v1.
