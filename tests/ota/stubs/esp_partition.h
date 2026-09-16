#pragma once
#include <stdint.h>
typedef struct { uint32_t address,size; int type,subtype; } esp_partition_t;
#define ESP_PARTITION_TYPE_APP 0
#define ESP_PARTITION_SUBTYPE_APP_OTA_0 16
#define ESP_PARTITION_SUBTYPE_APP_OTA_1 17
