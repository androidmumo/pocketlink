[简体中文](device-provisioning.md) | English

# QR provisioning and device text reception

This page describes the P3 development firmware in `firmware/apps/pocketlink`. It works
with the deployed P2 text server without a server upgrade or additional public ports.
The flows below are implemented in source; mobile compatibility, peak memory, physical
power-loss behavior and two-device communication still require the acceptance checks
at the end. A successful build is not a guarantee of everyday device readiness.

## Usage

1. While your phone has internet access, log into the console, generate a pairing code
   and copy it. The code has 43 characters, expires after ten minutes and works once.
2. On first boot, the device displays a Wi-Fi QR code for `PocketLink-XXXXXX`. Scan it
   with the phone's system camera and confirm joining. Keep the connection when the
   phone warns that it has no internet. Camera support varies; WeChat scanning is not guaranteed.
3. The device attempts captive-portal discovery. If no page opens, press OK to switch
   to the webpage QR code, or open `http://192.168.4.1` manually. OK switches between
   the two QR codes. The screen also shows the hotspot name and password.
4. Scan and select a **2.4 GHz** network, or enter a hidden SSID manually. SSIDs use
   1–32 UTF-8 bytes; a Chinese character normally takes three. Leave the password
   empty for open networks, otherwise use 8–63 bytes. Enterprise Wi-Fi, WPA1-only,
   WPA3-only and 64-character hexadecimal PSKs are unsupported.
5. The default server is `pocketlink.mcloc.cn`, port **443**. This is the public HTTPS
   port, not the host's internal 3002. Use a hostname or IPv4 address, without a URL
   path. IPv6 and plain HTTP are unsupported. The server needs a valid TLS certificate.
6. Paste the pairing code and submit. The device joins Wi-Fi, synchronizes its clock
   over SNTP, validates HTTPS, pairs and saves configuration. A hotspot channel switch
   can briefly disconnect the phone; reconnect to inspect status rather than repeatedly
   generating new codes.
7. After the screen confirms the configuration was saved, the hotspot and local web
   server close after approximately four seconds. The phone can return to its normal network.
8. Refresh the console, add the device to a room and send text. The device checks for
   messages approximately every five seconds. Up/down turn pages; OK marks the current
   message read. After the read receipt is confirmed, the next message can arrive.

Wi-Fi passwords never go to the relay. The page does not store configuration in
localStorage. Page assets live on the device: no CDN or phone app is required.
The firmware does not read or overwrite `cardid`; it uses `PL-<Wi-Fi MAC>` as a public
SN. The SN is not an authentication credential.

The portal supports `http://192.168.4.1` and explicit default port `http://192.168.4.1:80`. If older firmware rejects access despite a hotspot connection, install firmware containing the dual-stack address validation fix. Do not disable phone IPv6 or remove access validation.

## Later boots and reconfiguration

- Configured devices reconnect automatically. Temporary network failures do not open a hotspot.
- Hold OK to provision again with a fresh random hotspot password and page session token.
  Reception pauses during provisioning.
- The provisioning window lasts ten minutes. Timeout closes the hotspot and preserves
  saved configuration. Hold OK again to start another window.
- For a Wi-Fi-only change, keep the server address/port and leave the pairing code empty
  to retain the credential and unread message.
- Changing servers requires a new pairing code. Explicit re-pairing replaces the credential
  and clears the old binding's local message. Revoke an existing active SN in the console
  before pairing it again with the same server.
- Failed validation does not replace saved configuration. Reboot attempts the old settings;
  an unsuccessful first configuration remains unpaired.
- A lost successful pairing response or a local save failure can leave the server paired
  while the device has no saved credential. Do not assume the code is reusable: revoke
  that device in the console, generate a fresh code and retry.
- There is no automatic factory reset. Storage initialization/version failures stop
  operation without erasing any partition.

## Delivery and receipts

The first device client polls `/api/v1/device/inbox` and `/api/v1/device/ack` over HTTPS,
without using the server's available WSS channel or adding a new network dependency.
Successful requests are approximately five seconds apart; TLS, weak networks and
reconnections can add latency.

The device durably retains **one current message**. Other pending messages stay on the
server. It atomically saves a received message before sending `received`. Only pressing
OK persists a pending-read state and sends `read`. State remains retryable until the
server confirms success. Duplicate receipts are idempotent: power loss may cause retries,
but merely reading HTTP data never fabricates a read receipt. Confirmed read messages
are removed locally. There is no device history list; history remains in the console.

A 404 receipt response after room permission withdrawal removes the current local
message. Already downloaded content cannot be instantly erased while offline or while
remaining on an already received message. Revoked credentials require re-pairing and
cannot receive new messages. Console credential validity is still not live online status.


The font covers ASCII, 6,763 common GB2312 Han characters and all device UI text. Rare characters, some Traditional Chinese characters and emoji may remain unsupported. Host checks reject UI text with missing glyphs.

## Storage and security boundaries

| Region | Address / size | Purpose |
| --- | --- | --- |
| ota_0 | `0x10000` / `0x300000` | First application slot, preserving the 3 MB limit |
| otadata | `0x310000` / `0x2000` | OTA boot selection |
| cardid | `0x356000` / `0x4000` | Existing identity partition; never overwrite |
| pocketcfg | `0x35a000` / `0x10000` | Dedicated NVS configuration, current message and update state |
| ota_1 | `0x370000` / `0x300000` | Second application slot |

Configuration and inbox are one versioned blob: a binding and its message are not saved
as two independent updates. The Wi-Fi driver uses RAM storage rather than its default
Wi-Fi NVS credential storage. Flash/NVS encryption and Secure Boot are not enabled;
physical Flash access can expose local credentials and messages. Do not claim encrypted storage.

The WPA2 hotspot uses a random 16-hex-character password and allows one phone connection.
POST requests require the local interface, exact Host/Origin and a random session token.
Request size, JSON depth and queues are bounded. The local HTTP server only runs during
provisioning. WPA2 protects the local link; this local page is not HTTPS. Relay requests
always validate certificates and hostname, reject redirects and offer no certificate-bypass
option. Clock synchronization uses `ntp.aliyun.com` / `pool.ntp.org`; failure never bypasses
certificate checks. Avoid a home LAN using the hotspot's `192.168.4.0/24` subnet because of routing conflicts.

## Building and flashing

```bash
source /path/to/esp-idf-v5.5.3/export.sh
./tools/validate.sh --static
./tools/validate.sh --firmware
```

Both firmware applications are built and verified, with outputs under
`dist/firmware/board-check/` and `dist/firmware/pocketlink/`. CI uploads separate artifacts
and SHA256 files. To build this application directly:

```bash
cd firmware/apps/pocketlink
idf.py build
idf.py -p /dev/cu.YOUR_DEVICE flash monitor
```

Before flashing, verify the port, ESP32-C3/8 MB target, partition table and current backup.
Initial OTA migration writes the bootloader, partition table, application and blank otadata,
not `cardid` or `pocketcfg`. **Never use `erase-flash` or write files to protected partitions.**
A merged image is suitable for a provisioned device only when its entire written range
ends before `0x356000`. The build verifier permits padding in the protected region, but
writing that padding would still erase existing identity data; this does not authorize
flashing a larger image across the protected partition. Initial wired installation, hard-reset boot and identity readback comparison passed on one device on 2026-09-20. Phone provisioning, reception and OTA still require acceptance. After flashing, verify reset into the application before device tests.

## Device acceptance checklist

- iPhone/Android camera scanning, automatic portal opening and alternate QR/manual URL.
- Wrong password, hidden/missing SSID, unavailable clock service, invalid TLS certificate,
  unreachable server, expired code and revoked credential.
- Ten-minute timeout, repeated provisioning entry, phone departure, repeated submit and
  hotspot shutdown after successful provisioning.
- Wi-Fi changes retaining binding; server changes requiring pairing; refresh revealing
  neither Wi-Fi password nor device credential.
- Two devices joining rooms, Chinese text paging, received/read status, network loss
  and power loss while receiving or acknowledging.
- Minimum available heap, task stack margins, prolonged reconnects, watchdog and display
  stability with Wi-Fi/TLS/QR active together.

Host tests cover configuration boundaries, base64url credentials, UTF-8, JSON depth, malformed DNS,
inbox state recovery and page slow responses, duplicate submission and error recovery.
Actual radio, button, display and physical power-loss behavior still need device acceptance.

Signed OTA source and usage (device acceptance pending): [OTA](firmware-ota.en.md)

## Cancel provisioning and read pages

Long-press OK again to leave provisioning, retain saved settings and reconnect to the saved network. Buttons are locked while a phone-submitted connection attempt is running; wait for it to finish, then long-press OK again to exit. Up/down now switch fixed text pages without scrolling or animation. Single-page messages do not move; multi-page messages show their page number. Receipt updates preserve the reading position. Verify cancellation after a failed Wi-Fi attempt and short, long, mixed-language and newline-heavy messages on hardware.

## Bluetooth compatibility

ESP32-C3 supports BLE, but this firmware does not enable Bluetooth provisioning yet. Android Chrome supports Web Bluetooth from HTTPS pages; common iPhone browsers do not offer equivalent support, so browser Bluetooth cannot be the sole entry point for both platforms. Hotspot provisioning remains available; a shared BLE experience needs a separate BLE-capable app or client. See [Chrome Web Bluetooth](https://developer.chrome.com/docs/capabilities/bluetooth).

A future logged-in client can generate and transfer a single-use binding credential automatically, removing manual pairing-code entry. Account ownership verification must remain; SN alone must not authorize binding. Existing Wi-Fi-only reconfiguration already permits an empty pairing code while retaining the device binding.

## Battery and operation feedback

The top-right CW2017 fuel-gauge percentage is sampled at startup and approximately every 30 seconds in the idle worker loop; long network tasks can defer sampling. Values at or below 20% are red; failed reads show --%. The driver has no charging-status API, so no charging state is shown.

Only manual provisioning, read-receipt submission and OTA checks/installation show a single-line wait message with a trailing spinner. The LVGL timer runs independently of network work; manual actions reserve input before queuing and release it on success or failure. Page turns and QR switching do not show loading. Background polling, battery reads, reconnects and update checks remain quiet and do not claim the interaction lock. Keys are accepted, but the single worker completes its current network request before processing them. Keys whose message/view context changed while waiting are discarded to avoid acting on new content. On 2026-09-21 the user confirmed battery display and loading/input-lock behavior in 0.4.2. Gauge accuracy and recovery under fault conditions still require dedicated acceptance.
