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

WebSocket uses `github.com/coder/websocket` v1.8.15 (ISC license), pinned in the Go module files.

The PocketLink bitmap font is generated from Adobe Source Han Sans 2.004R, `SourceHanSansSC-Regular.otf`, under [SIL OFL 1.1](../firmware/components/pocketlink_font/LICENSE.txt). The official source URL and SHA256 are pinned in `tools/generate-font.py`; the converter is pinned to `lv_font_conv@1.5.3`. The derivative is named `pocketlink_font_14` and covers ASCII, 6,763 GB2312 Han characters and UI text, not all Unicode or emoji. Regenerate with `python3 tools/generate-font.py` (requires Node.js/npm and network). Ordinary builds use committed generated data without downloading the font.
