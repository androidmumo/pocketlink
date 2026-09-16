#pragma once
#include <stdint.h>
#include "esp_err.h"
#include "esp_partition.h"
#include "nvs.h"
#include "pocketlink_config.h"
typedef struct { int64_t sequence; int size; char version[33]; char sha256[65]; } pl_ota_offer;
esp_err_t pl_ota_parse(const char *manifest, size_t length, int64_t installed, pl_ota_offer *offer);
esp_err_t pl_ota_download(const pl_config *config, const pl_ota_offer *offer, void (*progress)(int), const esp_partition_t **written);
esp_err_t pl_ota_activate(nvs_handle_t storage, const pl_ota_offer *offer, const esp_partition_t *written);
esp_err_t pl_ota_confirm(nvs_handle_t storage);
int64_t pl_ota_sequence(nvs_handle_t storage);
void pl_ota_boot_failed(void);
bool pl_ota_available(void);

bool pl_ota_partition(const esp_partition_t *partition);
