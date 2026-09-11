[简体中文](README.md) | English

# AI Passport board baseline

Imported from `https://github.com/folotoy/ai-passport` at commit
`a5c30d342940f70be141b19aa604f67774aec4dc`. See [upstream.json](upstream.json)
for each original path and SHA256, and [LICENSE](LICENSE) for the MIT notice.
BSP, demo source, defaults and dependency lock are copied from committed objects,
not the original workspace's uncommitted changes.

`firmware/apps/board-check` is a diagnostic demo only. CMake locates this shared
BSP through `EXTRA_COMPONENT_DIRS`; original application/merged image names are
retained so the upstream firmware verifier remains unchanged.

Hardware: ESP32-C3, 8 MB Flash, no PSRAM; ST7789P3 240x320 screen; three ADC-ladder
buttons; ES8311 audio; CW2017 battery gauge. Pins and thresholds live in
`components/bsp/include/bsp_pins.h`. There is no BSP touchscreen API.
LVGL is not thread safe; button callbacks must not block. Audio needs worker tasks.

The factory application remains at 0x10000 with size 0x300000. Protected cardid
is data/NVS at 0x356000, size 0x4000. Never erase Flash on a provisioned device.
Use segmented `idf.py flash`; raw merged flashing is allowed only if its full
byte range ends before cardid or the target is blank. No device flashing is part
of this foundation delivery. Radio range, audio quality and battery life remain
unverified until physical acceptance tests.
