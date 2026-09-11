简体中文 | [English](authentication.en.md)

# 设备鉴权增量（P2a）

P0/P1 产物已通过 CI。P2a 增加单管理员和独立设备凭证，P2 房间、文字、回执和 WebSocket 已在后续增量实现，见[文字协议](messaging.md)。board-check 固件不变。体验环境已部署，详见 [README](../../README.md)。

## 配置

同时设置以下两项才启用业务接口与嵌入式管理页：

| 配置 | 约定 |
| --- | --- |
| `POCKETLINK_PUBLIC_ORIGIN` | 精确的 HTTPS 来源，如 `https://pocketlink.example`，不带尾斜杠、路径、查询 |
| `POCKETLINK_ADMIN_PASSWORD_FILE` | 挂载的密码文件，一行 16..256 字节，允许结尾 LF/CRLF |

两项都未设置时保留基础模式；只设置一项或格式错误则拒绝启动。密码文件只在启动时读取，
不进入日志或命令参数。使用独立的长随机密码。更改文件后强制重新创建本服务容器，以重新挂载密码文件并清除所有管理会话，命令见 README。

HTTP 后端只能通过回环地址或独立代理网络访问，TLS 在专属反向代理终止，不公开后端端口。
不信任任何转发头。浏览器修改请求必须精确匹配配置的 Origin，即使代理重写 Host 也如此。
不开放 CORS。设备必须校验证书和域名。本次不创建或修改代理、证书、DNS、生产服务器。

`deploy/compose.auth.yaml` 是可选 Compose 覆盖文件。把 `POCKETLINK_ADMIN_SECRET_SOURCE`
设为仓库外、容器 UID 65532 可读的文件，并提供 HTTPS 来源。Compose 只读挂载该文件；
本地绑定式 secret 不映射所有者，因此必须显式设置源文件权限，并保护其父目录。
未来部署前先验证合并配置：

```bash
docker compose --env-file /opt/pocketlink/.env -p pocketlink-prod \
  -f deploy/compose.yaml -f deploy/compose.auth.yaml config --quiet
```

## 接口

JSON 请求需要 `Content-Type: application/json`，请求体最多 4096 字节。
错误只包含稳定的通用错误码，不返回 SQL 或凭证内容。

| 方法/路径 | 凭证 | 请求/结果 |
| --- | --- | --- |
| `POST /api/v1/auth/login` | 密码与精确 Origin | `{password}`，设置会话 Cookie |
| `GET /api/v1/auth/me` | 管理 Cookie | 登录状态 |
| `POST /api/v1/auth/logout` | 管理 Cookie 与 Origin | 注销当前会话 |
| `POST /api/v1/pairings` | 管理 Cookie 与 Origin | `{name}`，返回 `{code, expires_at}` |
| `GET /api/v1/devices` | 管理 Cookie | `{devices}`，包含撤销状态，不含密钥或摘要 |
| `DELETE /api/v1/devices/{id}` | 管理 Cookie 与 Origin | 立即撤销；对已知 ID 可重复调用 |
| `POST /api/v1/device/pair` | 一次性码，无浏览器 Origin | `{code, sn}`，仅当次返回 `{device, credential}` |
| `GET /api/v1/device/me` | `Authorization: Bearer <credential>` | 当前有效的设备身份 |

管理 Cookie 不能作为设备凭证，设备密钥不能执行管理操作。Cookie 为 Secure、HttpOnly、
SameSite=Strict 且只对当前主机有效，12 小时过期。内存中最多保留 32 个会话，重启后需重新登录。
密码校验使用随机启动盐和 600,000 次 PBKDF2-HMAC-SHA256，最多同时执行一次。
登录与设备配对共用每分钟 20 次的全局限额，不依赖可伪造的代理/IP 请求头。该方案针对小型私有服务，
攻击者可能暂时耗尽共享配额。

配对码和设备密钥都包含 256 位随机数，SQLite 只保存 SHA-256 摘要。配对码 10 分钟过期且
在事务中消费。最多 32 个有效配对码、256 台有效设备、1024 个保留的不同序列号。
名称为 1..40 个 Unicode 字符，不含控制字符。SN 只是标识而非秘密，允许 1..64 个 ASCII
字母、数字、点、下划线、冒号、连字符，首字符必须是字母或数字。

不能静默覆盖有效 SN。先撤销再重新配对，保留设备 ID 并替换密钥。若配对响应丢失，已消费的
配对码不能恢复密钥：在列表中撤销该设备并生成新码。管理员需保护页面展示的配对码。
本增量中设备密钥不自动过期，通过撤销和重新配对轮换。任何凭证都不放入 URL。

## 验证及待完成事项

主机测试覆盖来源检查、会话过期和退出、角色隔离、请求大小、并发单次配对、过期、限额、撤销、
重新配对与 SQLite 重开。真实进程测试还检查启用配置和重启行为。页面资源通过嵌入资源 HTTP 测试
和 JavaScript 语法检查；这些不证明浏览器交互、TLS 代理配置、真机存储或端到端配对。
迁移 002 新增设备和配对表，旧镜像拒绝打开新数据库。未来生产升级前须保留一致的升级前备份。
