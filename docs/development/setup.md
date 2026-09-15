简体中文 | [English](setup.en.md)

# 开发环境

## 工具链

- Go 1.26.8；从 [Go 下载页](https://go.dev/dl/) 安装并校验摘要。
  `source tools/env.sh` 也会发现 `$HOME/.local/share/pocketlink/go1.26.8/bin`
  下的用户级安装，不修改 shell 启动文件。
- actionlint 1.7.12：`go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`。
  将 `$(go env GOPATH)/bin` 加入 PATH。
- Python 3.10+、C 编译器、Git、Bash。
- 固件使用 ESP-IDF 5.5.3 与 ESP32-C3 工具，先激活其 `export.sh`。
- 本地容器测试需要 Docker Engine/Desktop 和 Compose。不可用时由 GitHub Actions
  构建和验证镜像，不得宣称通过本地 Docker 测试。
- Node.js 24 用于管理页脚本语法检查。

## 检查

```bash
source tools/env.sh
./tools/validate.sh --static
source /path/to/esp-idf-v5.5.3/export.sh
./tools/validate.sh --firmware
./tools/validate.sh --all
```

静态验证涵盖双语文档、固定 Action、受保护默认配置、C/Python 固件主机测试、Go vet/race
测试、协议向量，以及真实服务启动、数据库重开和 SIGTERM 测试。
固件验证使用临时构建目录和 sdkconfig，检查合并镜像、分区 MD5、应用大小与身份保护。
产物位于 `dist/firmware/board-check/`。禁止提交生成物、数据库和密钥。

SQLite 依赖固定在 `services/relay/go.mod` 和 `go.sum`。
ESP-IDF 组件固定在 `firmware/apps/board-check/dependencies.lock`。重新生成锁文件后必须审查变化。

## 本地服务配置

| 变量 | 默认值 | 校验 |
| --- | --- | --- |
| `POCKETLINK_LISTEN` | `127.0.0.1:8080` | IP 字面量或通配地址，端口 1-65535 |
| `POCKETLINK_DATABASE` | `./data/relay.db` | 文件路径，不支持 SQLite URI 和内存模式 |
| `POCKETLINK_LOG_LEVEL` | `info` | debug/info/warn/error |
| `POCKETLINK_SHUTDOWN_TIMEOUT` | `10s` | 1s-1m |

Go 程序不自动加载 `.env`。Compose 解析部署变量，shell 启动使用导出的环境变量。
P1 程序提供本地或反向代理使用的 HTTP，不用于直接公开凭证接口或 WSS。

## 发布流水线

使用 `codex/*` 分支。PR、main、codex 分支、手动请求、`relay/v*` 或
`firmware/board-check/v*` 标签触发 CI。当前统一执行全部基础检查，后续明确共享依赖后再优化路径触发。
镜像任务等待静态与固件检查通过，先测试本机架构容器；仅 main 或 relay 标签发布 amd64/arm64 镜像。
main 生成 `edge` 及提交标签；relay 标签生成对应版本和提交标签。
固件及 SHA256 作为 CI 附件上传，不自动刷机。
流水线不保存生产 SSH 凭证，也不自动部署生产。

## 设备鉴权

通过 HTTPS 来源和密码文件启用管理页及设备接口，见[鉴权说明](authentication.md)。
前端无第三方依赖，直接嵌入源码资源；Node.js 24 用于 JavaScript 语法检查，无需打包工具。

`--firmware` 同时构建 `board-check` 与 `pocketlink`，后者产物位于 `dist/firmware/pocketlink/`。
两者依赖锁均须保持固定，CI 分别上传附件。新增配置、收信状态和手机页面逻辑主机测试。
配网开发固件使用教程见[设备配网](device-provisioning.md)。
