[English](README.md) | 简体中文

# PocketLink

面向口袋设备互联、中转服务和网页应用的单仓库项目。

**当前阶段：P2a 设备鉴权。** 中转服务已提供配置校验、SQLite 迁移、健康接口、协议解析器
及可选的管理员和设备鉴权。文字消息、WebSocket 会话、语音转发和游戏仍待实现。
`board-check` 是导入的硬件演示基线，不是对讲机固件。

## 目录

| 目录 | 用途 |
| --- | --- |
| `services/relay` | Go 中转服务 |
| `apps/console` | 嵌入式登录、设备配对和撤销管理页 |
| `firmware/apps/board-check` | 可独立编译的上游硬件基线 |
| `firmware/boards/ai_passport` | 共享 BSP 及来源记录 |
| `packages/protocol` | 协议 v1 规范和共享测试向量 |
| `deploy` | 隔离的 Docker Compose 配置 |
| `tools`、`tests`、`.github/workflows` | 本地及 CI 检查、发布 |

后续有实际代码时，再建立 `firmware/apps/intercom`、`firmware/components`、
`packages/web-sdk` 及其他应用、服务目录。

## 启动

Go 1.26.8 和 ESP-IDF 5.5.3 的使用方式见[环境准备](docs/development/setup.zh_CN.md)。

```bash
source tools/env.sh
cd services/relay
go run ./cmd/server
```

HTTP 默认监听 `127.0.0.1:8080`。可检查 `/health/ready` 和 `/api/v1/capabilities`。
只有同时提供两项鉴权配置才启用业务功能，见[设备鉴权](docs/development/authentication.zh_CN.md)。

- [架构和阶段](docs/architecture/overview.zh_CN.md)
- [协议 v1](packages/protocol/README.zh_CN.md)
- [部署与回滚](docs/operations/deployment.zh_CN.md)
- [硬件导入及约束](firmware/boards/ai_passport/README.zh_CN.md)
- [第三方声明](docs/THIRD_PARTY.zh_CN.md)
- [更新记录](docs/CHANGELOG.zh_CN.md)

项目尚未选择整体许可证。导入的 AI Passport 源码遵循附带的 MIT 许可证；依赖各自许可证仍适用。
