简体中文 | [English](AGENTS.en.md)

# PocketLink 仓库约定

首先阅读本文件。范围见[架构](docs/architecture/overview.md)，工具与检查见
[环境准备](docs/development/setup.md)，部署前阅读[运维](docs/operations/deployment.md)。

- 开始先执行 `git status --short --branch`，保留用户修改，不修改其他仓库。
- 使用 `codex/*` 分支及英文 Conventional Commit 提交信息。用户已授权相关检查通过后
  提交、推送，以及部署本项目经验证的版本；此授权不包含修改其他服务。
- 不得把后续阶段功能当作已完成。当前包含 P2 服务端与 P3 配网/收信及签名 OTA 开发固件；真机验收、音频仍待完成。
- 执行 `./tools/validate.sh --static`；修改硬件、基线、依赖或固件构建工具时，
  激活 ESP-IDF 5.5.3 后执行 `--firmware`。
- 业务状态和协议保持独立于界面、硬件，可在主机测试。
- 服务编译不等于 Docker 启动测试；固件编译不等于设备测试。
  分别报告 Build、Host tests、Device tests、Unverified。
- 保持 ESP32-C3、8 MB Flash、无 PSRAM、3 MB 应用上限及 0x356000 的 cardid。
  禁止擦除已配置设备 Flash，禁止在发布物中包含设备身份数据。
- 板级代码位于 `firmware/boards/ai_passport/components/bsp`；界面及应用任务位于
  `firmware/apps`；可复用客户端位于 `firmware/components`。外部访问 LVGL 必须持有
  BSP 锁。按键回调只投递工作；删除界面前停止所有生产者。
- 硬件事实优先采用实测规格，其次 `bsp_pins.h`、BSP 接口和实现；未定义电气信息先询问。
- 使用有界队列、认证身份、房间权限和协议限制。不记录凭证、文字或音频载荷。
  真实密钥不得进入 Git。
- 所有维护的 Markdown 默认使用简体中文 `.md`，英文使用同目录 `.en.md`，保留双向链接。
- 用户可见变更记入 `docs/CHANGELOG.md` 及英文版。
- 生产应用全部用 Docker。命令明确指定 Compose 文件和 `pocketlink-prod` 项目，
  先检查资源与端口。禁止全局 Docker 清理、重启 Docker/1Panel、修改其他应用。
- P0/P1 不执行生产部署，等待后续部署阶段。
- 保留第三方声明及固定版本导入记录，不复制原 AI Passport 工作区的未提交代码。
