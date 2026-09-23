简体中文 | [English](README.en.md)

# PocketLink 0.5.0 固件

优先从 [GitHub Releases](https://github.com/androidmumo/pocketlink/releases) 下载。这里保留同版发布文件，便于仓库内直接获取；没有私钥、设备身份或用户配置。

- [pocketlink-full.bin](pocketlink-full.bin)：首次 USB 刷机，地址 `0x0`，不要整片擦除。
- [pocketlink-ota.bin](pocketlink-ota.bin) 和 [manifest.json](manifest.json)：网页 OTA 管理上传包。
- [SHA256SUMS](SHA256SUMS)：SHA256 校验。

源码提交 `78a53484c3ac57fe986276989bae8332e695950f`，序号 5。OTA 与 2026-09-22 已发布包完全相同，不重复签名；完整包中的应用与 OTA 文件逐字节一致。构建和主机测试通过，0.5.0 真机验收待完成。

[刷机教程](../../../docs/guides/device.md)
