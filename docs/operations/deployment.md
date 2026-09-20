简体中文 | [English](deployment.en.md)

# 部署边界

已按用户授权部署 P2 文字体验版，提供登录、设备管理、房间成员权限、文字发送、离线补发和回执。
设备收信固件与语音对讲仍待后续阶段。
代理目标验证后，由用户在 1Panel 管理域名站点。

## 隔离部署配置

使用 `/opt/pocketlink`、明确的 `deploy/compose.yaml` 文件及 `pocketlink-prod` 项目名。
不要在不明确的目录调用 Compose；修改前检查现有容器、监听端口、资源和代理网络模式。

`POCKETLINK_IMAGE` 使用已发布版本的**镜像摘要**，不使用浮动 edge 标签。
模板见 `deploy/env.example`，其中示例不代表已发布版本。

```bash
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml -f /opt/pocketlink/compose.auth.yaml config --quiet
# 仅在已授权的部署阶段执行：
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml -f /opt/pocketlink/compose.auth.yaml pull
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml -f /opt/pocketlink/compose.auth.yaml up -d --wait
```

容器使用非 root 用户，除专属数据卷和小型 tmpfs 外只读，移除能力、禁止提权，限制日志、
内存、CPU 和进程数，配置就绪检查。默认仅发布到宿主机回环 `127.0.0.1:3002`，先确认未被占用。
认证媒体服务实现前不发布 UDP 端口。不修改其他应用、Docker 守护进程和 1Panel 共享网络。
禁止全局 prune 和 `down -v`。Docker 无法隔离内核、总磁盘耗尽和宿主机故障。

OpenResty 使用 host 网络时，可以代理回环应用端口；若代理运行在 bridge 网络，需另行审查
专属网络连接。应用端口不加入公网安全组。TLS 和 WebSocket Upgrade 由专属代理站点处理。
P2 在根据转发头做安全判断前，必须定义可信代理。

## 备份与回滚

默认模板使用 `pocketlink-prod_data` 命名卷；当前部署使用下文记录的 `data/` 绑定目录保存 SQLite 及 WAL。
不能只复制运行中的主数据库。
迁移前使用经测试的 SQLite 在线备份，或仅停止本服务后备份完整数据卷。
备份独立于在线卷，并按私有消息数据保护。生产前必须实现和演练备份恢复工具。

保留旧镜像摘要和数据库版本。兼容镜像可以直接重新部署；降级数据库需明确的恢复步骤。
恢复快照会丢失快照后的写入，因此不得在用户仍发送消息时静默恢复。
启动迁移保护会拒绝未知的新版本数据库。

私有 GHCR 包拉取可能需要登录。注册表和服务器凭证不得进入 Git 或日志。
GITHUB_TOKEN 发布权限与生产拉取权限不同。CI 不更新生产。

## 当前体验环境（2026-09-20 升级并复核）

- 地址：`https://pocketlink.mcloc.cn`，1Panel 代理到 `127.0.0.1:3002`。
- 应用版本：`ea334044b2326097123437c435132f92cac4d067`。
- 镜像：`ghcr.io/androidmumo/pocketlink-relay@sha256:51be4f9cdadf599739cdcb4c10f8f55f8bcc2e0258830e5e28ea77b7635d5012`。
- 生效配置：`/opt/pocketlink/compose.yaml`、`compose.auth.yaml` 和 `.env`。
- 所有持久化文件位于 `/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/`：
  `data/` 为 SQLite，`secrets/admin_password` 为管理密码，`backups/` 为一致性备份。

服务器 Compose 将 `/data` 改为上述 `data/` 的 bind 挂载，不使用模板中的命名卷。
后续更新不得直接用仓库模板覆盖服务器配置而丢失这一挂载。
私有父目录权限 0700，数据库归 UID/GID 65532，密码文件 0400。
站点必须保留完整反向代理，不能改成静态文件站点来公开持久化文件。

已验证 HTTPS 登录、退出、重启失效、健康检查，以及停止本服务后的数据备份、
在临时目录恢复并执行 SQLite 完整性检查。现有 bing、chat、OpenResty 容器未重启。
初始备份为 `backups/initial-20260911T011224Z.tar.gz`；未配置自动备份任务。
密码可由管理员在 1Panel 文件管理中读取；不提交到 Git，也不打印到部署日志。

文字版本升级前备份为 `backups/before-text-20260911T073934Z.tar.gz`。
2026-09-14 复核通过：新版容器健康、3002 回环端口和数据挂载、线上数据库完整性、
升级前备份恢复检查、HTTPS 文字能力声明、登录及房间/设备列表读取、退出。
现有 bing、chat、OpenResty 的启动时间与升级前一致。
WebSocket 收信、重连补发和回执已通过主机与 CI 测试；真实代理后的设备联调、
完整浏览器交互和真机收信尚未验证。

2026-09-20 已部署签名 OTA 管理。升级前备份 `backups/before-ota-20260920T021947Z.tar.gz` 包含数据及部署配置，已解压验证数据库完整性。迁移 004、HTTPS 登录/固件列表/退出、健康检查通过；固件发布通道为空。其他容器启动时间和重启次数未变。回退至旧文字版本需要同时恢复旧数据库，不能仅更换旧镜像。
