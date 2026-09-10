[English](README.md) | 简体中文

# AI Passport 硬件基线

导入自 `https://github.com/folotoy/ai-passport`，提交为
`a5c30d342940f70be141b19aa604f67774aec4dc`。原始路径和 SHA256 见
[upstream.json](upstream.json)，MIT 声明见 [LICENSE](LICENSE)。
BSP、演示源码、默认配置和依赖锁均从已提交对象复制，不读取原工作区未提交修改。

`firmware/apps/board-check` 仅用于硬件诊断。CMake 通过 `EXTRA_COMPONENT_DIRS` 引用共享
BSP；保留原始应用及合并镜像名称，因此上游固件校验器不需修改。

硬件为 ESP32-C3、8 MB Flash、无 PSRAM，ST7789P3 240x320 屏幕，三个 ADC 电阻梯按键，
ES8311 音频、CW2017 电量计。引脚和阈值见 `components/bsp/include/bsp_pins.h`。
BSP 无触摸屏接口。LVGL 非线程安全，按键回调不得阻塞，音频使用工作任务。

factory 应用保持在 0x10000，大小 0x300000。受保护 cardid 为 0x356000 的 data/NVS，
大小 0x4000。禁止擦除已配置设备的 Flash。使用分段 `idf.py flash`；仅当合并镜像完整范围
结束于 cardid 之前或设备为空白时才允许整体刷入。本阶段不执行任何刷机。
无线距离、音质和续航等待实物验收。
