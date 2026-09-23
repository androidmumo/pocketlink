简体中文 | [English](README.en.md)

# PocketLink

把 AI Passport 变成口袋里的收信器和对讲机。支持手机扫码配网、文字收信、房间对讲和在线升级；网页端可以管理设备、邀请朋友、发送消息和参与对讲。

## 从这里开始

| 你想做什么 | 打开这里 |
| --- | --- |
| 下载最新固件 | **[最新版下载](https://github.com/androidmumo/pocketlink/releases/latest)** |
| 给设备刷机、配网、收信和对讲 | **[简单教程：设备上手](docs/guides/device.md)** |
| 在自己的服务器部署 Docker | **[简单教程：服务端部署](docs/guides/docker.md)** |
| 查看协议、源码、备份和开发说明 | [技术说明](docs/technical.md) |

已经有 PocketLink 服务地址和账号？直接看设备教程，不需要自己部署服务器。

## 固件该下载哪个？

在 Release 的 **Assets** 中下载：

- **`pocketlink-full.bin`**：第一次安装，用电脑打开[官方刷机页面](https://ai-passport.folotoy.cn/tools/web-flasher/)写入。
- **`pocketlink-ota.bin` + `manifest.json`**：供管理员上传到 PocketLink 的“固件管理”，让已安装设备在线升级。
- `SHA256SUMS`：用于检查下载文件是否完整。

不要下载 GitHub 自动提供的 `Source code` 来刷机，也不要把 OTA 文件当作完整刷机包。
[直接下载完整刷机包](https://github.com/androidmumo/pocketlink/releases/latest/download/pocketlink-full.bin)。仓库内同时保留 [0.5.0 发布快照](firmware/releases/0.5.0/README.md)，最新版本以 Releases 为准。

## 当前能做什么

- 管理员生成注册邀请码、查看和管理用户；朋友注册后拥有独立设备和房间。
- 文字消息离线保留，设备显示发送人、发送时间并支持翻页。
- 房间内按住讲话、松开收听，网页和设备都能参加。
- 电量、北京时间显示，以及签名 OTA 更新。

对讲目前是开发版，构建和模拟音频测试已通过，真机音质、延迟和手机兼容性仍待验收。

[更新记录](docs/CHANGELOG.md) · [架构与计划](docs/architecture/overview.md) · [第三方声明](docs/THIRD_PARTY.md)

项目尚未选择整体许可证；导入源码及依赖遵循各自许可证。
