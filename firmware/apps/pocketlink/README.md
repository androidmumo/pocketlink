简体中文 | [English](README.en.md)

# PocketLink 设备固件（P3 开发版）

用于 AI Passport ESP32-C3。支持二维码热点配网、HTTPS 设备绑定、文字接收和已读回执。
语音、WSS 客户端、TCP/UDP 客户端与 OTA 尚未实现。固件构建和主机测试不能代替真机验收。
操作、协议、存储和刷机注意事项见[设备配网与收信](../../../docs/development/device-provisioning.md)。

与 `board-check` 分开构建；共享固定版本 BSP，不包含上游工作区的未提交代码。
应用和合并镜像保留 `FoloToy-AI-Passport` 名称以复用分区校验器；下载时必须选
`pocketlink-<提交>` 的 CI 附件，不能用 `board-check` 附件替代。
