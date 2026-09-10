English | [简体中文](README.zh_CN.md)

# PocketLink

A monorepo for connected pocket devices, relay services and web applications.

**Current status: P0/P1 foundation.** The relay provides validated configuration,
SQLite migrations, health endpoints and protocol parsers. It does **not** yet
provide authentication, messaging, WebSocket sessions, voice relay or games.
`board-check` is the imported hardware demo, not intercom firmware.

## Layout

| Directory | Purpose |
| --- | --- |
| `services/relay` | Go relay service |
| `apps/console` | Reserved for the P2 administration UI; no application yet |
| `firmware/apps/board-check` | Independently buildable upstream hardware baseline |
| `firmware/boards/ai_passport` | Shared BSP and provenance |
| `packages/protocol` | Protocol v1 specification and shared test vectors |
| `deploy` | Isolated Docker Compose configuration |
| `tools`, `tests`, `.github/workflows` | Local/CI validation and releases |

Future apps will create `firmware/apps/intercom`, `firmware/components`,
`packages/web-sdk` and additional app/service directories when they contain code.

## Start

See [setup](docs/development/setup.md) for Go 1.26.8 and ESP-IDF 5.5.3.

```bash
source tools/env.sh
cd services/relay
go run ./cmd/server
```

The HTTP listener defaults to `127.0.0.1:8080`. Check `/health/ready` and
`/api/v1/capabilities`. There are no enabled business features yet.

- [Architecture and milestones](docs/architecture/overview.md)
- [Protocol v1](packages/protocol/README.md)
- [Deployment and rollback](docs/operations/deployment.md)
- [Board import and hardware constraints](firmware/boards/ai_passport/README.md)
- [Third-party notices](docs/THIRD_PARTY.md)
- [Changelog](docs/CHANGELOG.md)

No project-wide license has been selected yet. Imported AI Passport source remains
under its included MIT license; dependency licenses remain applicable.
