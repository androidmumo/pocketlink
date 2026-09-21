简体中文 | [English](deployment.en.md)

# 部署边界

已按用户授权部署邀请注册与共享房间版本，提供独立账号、设备管理、房间成员权限、文字发送、离线补发和回执。
设备收信固件仍待真机验收，语音对讲尚未实现。
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

## 当前体验环境（2026-09-21 升级并复核）

- 地址：`https://pocketlink.mcloc.cn`，1Panel 代理到 `127.0.0.1:3002`。
- 应用版本：`afe9d363af554d1dc1f2b60cace6e86491cfc660`。
- 镜像：`ghcr.io/androidmumo/pocketlink-relay@sha256:556ae4ef97de6ac1634a93ab030afdf72895f78190aad1406fe81aa910543740`。
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
真机收信尚未验证。

2026-09-20 已部署签名 OTA 管理。升级前备份 `backups/before-ota-20260920T021947Z.tar.gz` 包含数据及部署配置，已解压验证数据库完整性。迁移 004、HTTPS 登录/固件列表/退出、健康检查通过；固件发布通道为空。其他容器启动时间和重启次数未变。回退至旧文字版本需要同时恢复旧数据库，不能仅更换旧镜像。

2026-09-21 已部署邀请码注册、账号隔离、共享房间和新版工作台。升级前备份 `backups/before-accounts-20260921T060225Z.tar.gz` 包含数据及部署配置，解压后的 SQLite 完整性检查通过。迁移 005、旧数据归属、原数据数量、HTTPS 管理员登录/账号身份/设备/房间/固件/邀请码列表及退出检查通过；其他容器启动时间和重启次数未变。本地隔离浏览器测试已覆盖注册、两个账号加入/退出房间、发信和回执、桌面与手机布局；未在生产创建测试账号。回退需要恢复升级前数据库，可能丢失备份后的写入，不得静默回退。

2026-09-21 后续部署网页滚动修复：固定文档和导航，内容区域独立滚动；五种桌面/手机/横屏尺寸的浏览器检查通过。升级前备份为 `backups/before-scroll-20260921T064947Z.tar.gz`，未改变数据库结构。线上样式与已验证文件一致，HTTPS 登录/接口/退出及容器健康检查通过，其他容器未重启。

2026-09-21 已部署消息记录区域高度限制，最多 420px 且不超过视口一半，内部滚动。备份 `backups/before-history-20260921T115006Z.tar.gz` 包含数据与配置，恢复到临时目录的完整性检查通过；线上 CSS 与浏览器验证文件一致，HTTPS 登录/接口/退出正常，其他容器未重启。OTA 0.4.2（序号 2）保持发布，用户已确认在线升级、电量显示及加载期间操作锁正常。

## 邀请码恢复与版本时间

加密密钥自动写入数据库路径加 `.invitation-key` 的文件（默认 `data/relay.db.invitation-key`，0600 权限）。备份和恢复必须包含**整个 data 目录**及匹配的密钥；不要只备份 SQLite。已有密文而密钥缺失时启动会失败，需恢复匹配的备份；不要删除或重新生成密钥。密钥不是管理员密码，不受管理员换密码影响。数据库与密钥一起泄露仍会暴露有效邀请码，二者及备份均按凭证保护。


工作台的房间选择使用自定义下拉列表，支持键盘方向键、Enter 确认和 Escape 关闭。新邀请码可在“邀请码”列表再次查看、复制；旧码仅存哈希，遗失后需撤销重建。版本库分别显示上传时间和最近发布时间（浏览器本地时区）；迁移前的发布时间显示“未记录”，不以上传时间代替。重复发布当前版本保留时间，撤回保留历史时间，重新发布更新最近发布时间。

2026-09-21 已部署自定义房间下拉框、邀请码加密恢复和版本发布时间。升级前备份 `backups/before-console-refinements-20260921T125516Z.tar.gz` 解压及 SQLite 完整性检查通过；迁移 006、原数据数量、密钥文件权限、HTTPS 登录与接口检查正常，其他容器启动时间与重启次数未变。生成的单个验证邀请码已在验证原码恢复后撤销，未创建账号或发送消息。线上三份网页资源与本地浏览器验收文件的 SHA256 一致。当前 OTA 0.4.4 已发布；本版设备显示待确认。后续备份必须保留匹配的 `.invitation-key` 文件。
