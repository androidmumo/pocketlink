English | [简体中文](README.zh_CN.md)

# Console implementation boundary

The web application starts in P2. There is no UI bundle or login page yet.
Planned flows: administrator login, one-time device pairing, room membership,
plain-text sending, per-device delivery/read receipts and bounded history paging.
The build will embed static assets in the relay image so the first deployment
runs one application container. Do not add mock authentication to the P1 service.
