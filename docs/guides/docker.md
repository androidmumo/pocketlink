简体中文 | [English](docker.en.md)

# 简单教程：用 Docker 部署服务端

**已有可用服务地址的设备用户可以跳过本文。** 网页已包含在镜像中，只需一个容器，不需要另外安装数据库。

准备：已安装 Docker / Compose 的 Linux 服务器、一个解析到服务器的域名，以及 HTTPS 反向代理。以下用 1Panel 举例，需在服务器终端以 root 执行。**已部署的 PocketLink 不要重复初始化**；升级与备份看[运维说明](../operations/deployment.md)。本文不会修改已有生产环境。

## 1. 准备独立目录

把下面第一行换成你自己的域名，再整段执行。若 3002 端口已占用，先换成空闲端口并在第 2、3 步保持一致，不要停止其他服务。

```bash
DOMAIN=pocketlink.example.com
ss -lnt | grep ':3002 ' || true
```

确认端口可用后：

```bash
set -eu
STORAGE="/opt/1panel/www/sites/$DOMAIN/index/pocketlink"
test ! -e /opt/pocketlink
test ! -e "$STORAGE"
install -d -m 700 /opt/pocketlink "$STORAGE" "$STORAGE/secrets" "$STORAGE/backups"
install -d -m 700 -o 65532 -g 65532 "$STORAGE/data"
curl -fL https://raw.githubusercontent.com/androidmumo/pocketlink/main/deploy/compose.quickstart.yaml -o /opt/pocketlink/compose.yaml
python3 - "$STORAGE/secrets/admin_password" <<'PYTHON'
import os, secrets, sys
with open(sys.argv[1], 'x') as f:
    f.write(secrets.token_urlsafe(32) + '\n')
os.chown(sys.argv[1], 65532, 65532)
os.chmod(sys.argv[1], 0o400)
PYTHON
```

密码已生成并保存在 `secrets/admin_password`，没有默认密码。不要将整个存储目录改为公开可读。

## 2. 写配置并启动

继续在同一个终端执行：

```bash
cat > /opt/pocketlink/.env <<ENV
POCKETLINK_IMAGE=ghcr.io/androidmumo/pocketlink-relay@sha256:70f5ba21e2b985414caf92533a7205d000d7f60a7e992ffea7ed2970741f0641
POCKETLINK_PUBLIC_ORIGIN=https://$DOMAIN
POCKETLINK_HTTP_PORT=3002
POCKETLINK_STORAGE_ROOT=$STORAGE
ENV
chmod 600 /opt/pocketlink/.env
docker compose --env-file /opt/pocketlink/.env -p pocketlink-prod -f /opt/pocketlink/compose.yaml config --quiet
docker compose --env-file /opt/pocketlink/.env -p pocketlink-prod -f /opt/pocketlink/compose.yaml up -d --wait
curl -f http://127.0.0.1:3002/health/ready
```

看到容器 healthy，健康接口成功返回，即可继续。上述镜像为已验证版本；以后更新时从发布说明选择新镜像，先备份再升级，不要删除 `data` 目录。

## 3. 在 1Panel 配置 HTTPS

1. 给该域名创建**反向代理网站**，目标为 `http://127.0.0.1:3002`。
2. 申请并启用 HTTPS 证书，启用 WebSocket 转发（对讲需要）。
3. 浏览器访问 `https://你的域名`，看到 PocketLink 登录页即可。

以上代理地址适用于 OpenResty 使用宿主机网络的部署。如果代理在隔离容器网络中，容器内的 `127.0.0.1` 不是宿主机，请先按自己的 1Panel 网络配置处理可达地址，不能直接改成公网暴露 3002。

设备连接的也是域名和 HTTPS 端口 **443**。3002 仅供服务器上的反向代理访问，无需在阿里云安全组开放。安全组需允许网站的 80/443（证书和 HTTPS）。

存储虽然放在站点目录内，**网站必须是反向代理，不要把这个目录作为静态网站公开**。

## 4. 登录和添加设备

在 1Panel 文件管理中打开 `/opt/1panel/www/sites/你的域名/index/pocketlink/secrets/admin_password`，复制其中密码；网页登录选择“管理员”。

- 自己使用：在“我的设备”生成配对码，按[设备教程](device.md)完成绑定。
- 邀请朋友：在“注册邀请码”生成邀请码，朋友注册后管理自己的设备。
- 发消息/对讲：在“房间与消息”创建房间并勾选设备。
- 发布 OTA：从 [Releases](https://github.com/androidmumo/pocketlink/releases) 下载同版 `manifest.json` 和 `pocketlink-ota.bin`，在“固件管理”上传并发布。

**重启、更新前请备份完整 `data` 目录及 `.env`、Compose 配置和密码文件。** `data` 中的数据库和 `.invitation-key` 必须成套保留。备份时停本项目容器或使用 SQLite 一致性备份，不要直接复制正在写入的数据库单文件。具体命令见[备份与恢复](../technical.md#备份与恢复)。

[返回首页](../../README.md) · [技术说明](../technical.md)
