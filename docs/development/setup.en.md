[简体中文](setup.md) | English

# Development setup

## Toolchains

- Go 1.26.8; install a checksum-verified archive from [Go downloads](https://go.dev/dl/).
  `source tools/env.sh` also finds a user-local installation at
  `$HOME/.local/share/pocketlink/go1.26.8/bin` without modifying shell startup files.
- actionlint 1.7.12: `go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`.
  Add `$(go env GOPATH)/bin` to PATH.
- Python 3.10+, a C compiler, Git, and Bash.
- ESP-IDF 5.5.3 with ESP32-C3 tools for firmware. Activate its `export.sh`.
- Docker Engine/Desktop plus Compose for local container testing. If unavailable,
  GitHub Actions builds and smoke-tests the image; do not claim a local Docker test.
- Node.js 24 is required for console JavaScript syntax checks.

## Checks

```bash
source tools/env.sh
./tools/validate.sh --static
source /path/to/esp-idf-v5.5.3/export.sh
./tools/validate.sh --firmware
./tools/validate.sh --all
```

Static validation covers paired docs, pinned actions, protected defaults, C and
Python firmware host tests, Go vet/race tests, protocol fixtures, and a real
server startup/reopen/SIGTERM test. Firmware validation uses a fresh temporary
build and sdkconfig, then verifies the merged image including partition MD5,
application size and protected identity. Output: `dist/firmware/board-check/`.
Generated build files, databases and secrets must not be committed.

SQLite dependency versions are pinned in `services/relay/go.mod` and `go.sum`.
ESP-IDF Managed Components are pinned in `firmware/apps/board-check/dependencies.lock`.
Review any regenerated dependency lock changes.

## Local service configuration

| Variable | Default | Validation |
| --- | --- | --- |
| `POCKETLINK_LISTEN` | `127.0.0.1:8080` | Literal IP (or wildcard) and port 1-65535 |
| `POCKETLINK_DATABASE` | `./data/relay.db` | File path, no SQLite URI or in-memory mode |
| `POCKETLINK_LOG_LEVEL` | `info` | debug/info/warn/error |
| `POCKETLINK_SHUTDOWN_TIMEOUT` | `10s` | 1s-1m |

There is no automatic `.env` loader in the Go binary. Compose resolves deployment
variables; shell runs use exported environment variables. The P1 process serves
plain HTTP for local/reverse-proxy use, not direct public credentials or WSS.

## Release workflow

Use `codex/*` branches. CI runs on pull requests, main, codex branches, manual
requests, and `relay/v*` or `firmware/board-check/v*` tags. All foundation gates run
for now; path-based optimization can follow once shared dependencies are mapped.
The image job waits for static and firmware gates, smoke-tests a native container,
and publishes multi-architecture amd64/arm64 images only on main or relay tags.
Main gets `edge` and a commit tag; relay tags get their version and commit tag.
Firmware output and SHA256 are uploaded as CI artifacts, not automatically flashed.
No workflow holds production SSH credentials or deploys production.

## Device authentication

See [authentication](authentication.en.md) to enable the console and device endpoints using
an HTTPS origin and password file. Frontend assets are dependency-free and embedded directly;
Node.js 24 is used for JavaScript syntax validation, not bundling.

`--firmware` builds both `board-check` and `pocketlink`; the latter outputs to
`dist/firmware/pocketlink/`. Both dependency locks remain pinned and CI uploads separate
artifacts. Host checks include configuration, inbox state and phone-page logic.
See [device provisioning](device-provisioning.en.md).
