简体中文 | [English](releases.en.md)

# 固件 Releases 维护

用户下载入口：[GitHub Releases](https://github.com/androidmumo/pocketlink/releases)。每次发布包含四个文件：`pocketlink-full.bin`（USB 写入 0x0）、`pocketlink-ota.bin`、`manifest.json`、`SHA256SUMS`。不要把源码归档或 CI 临时 Artifact 当成面向用户的下载入口。

## 当前发布

0.5.0 的原始文件在 `firmware/releases/0.5.0/`，源码为 `source.txt` 指定的 `78a5348`；签名 OTA 与线上版本相同，不重签。标签 `firmware/pocketlink/v0.5.0` 指向包含发布配置的提交，Release 正文另列真正的固件源码提交，二者不要混淆。

## 后续新固件

在 GitHub Actions → **Signed firmware package** → Run workflow，选择 main，填写新版本及全新递增序号。流程构建完整包和 OTA，运行主机测试，使用已有 Secret 签名，然后自动创建 Release。它不部署 Docker、不切换服务端 OTA 通道，也不让设备自动安装。管理员仍需下载签名包，在服务端“固件管理”上传并发布。

只修改文档时不要运行签名流程。同一版本或序号不能重复用于不同固件；既有版本禁止覆盖。Release 的四个附件优先于仓库内保留的 0.5.0 历史快照，不把每次构建产物写回主分支。

对于已验证、已签名且必须保持字节一致的包，可在 `firmware/releases/<版本>/` 放入四个文件和 `source.txt`，本地执行 `python3 tools/check-release.py firmware/releases/<版本>`，提交并推送 `firmware/pocketlink/v<版本>` 标签，由 **Publish verified firmware bundle** 发布。本次 0.5.0 采用此路径。

## 权限及失败处理

经用户明确授权，仅两个发布 job 使用 `contents: write`；默认 workflow 权限仍为只读。流程验证签名、SHA256、完整包中的应用与 OTA 一致、分区保护和标签版本。先创建草稿，四个附件全部上传完成才公开。已存在的 Release 不覆盖；上传失败可能留下草稿，维护者应先检查缺失文件与失败原因，再明确处理草稿，不要盲目重跑或删除已发布版本。

当前 Release 标题和正文标注“开发版”，供用户获取最新可下载包；真机验收边界以具体版本说明为准。GitHub Releases 发布不等于真机安装成功。
