[简体中文](THIRD_PARTY.md) | English

# Third-party notices

AI Passport source, baseline configuration, UI math tests, firmware verifier and
its tests were imported from FoloToy under MIT. The original copyright and license
are preserved in [LICENSE](../firmware/boards/ai_passport/LICENSE).
Import provenance is in [upstream.json](../firmware/boards/ai_passport/upstream.json).
The verifier tests only change their repository-root lookup after relocation.

Go dependencies are recorded in `services/relay/go.mod` and `go.sum`; ESP-IDF
Managed Components are recorded in the board-check manifest/lock. Their original
licenses apply. No new project-wide license grant is implied by these notices.
