[简体中文](AGENTS.md) | English

# PocketLink repository rules

Read this file first. Read [architecture](docs/architecture/overview.en.md) for scope,
[setup](docs/development/setup.en.md) for tools and checks, and
[operations](docs/operations/deployment.en.md) before deployment.

- Start with `git status --short --branch`. Preserve user changes and other repositories.
- Use `codex/*` branches. Use English Conventional Commit subjects. The user authorizes
  commits and pushes after relevant checks pass, and deployment of this project's
  validated versions; this does not authorize changes to unrelated services.
- Do not implement or advertise later milestones as completed. Current scope is P2 device authentication; rooms and messaging remain pending.
- Run `./tools/validate.sh --static`; run `--firmware` with ESP-IDF 5.5.3 when
  board code, baseline code, dependencies or firmware build tools change.
- Keep business state/protocols testable independently from UI and hardware.
- A server build is not a Docker smoke test; a firmware build is not a device test.
  Report Build, Host tests, Device tests and Unverified separately.
- Preserve ESP32-C3, 8 MB Flash, no PSRAM, 3 MB application limit and cardid at
  0x356000. Never erase provisioned Flash or include identity data in artifacts.
- Shared board code belongs in `firmware/boards/ai_passport/components/bsp`.
  UI/application tasks belong in `firmware/apps`, portable client logic in
  `firmware/components`. External LVGL access requires the BSP lock.
  Button callbacks only enqueue work. Stop producers before deleting UI objects.
- Hardware facts follow measured specifications, then `bsp_pins.h`, BSP interfaces
  and implementation. Ask when an electrical fact is unspecified.
- Use bounded queues, authenticated identities, room permissions and protocol limits.
  Never log credentials or message/audio payloads. Keep real secrets out of Git.
- Every maintained Markdown document defaults to Simplified Chinese at `.md`, with
  a sibling English `.en.md` and reciprocal links. Update both together.
- Record user-visible changes in `docs/CHANGELOG.md` and its English pair.
- All production applications run in Docker. Scope every command to the explicit
  Compose file and `pocketlink-prod` project. Inspect resources/ports first.
  Never run global Docker cleanup, restart Docker/1Panel, or alter other applications.
- Do not deploy during P0/P1. Wait for the later deployment milestone.
- Preserve third-party notices and the pinned upstream import record. Do not copy
  uncommitted code from the original AI Passport workspace.
