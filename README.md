简体中文 | [English](README.en.md)

# PocketLink

一个仓库，管理口袋设备固件、中转服务和网页管理端，后续扩展联机游戏与其他应用。

**工作台支持邀请码注册、独立用户空间和跨账号共享房间，保留文字发送、离线补发和逐设备回执。** [账号与邀请教程](docs/development/accounts.md)
新增 **P3 设备开发版**：二维码热点配网、HTTPS 配对、文字收信及签名 OTA，尚待真机验收。
语音尚未实现。`board-check` 仍只是硬件诊断；设备功能请使用 `pocketlink` 固件。
[设备配网与刷机教程](docs/development/device-provisioning.md)

[功能与使用](#使用管理页) · [部署](#首次部署到新服务器) · [维护与升级](#日常维护与升级) · [备份恢复](#备份与恢复) · [常见问题](#常见问题) · [开发](#本地开发与仓库结构)

## 当前已部署的环境

访问 **[PocketLink 管理页](https://pocketlink.mcloc.cn)**。以下是 2026-09-21 升级并复核的部署记录，后续实际配置以服务器文件为准。

| 项目 | 配置 |
| --- | --- |
| 入口 | 1Panel HTTPS 反向代理 → `http://127.0.0.1:3002` |
| 容器 / Compose 项目 | `pocketlink-prod-relay-1` / `pocketlink-prod` |
| 生效配置 | `/opt/pocketlink/compose.yaml`、`compose.auth.yaml`、`.env` |
| 持久化根目录 | `/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/` |
| 数据库 | 上述目录的 `data/relay.db`，以及运行时的 SQLite WAL 文件 |
| 管理员密码 | 上述目录的 `secrets/admin_password` |
| 备份 | 上述目录的 `backups/`；目前没有自动备份任务 |
| 应用版本 | `afe9d363af554d1dc1f2b60cace6e86491cfc660` |
| 资源限制 | 192 MB 内存、0.5 CPU、128 个进程；非 root、只读根文件系统 |

网页已包含在服务端镜像中，无需另外部署前端或安装 MySQL。SQLite 使用专属目录持久化。
现有部署通过了 HTTPS 登录、退出、重启会话失效、备份恢复及数据库完整性检查；尚未验证真机配对。


签名 OTA 管理已于 2026-09-20 部署：可上传、发布、撤回和删除固件包。当前已发布 OTA `0.4.4`（序号 4），删除空收信页重复的配网说明，保留静默后台任务、单行手动加载提示，保留电量显示、配网退出和消息翻页；已完成首次有线安装的设备可在线升级；用户已确认此前 0.4.3 升级及静默/手动加载反馈正常；0.4.4 的真机显示待确认。详见 [OTA 教程](docs/development/firmware-ota.md)。升级前备份为 `backups/before-ota-20260920T021947Z.tar.gz`。

邀请码注册、独立用户空间、共享房间和新版工作台已于 2026-09-21 部署。原有数据归管理员，原密码文件及设备凭证保留。升级前备份为 `backups/before-accounts-20260921T060225Z.tar.gz`；回退旧镜像必须同时恢复匹配的旧数据库。

## 使用管理页

1. 在 **1Panel → 文件** 中打开 `secrets/admin_password` 的完整服务器路径，复制文件中的密码。
   在登录页选择“管理员”使用该密码；普通用户凭管理员邀请码注册后，以用户名和密码登录。没有默认密码。不要把密码提交到 Git 或发送到公开聊天。
2. 用 HTTPS 打开管理页，粘贴密码并登录。登录会话最多保留 12 小时；服务重启后需要重新登录。
3. 输入设备名称并点击“生成配对码”。名称最多 40 个字符，配对码有效期 10 分钟，且只能使用一次。
4. 使用 `pocketlink` 开发固件时，扫码连接设备热点，在本地页面填写 Wi-Fi 和配对码。
   详见[设备配网](docs/development/device-provisioning.md)。**仅生成配对码不会自动添加设备**；
   设备成功联网并提交配对码后才会出现。开发者也可按[鉴权接口](docs/development/authentication.md)联调。
5. 设备配对成功后点击“刷新”。列表中的“有效”表示凭证有效，**不是在线状态**。
6. 丢失设备或需要更换凭证时点击“撤销凭证”。旧密钥立即失效；同一 SN 需要先撤销，再通过原归属账号重新配对；不支持跨账号转移。
7. 使用完点击“退出登录”。若配对响应丢失，无法找回设备密钥，应撤销该设备并生成新配对码。

SN 只是设备标识，不是密码。配对码和设备密钥都是随机凭证，数据库只保存摘要。
最多允许 32 个未过期设备配对码、256 台有效设备、1024 个保留序列号；登录、注册和配对共用每分钟 20 次限额。
目前不提供网页修改/找回密码、设备密钥自动过期或额外管理员账号。

### 房间和文字

创建房间后，从成员列表勾选已配对设备，填写最多 200 字符的正文发送。消息先在服务端落盘，
设备连接后接收；历史记录中的“查看 / 刷新回执”显示待接收、已接收、已读或撤回状态。
新成员不接收旧消息，移出成员或归档房间后停止投递；真机收信使用开发固件，仍待设备验收。
网络异常后同一正文可重试，页面重载后重发前先核对历史。详见[文字接口和 WSS 协议](docs/development/messaging.md)。

## 首次部署到新服务器

**当前服务器已经部署，不要在那里重复执行首次安装步骤。** 以下示例复现当前体验版；
如果使用其他域名或路径，必须同步修改代理、环境配置和挂载目录。

准备 Docker Engine、Docker Compose、Git、Python 3，以及配置了 OpenResty 的 1Panel。
以服务器管理员身份操作，先确认资源足够、3002 未占用，并检查已有服务。

### 1. 准备配置和私有目录

```bash
# 仅用于首次安装；已部署的服务器不要重复执行。
set -e
ss -lnt | grep ':3002 ' || true
free -m
df -h /opt
# 若 3002 已占用，先停止这里的操作并查明占用者，不要结束其他服务。
test ! -e /opt/pocketlink
test ! -e /opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink
git clone https://github.com/androidmumo/pocketlink.git /opt/pocketlink-src
cd /opt/pocketlink-src
git checkout afe9d363af554d1dc1f2b60cace6e86491cfc660
install -d -m 700 /opt/pocketlink
cp deploy/compose.yaml deploy/compose.auth.yaml /opt/pocketlink/
base=/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink
install -d -m 700 "$base" "$base/secrets" "$base/backups"
install -d -m 700 -o 65532 -g 65532 "$base/data"
python3 - <<'PASSWORD'
from pathlib import Path
import os, secrets
p = Path('/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/secrets/admin_password')
with p.open('x') as f:
    f.write(secrets.token_urlsafe(32) + '\n')
os.chown(p, 65532, 65532)
p.chmod(0o400)
PASSWORD
```

在 `/opt/pocketlink/.env` 写入以下内容，设置权限为 `600`。固定镜像摘要来自
[已通过的 CI 构建](https://github.com/androidmumo/pocketlink/actions/runs/35601108984)，不要把浮动 `edge` 当作固定版本。

```dotenv
POCKETLINK_IMAGE=ghcr.io/androidmumo/pocketlink-relay@sha256:556ae4ef97de6ac1634a93ab030afdf72895f78190aad1406fe81aa910543740
POCKETLINK_HTTP_PORT=3002
POCKETLINK_LOG_LEVEL=info
POCKETLINK_PUBLIC_ORIGIN=https://pocketlink.mcloc.cn
POCKETLINK_ADMIN_SECRET_SOURCE=/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/secrets/admin_password
```

编辑 `/opt/pocketlink/compose.yaml`：把服务内的 `- data:/data` 替换成下面的绑定挂载，
并删除文件末尾不再使用的顶层 `volumes: / data:` 定义。保留其余非 root、端口和资源限制。

```yaml
    volumes:
      - type: bind
        source: /opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink/data
        target: /data
        bind:
          create_host_path: false
```

**仓库模板默认使用命名卷，当前站点使用上述 bind 挂载。后续不要用模板直接覆盖服务器文件。**
父目录应保持 `0700`，数据目录归 `65532:65532`，密码文件权限 `0400`。不对整个站点目录递归改权限。
密码通过只读文件挂载进入容器，不写在环境变量值或镜像中。SELinux 开启的其他服务器还需处理
专属目录的容器访问标签，不能通过关闭全局 SELinux 来解决。

### 2. 启动 Docker 服务

定义一个明确限定项目与两份配置的快捷函数；下面后续运维命令都使用它：

```bash
# 后续命令在服务器 Bash 中执行。只管理 PocketLink。
pl() {
  docker compose --env-file /opt/pocketlink/.env -p pocketlink-prod \
    -f /opt/pocketlink/compose.yaml -f /opt/pocketlink/compose.auth.yaml "$@"
}
```

首次启动时执行以下命令；配置反向代理后再检查公网 HTTPS 健康接口：

```bash
pl config --quiet
pl pull
pl up -d --wait --wait-timeout 90
pl ps
curl -fsS http://127.0.0.1:3002/health/ready
```

健康接口应返回 `{"status":"ready"}`。如果 GHCR 拉取返回权限错误，确认镜像可见性；
私有包需要有读取权限的注册表凭证，Git SSH 公钥不等于镜像拉取凭证。

### 3. 配置 1Panel 反向代理与 HTTPS

在“网站 → 创建网站 → 反向代理”中设置：主域名 `pocketlink.mcloc.cn`，代理地址
`http://127.0.0.1:3002`，绑定有效证书并启用 HTTP 跳转 HTTPS。
参见 [1Panel 官方说明](https://1panel.cn/docs/v2/user_manual/websites/website_create/)。

当前 OpenResty 使用 host 网络，所以可以访问回环端口；bridge 网络部署需要单独配置可达的私有网络，
不能假设容器内的 `127.0.0.1` 指向宿主机。不要公开 3002，不需要额外开放 UDP。
`POCKETLINK_PUBLIC_ORIGIN` 必须与浏览器地址完全一致，不带尾部斜杠或路径。

持久化文件位于站点目录下，**该站点必须保持完整反向代理，不得改成公开静态文件的站点**。
上线后检查首页及 `/health/ready`，并确认 `/pocketlink/data/relay.db` 和
`/pocketlink/secrets/admin_password` 均无法下载。当前配置返回 404。

## 日常维护与升级

- 查看状态：`pl ps`；查看最近日志：`pl logs --tail 100 relay`。
- 临时停止本服务：`pl stop relay`；恢复：`pl up -d --wait --wait-timeout 90`。
- 更改管理员密码：在 1Panel 编辑密码文件，保持一行 16..256 字节及原权限，然后执行
  `pl up -d --force-recreate --wait --wait-timeout 90 relay`。重新创建可确保文件被编辑器替换后挂载的是新文件。
  所有管理会话失效，设备凭证保持不变。忘记密码也按此方法设置新密码。
- 更新版本：先阅读更新记录，备份数据、两份 Compose 和 `.env`；记录旧镜像摘要。
  再只修改 `.env` 的 `POCKETLINK_IMAGE` 为目标 CI 输出的摘要，执行
  `pl config --quiet`、`pl pull`、`pl up -d --wait --wait-timeout 90`，最后验证健康和登录。
- 修改 Compose 或 `.env` 后使用 `up -d` 应用配置，单纯 `restart` 不会应用新的容器配置。
- GitHub Actions 负责测试、镜像发布与固件打包，**不会自动部署服务器**。
  推送到 `main` 生成 `edge` 和提交标签，`relay/v*` 标签生成版本镜像。

数据库迁移可能阻止旧镜像启动。回滚前确认数据库兼容性；不兼容时必须恢复匹配版本的备份，
不能只切旧镜像。恢复备份会丢失备份之后的写入，需先停止使用。
不运行全局 Docker 清理、不重启 Docker/1Panel、不修改其他服务；不要执行 `down -v` 或删除数据目录。

## 备份与恢复

文字版本升级前备份为 `backups/before-text-20260911T073934Z.tar.gz`，已恢复至临时目录并通过数据库完整性检查。
当前初始备份为 `backups/initial-20260911T011224Z.tar.gz`，只代表当时的数据库状态，
不是持续备份。数据库运行时可能有 WAL 文件，**不能只复制运行中的 `relay.db`**。

一次手动一致性备份示例：

```bash
# 先定义上面的 pl 函数；备份过程中只有 PocketLink 短暂停机。
(
  set -e
  base=/opt/1panel/www/sites/pocketlink.mcloc.cn/index/pocketlink
  stamp=$(date -u +%Y%m%dT%H%M%SZ)
  umask 077
  pl stop relay
  trap 'pl up -d --wait --wait-timeout 90' EXIT
  tar -czf "$base/backups/data-$stamp.tar.gz" -C "$base" data
  tar -tzf "$base/backups/data-$stamp.tar.gz" >/dev/null
)
```

上面的解包列表检查仅验证归档可读，不等于数据库可恢复。恢复演练应把自己生成的备份解压到
私有临时目录，检查其中数据库的 `PRAGMA quick_check`，并用匹配版本服务验证。
正式恢复时先停止本服务，把现有 `data/` 移到保留目录，再恢复选定备份中的完整 `data/`，
核对所有者 `65532:65532` 和权限，然后强制重新创建容器并验证健康、登录和设备列表。
不要在运行中的数据库目录直接覆盖文件，也不要先删除当前数据。

另行安全备份 `/opt/pocketlink/` 配置和 `secrets/admin_password`，恢复管理员登录时需要它们。
上述归档命令仅包含数据库；备份和密钥需要加密保存到另一台机器。站点下的同盘备份不能防止磁盘故障，
目前没有自动备份或自动清理策略，需自行安排并监控磁盘空间。

## 常见问题

| 现象 | 检查方法 |
| --- | --- |
| 502 | 用 `pl ps` 和本机健康接口确认容器运行，再核对代理的 3002 端口及 OpenResty 网络模式 |
| 首页或管理接口 404 | 未同时设置 HTTPS 来源和密码文件时只启用基础接口；检查容器实际配置 |
| 来源不匹配 / 403 | 使用配置中的精确 HTTPS 域名，检查尾斜杠、端口和代理；不要用 HTTP/IP 地址登录 |
| 401 / 登录失效 | 密码不正确、会话 12 小时过期、退出或重启后需重新登录；撤销后的设备密钥也返回 401 |
| 429 | 登录与配对共享限额，等待约一分钟；也可能达到 256 个总会话或单账号 16 个会话上限，需退出旧会话或等其过期 |
| 409 | 达到配对码、有效设备或保留序列号限额，检查现有记录；过期配对码会在创建新码时清理 |
| 无法读取密码或数据库 | 检查 UID/GID、文件权限和实际挂载；不要给整个站点开放读写权限 |
| 配对码过期或已使用 | 重新生成；若设备已创建但丢失响应，先撤销再配对 |
| 设备没有收信、讲话按钮 | 收信与 OTA 开发固件需首次有线安装及真机验收；音频尚未实现 |

## 本地开发与仓库结构

| 目录 | 用途 |
| --- | --- |
| `services/relay` | Go 服务、鉴权、SQLite、协议解析 |
| `apps/console` | 随服务嵌入的网页管理端，无第三方前端依赖 |
| `firmware/apps/board-check` | 独立 ESP-IDF 硬件基线 |
| `firmware/apps/pocketlink` | 二维码配网、绑定与文字接收开发固件 |
| `firmware/components/pocketlink_config` | 配置校验与可主机测试的消息状态 |
| `firmware/boards/ai_passport` | BSP、许可证与导入来源 |
| `packages/protocol` | 协议文档与共享测试向量 |
| `deploy` | Compose 模板；实际服务器挂载见上文 |
| `tools`、`tests`、`.github/workflows` | 本地校验、测试与 CI |

开发环境需要 Go 1.26.8、Node.js 24、Python 3.10+、C 编译器、actionlint 1.7.12；
固件构建另需 ESP-IDF 5.5.3，安装步骤见[环境准备](docs/development/setup.md)。

```bash
source tools/env.sh
(cd services/relay && go run ./cmd/server)
# 另一个终端检查；不配置鉴权时不会提供管理网页。
curl -fsS http://127.0.0.1:8080/health/ready
./tools/validate.sh --static
# 先激活 ESP-IDF 5.5.3 环境，再构建和校验固件：
./tools/validate.sh --firmware
```

本地服务默认端口是 8080，服务器 Docker 映射端口是 3002。固件合并包在
`dist/firmware/board-check/` 和 `dist/firmware/pocketlink/`，GitHub Actions 分别上传固件及 SHA256 文件。
构建通过不代表真机验证：必须保护 8 MB Flash、3 MB 应用上限，以及 `0x356000` 的设备身份分区，
不要擦除整片 Flash 或把设备身份数据加入发布物。

所有维护文档默认使用简体中文 `.md`，英文为同目录 `.en.md`，页首可互相切换。
新增功能应同步两种语言，不能把计划中的能力写成已完成。

- [架构与阶段计划](docs/architecture/overview.md)
- [设备鉴权及 API](docs/development/authentication.md)
- [协议格式](packages/protocol/README.md)
- [部署记录](docs/operations/deployment.md)
- [硬件约束与来源](firmware/boards/ai_passport/README.md)
- [更新记录](docs/CHANGELOG.md) · [第三方声明](docs/THIRD_PARTY.md)

项目尚未选择整体许可证；导入的 AI Passport 源码遵循附带的 MIT 许可证，依赖各自许可证仍适用。

签名 OTA 源码及使用教程（真机验收待完成）：[OTA](docs/development/firmware-ota.md)

## 邀请码恢复与版本时间

加密密钥自动写入数据库路径加 `.invitation-key` 的文件（默认 `data/relay.db.invitation-key`，0600 权限）。备份和恢复必须包含**整个 data 目录**及匹配的密钥；不要只备份 SQLite。已有密文而密钥缺失时启动会失败，需恢复匹配的备份；不要删除或重新生成密钥。密钥不是管理员密码，不受管理员换密码影响。数据库与密钥一起泄露仍会暴露有效邀请码，二者及备份均按凭证保护。


工作台的房间选择使用自定义下拉列表，支持键盘方向键、Enter 确认和 Escape 关闭。新邀请码可在“邀请码”列表再次查看、复制；旧码仅存哈希，遗失后需撤销重建。版本库分别显示上传时间和最近发布时间（浏览器本地时区）；迁移前的发布时间显示“未记录”，不以上传时间代替。重复发布当前版本保留时间，撤回保留历史时间，重新发布更新最近发布时间。
