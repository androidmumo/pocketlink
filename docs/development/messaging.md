简体中文 | [English](messaging.en.md)

# 房间、可靠文字和回执（P2）

管理页现在可以创建房间、添加或移除已配对设备、发送文字、翻页查看历史和逐设备回执。
设备通过 WSS 接收，HTTPS 收件箱作为调试/轮询入口。对讲固件和音频仍未实现。

## 投递与权限

- 管理员是当前唯一文字发送者；设备仅能获取分配给自己的消息和更新自己的回执。
- 新消息和发送时的有效成员名单在同一 SQLite 事务落盘。新加入成员不获得历史消息。
- 请求携带 `request_id`，在房间内唯一。同一标识和正文重试返回原消息；更换正文返回 409。
  网络失败时网页保留同一正文的重试标识，但页面重载会丢失它，重发前先查历史。
- 服务端采用至少一次投递。设备必须按消息 `id` 去重，先持久化再发送 `received`，
  实际显示给用户后发送 `read`。回执保存在数据库；重复或迟到的 `received` 不会覆盖已读状态。
- 仅写入 WebSocket 不算已接收。未确认消息在重连后补发；已确认消息不再从服务端自动重播。
  因此设备不能在只放入易失内存时就确认。
- 移除成员、归档房间或撤销设备会撤回投递权限。已进入网络或设备的内容无法收回。
  重新添加成员不会恢复旧消息；重新配对保留设备 ID，但必须重新授权房间。
- 归档房间不能继续发送或添加成员，历史和回执保留。已经落盘的幂等请求仍可查询原结果。

## 管理接口

沿用管理员 Cookie、来源验证和 JSON 请求规则。时间戳为 Unix 秒，消息 ID 为递增整数。

| 方法与路径 | 请求 / 响应 |
| --- | --- |
| `POST /api/v1/rooms` | `{name}` → 房间对象 |
| `GET /api/v1/rooms` | `{rooms}`，包含已归档房间 |
| `DELETE /api/v1/rooms/{room}` | 归档并撤回所有投递权限 |
| `GET /api/v1/rooms/{room}/members` | `{device_ids}` |
| `PUT /api/v1/rooms/{room}/members/{device}` | 添加有效设备，无请求体 |
| `DELETE /api/v1/rooms/{room}/members/{device}` | 移除成员，无请求体 |
| `POST /api/v1/rooms/{room}/messages` | `{request_id,text}` → `{id,room_id,request_id,text,created_at}` |
| `GET /api/v1/rooms/{room}/messages?before={id}` | `{messages,next_before}`，最新在前，每页最多 20 条，0 表示结束 |
| `GET /api/v1/messages/{id}/receipts` | `{receipts}`，逐设备的 `delivered_at`、`read_at`、`withdrawn_at`，未发生为 null |

请求标识为 1..64 个协议标识字符，可用 UUID；文字为 1..200 Unicode 码点且最多 2048 UTF-8 字节，
不接受纯空白或控制字符（换行和 Tab 除外）。空房间拒绝发送，返回 `empty_room`。
当前一个数据库最多保留 64 个房间（含归档）和 10000 条消息；达到上限返回 409，不静默删除旧数据。
房间及消息管理修改请求合计最多 120 次/分钟。网页历史和回执按按钮刷新，不伪装成实时在线状态。

## 设备 HTTPS 接口

只接受 `Authorization: Bearer <credential>`，不能用管理 Cookie；拒绝携带浏览器 Origin 的请求。
凭证必须放在请求头，不能放 URL。所有设备网络连接都必须验证 TLS 证书和域名。

- `GET /api/v1/device/inbox`：最多 20 条最早的未确认消息，无可跳过消息的客户端游标。
- `POST /api/v1/device/ack`：`{message_id,state}`，state 为 `received` 或 `read`。
  伪造他人回执或操作已撤回消息均失败。

## 设备 WSS 协议

连接 `wss://pocketlink.mcloc.cn/api/v1/device/stream`，请求头包含 Bearer 凭证，
子协议为 `pocketlink.v1`。不支持查询参数或压缩。最多 64 个连接，每台设备只允许一个，
重复连接返回 409；异常断线后应退避重试。握手前后都检查容量，凭证持续重新校验。

服务端每秒检查待发队列。每个连接最多一条未确认消息，5 秒未确认会重发。
慢客户端写超时为 5 秒；每 20 秒 Ping、5 秒超时关闭。撤销凭证会关闭旧连接。
停机时关闭并等待所有 WebSocket 处理退出后再关闭数据库。

服务器发送（沿用 [v1 控制信封](../../packages/protocol/README.md)）：

```json
{"version":1,"type":"text.message","request_id":"send-001","namespace":"text","room_id":"r123","payload":{"id":1,"room_id":"r123","request_id":"send-001","text":"你好","created_at":1789100000}}
```

设备确认：

```json
{"version":1,"type":"text.ack","request_id":"ack-001","namespace":"text","payload":{"message_id":1,"state":"received"}}
```

设备确认不带 `room_id`，服务端从消息和认证凭证查询权限；设备不能自报发送者或接收者。
落盘后返回 `ack.result`，回显确认的 `request_id`，payload 为 `{message_id,acknowledged:true}`。
确认结果丢失可以重复发送。同一连接最多每分钟 120 个应用确认帧，超过则关闭。
只接受最多 8192 字节的文本控制帧；二进制、未知类型、非法信封和非法确认均关闭连接。

## 运维边界

迁移 003 新增房间、成员、消息和回执。旧 P2a 镜像拒绝打开此数据库，升级前必须做一致性备份。
当前没有自动过期、清理或大附件功能。端口仍是 `127.0.0.1:3002`，数据挂载目录不变。
1Panel 需保留 WebSocket Upgrade 和 Connection 转发；WSS 复用 HTTPS 443，不开放 UDP。
主机集成测试使用模拟设备覆盖断线重放、跨房间隔离、单调确认、撤销与停机，不能代替真机存储/收信验证。
