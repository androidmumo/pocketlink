[简体中文](voice.md) | English

# Room intercom (P4 development release)

The first release provides half-duplex PTT: one speaker per room, hold to talk and release to listen. Device/device, browser/device and browser/browser connections are supported. Audio is forwarded live without SQLite storage, offline delivery or recording playback. Text remains independently persistent.

## Usage

1. Administrators issue registration invitations through the administrator-only Registration invitations page.
2. The top action area in Rooms and messages contains Create room, Join room and Invite friends. Room codes are generated, viewed, copied and revoked inside the selected room's invitation panel; they cannot register accounts.
3. Select your devices under Receiving devices. A device can only enter active rooms it belongs to and its owner can access. Removing a device/user, archiving a room or revoking credentials closes voice access within approximately one second.
4. On the device inbox, long-Down enters intercom. Short Up/Down switches rooms; hold OK to speak, release to listen, long-Down to exit. Device text polling and OTA checks pause during intercom and resume afterward. Provisioning and upgrades remain on the inbox page.
5. On the web, select a room, click Connect intercom and grant microphone permission. Hold the talk button, or focus it and hold Space/Enter. Release stops transmission. Disconnecting, changing rooms, navigating away or backgrounding the page releases the microphone/connection; reconnect when returning.
6. Clients stop after 29 seconds per press; the server lease lasts at most 30 seconds. Busy, timeout and lease expiration require releasing and pressing again; no automatic queue or delayed surprise transmission.

The browser needs HTTPS, WebSocket, microphone and AudioWorklet support. Permission denial produces a recoverable message. Keep test endpoints apart to avoid one microphone recapturing another speaker. Device volume defaults to 55%. Mobile page zoom is disabled as requested; actual phone-browser acceptance remains necessary.

## Message information and clock

The device displays server `sender` (account name) and `created_at` (send time), never substituting receive time. The top bar shows `HH:mm` using SNTP and fixed Beijing time UTC+8, or `--:--` before synchronization. Message timestamps have minute precision. Message bodies use six lines per page with fixed paging.

The original `state` NVS blob ABI remains unchanged. Separate `inbox_meta` data is matched by message ID; upgrades preserve Wi-Fi, credentials, inbox and the identity region. Previously stored messages lack metadata and display unknown values; newly received messages include sender and timestamp. Display time conversion does not change stored server UTC timestamps.

## Endpoints and framing

- `GET /api/v1/device/rooms`: bearer-authenticated active device rooms.
- `WSS /api/v1/device/voice/{room_id}`: device bearer credential, no browser Origin.
- `WSS /api/v1/voice/{room_id}`: browser session cookie and exact configured Origin.
- Require subprotocol `pocketlink.voice.v1`; reject query parameters, including credentials.

Bounded JSON control: clients send `{"type":"request","request_id":1}`. The server replies with `grant` or `busy` referencing the request. Grants include `stream` and `lease_ms`; members receive `{"type":"floor","stream":1,"sender":"account or device name"}`. Release uses `{"type":"release","stream":1}`. Idle floor uses stream 0. A grant arriving after release is immediately surrendered without starting capture.

Each binary frame is exactly 648 bytes: bytes 0–3 are a big-endian uint32 stream generation; 4–7 a big-endian uint32 sequence starting at 1; 8–647 contain 320 little-endian signed 16-bit PCM samples. Format is 16 kHz mono, 20ms per frame, approximately 256 kbit/s before transport overhead. The server verifies ownership, generation, sequence and rate; sender identity is server-derived.

Limits: 32 voice connections, one per browser login session or device, sharing a global 64-stream cap with text. Outbound queues contain at most 16 frames; device receive and browser capture queues at most eight. Server/device audio frames older than 160ms in local queues are dropped; send backlog disconnects clients. Limit 60 audio frames and ten controls per second. Permissions/sessions are rechecked every second; WebSocket ping/pong keeps idle connections alive.

Use the existing HTTPS/WSS port and 1Panel proxy, without another public port. TCP head-of-line blocking and weak-network latency remain possible. DTLS/UDP, Opus/ADPCM, persistent audio, end-to-end encryption and mixing are not implemented.

## Validation and dependencies

Host tests cover contention, nonholder rejection, room isolation, replay, expiry, slow peers, handshake boundaries, revocation, browser 44.1/48 kHz conversion and bounded capture queues. The independent firmware PTT state machine covers early release, stale grants, timeouts and re-press requirements.

`tests/console/voice-browser.cjs` only accepts disposable loopback HTTPS servers. It uses synthetic microphone/device endpoints to verify bidirectional PCM, contention, release and navigation cleanup. Use the same `POCKETLINK_PLAYWRIGHT` and `POCKETLINK_TEST_PASSWORD_FILE` variables as `tests/console/browser.cjs`; the test proxy must support WebSocket Upgrade.

Firmware pins [Espressif esp_websocket_client 1.6.1](https://components.espressif.com/components/espressif/esp_websocket_client/versions/1.6.1/readme) with certificate-bundle validation, retaining LVGL 9.5.0. Builds and synthetic audio do not validate microphone/speaker quality, latency, real phone compatibility, two-device contention, network recovery or runtime peak memory.
