简体中文 | [English](THIRD_PARTY.en.md)

# 第三方声明

AI Passport 源码、基线配置、UI 数学测试、固件校验器及其测试从 FoloToy 按 MIT 导入。
原版权和许可保留在 [LICENSE](../firmware/boards/ai_passport/LICENSE)。
导入来源见 [upstream.json](../firmware/boards/ai_passport/upstream.json)。
校验器测试仅为适应新位置调整了仓库根目录查找。

Go 依赖记录在 `services/relay/go.mod`、`go.sum`；ESP-IDF 托管组件记录在 board-check
清单及锁文件，各自原许可证仍适用。本文不代表授予新的项目整体许可证。

WebSocket 使用 `github.com/coder/websocket` v1.8.15（ISC 许可证），版本及摘要固定在 Go 模块文件中。

PocketLink 中文点阵字体由 Adobe Source Han Sans 2.004R 的 `SourceHanSansSC-Regular.otf` 生成，采用 [SIL OFL 1.1](../firmware/components/pocketlink_font/LICENSE.txt)。官方来源和 SHA256 固定在 `tools/generate-font.py`，转换器固定为 `lv_font_conv@1.5.3`。生成物使用独立名称 `pocketlink_font_14`，覆盖 ASCII、GB2312 的 6763 个汉字及界面文案；不保证所有 Unicode 字符或 emoji。重新生成：`python3 tools/generate-font.py`（需要 Node.js/npm 和网络），无需在普通构建中重新下载字体。

设备语音使用 Espressif `esp_websocket_client` 1.6.1（Apache-2.0），固定版本和组件摘要记录于固件锁文件。BSP 仅增补释放事件和固定 LVGL 9.5.0，适配记录见 upstream.json。
