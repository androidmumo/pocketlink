[English](deployment.md) | 简体中文

# 部署边界

P0/P1 仅产出基础镜像，不得把它作为可用对讲服务公开。生产部署暂不执行。
代理目标验证后，由用户在 1Panel 管理域名站点。

## 后续阶段的隔离部署方式

使用 `/opt/pocketlink`、明确的 `deploy/compose.yaml` 文件及 `pocketlink-prod` 项目名。
不要在不明确的目录调用 Compose；修改前检查现有容器、监听端口、资源和代理网络模式。

`POCKETLINK_IMAGE` 使用已发布版本的**镜像摘要**，不使用浮动 edge 标签。
模板见 `deploy/env.example`，其中示例不代表已发布版本。

```bash
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml config --quiet
# 仅在已授权的部署阶段执行：
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml pull
docker compose --env-file /opt/pocketlink/.env   -p pocketlink-prod -f /opt/pocketlink/compose.yaml up -d --wait
```

容器使用非 root 用户，除专属数据卷和小型 tmpfs 外只读，移除能力、禁止提权，限制日志、
内存、CPU 和进程数，配置就绪检查。默认仅发布到宿主机回环 `127.0.0.1:18080`，先确认未被占用。
认证媒体服务实现前不发布 UDP 端口。不修改其他应用、Docker 守护进程和 1Panel 共享网络。
禁止全局 prune 和 `down -v`。Docker 无法隔离内核、总磁盘耗尽和宿主机故障。

OpenResty 使用 host 网络时，可以代理回环应用端口；若代理运行在 bridge 网络，需另行审查
专属网络连接。应用端口不加入公网安全组。TLS 和未来 WebSocket Upgrade 由专属代理站点处理。
P2 在根据转发头做安全判断前，必须定义可信代理。

## 备份与回滚

`pocketlink-prod_data` 命名卷保存 SQLite 及 WAL。不能只复制运行中的主数据库。
迁移前使用经测试的 SQLite 在线备份，或仅停止本服务后备份完整数据卷。
备份独立于在线卷，并按私有消息数据保护。生产前必须实现和演练备份恢复工具。

保留旧镜像摘要和数据库版本。兼容镜像可以直接重新部署；降级数据库需明确的恢复步骤。
恢复快照会丢失快照后的写入，因此不得在用户仍发送消息时静默恢复。
启动迁移保护会拒绝未知的新版本数据库。

私有 GHCR 包拉取可能需要登录。注册表和服务器凭证不得进入 Git 或日志。
GITHUB_TOKEN 发布权限与生产拉取权限不同。CI 不更新生产。
