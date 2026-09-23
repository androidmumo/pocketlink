[简体中文](device.md) | English

# Simple guide: flash and use your device

You need an AI Passport (ESP32-C3, 8 MB Flash), a USB data cable and desktop Chrome or Edge. Flash on a computer; provision Wi-Fi with a phone.

## 1. Download

Open [PocketLink Releases](https://github.com/androidmumo/pocketlink/releases), expand **Assets** and download **`pocketlink-full.bin`**. No development environment is needed.

First installation uses USB below. Already running PocketLink? Skip to [online upgrades](#online-upgrades). Back up important device data first using the [technical instructions](../development/device-provisioning.en.md); another application's settings may not apply after switching firmware.

## 2. Use the official web flasher

1. Connect the device using a USB data cable. Close software using its serial port.
2. Open the [official web flasher](https://ai-passport.folotoy.cn/tools/web-flasher/) in desktop Chrome or Edge; switch the site's language if needed.
3. Choose **Install local firmware**, then **Connect device**, and select the correct serial port.
4. Choose **Select .bin files**, select only `pocketlink-full.bin`, and check that its address is **`0x0`**.
5. Choose **Write firmware**, keep USB connected and wait for completion. Restart as instructed; reconnect power if necessary.
6. Continue when PocketLink or its Wi-Fi QR screen appears.

**Do not choose Clear device data or erase the entire flash.** The full image contains no device identity region; the official writer currently writes without a full-chip erase. If the page reports incompatibility, check the file instead of erasing. If no port appears, try another data cable/USB port and consult the [official guides](https://ai-passport.folotoy.cn/guides/).

## 3. Connect Wi-Fi

1. Open your PocketLink server. Ask its administrator for a **registration invitation** if you need an account. Administrators select the administrator login mode.
2. Under My devices, enter a name and generate a **device pairing code**, valid for ten minutes. This differs from registration and room invitations.
3. Scan the device QR code and join its hotspot. Stay connected when your phone warns that the hotspot has no Internet.
4. Open **`http://192.168.4.1`** in the phone browser. Enter your home Wi-Fi credentials, server host/port and pairing code.
5. Use a domain such as `pocketlink.example.com`; the HTTPS port is usually **443**, not the Docker mapping port 3002.
6. Save and wait for connection. Refresh the server device list, then reconnect your phone to its normal Wi-Fi.

Hold OK on the inbox to enter provisioning; hold OK again to cancel and restore saved networking. Do not disclose Wi-Fi passwords or pairing codes.

## 4. Messages and intercom

In Rooms and messages, choose Create room and select your device under Receiving devices. Send text from the console. Registered friends can use Join room with a **room invitation** you create.

| Device action | Control |
| --- | --- |
| Message pages | Tap Up / Down |
| Mark read and continue receiving | Tap OK |
| Enter / cancel provisioning | Hold OK on the inbox |
| Check updates | Hold Up on the inbox |
| Enter intercom | Hold Down on the inbox |
| Switch voice rooms | Tap Up / Down in intercom |
| Speak / listen | Hold / release OK in intercom |
| Exit intercom | Hold Down in intercom |

In the browser, select a room, Connect intercom, allow the microphone and hold the talk button. One speaker per room at a time. Device text polling pauses during intercom and resumes after exit. Keep endpoints apart to prevent acoustic feedback.

## Online upgrades

An administrator uploads `manifest.json` and `pocketlink-ota.bin` from the same Release under Firmware management, then publishes it. Skip uploading if your managed server already has the release.

With the device online and on stable power, **hold Up** on the inbox to check, then **hold Down** to confirm installation. Wait for automatic restart. Updates are not forced. Never upload `pocketlink-full.bin` to OTA management.

Physical installation, audio quality and latency for 0.5.0 remain unverified. Older retained messages may lack sender/time metadata; verify with a newly sent message.

[Home](../../README.en.md) · [Deploy a server](docker.en.md) · [Technical reference](../technical.en.md)
