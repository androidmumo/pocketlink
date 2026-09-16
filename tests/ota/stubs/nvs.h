#pragma once
#include "esp_err.h"
#include <stdint.h>
typedef int nvs_handle_t;
esp_err_t nvs_set_blob(nvs_handle_t,const char *,const void *,size_t);
esp_err_t nvs_get_blob(nvs_handle_t,const char *,void *,size_t *);
esp_err_t nvs_commit(nvs_handle_t);
esp_err_t nvs_erase_key(nvs_handle_t,const char *);
esp_err_t nvs_get_i64(nvs_handle_t,const char *,int64_t *);
esp_err_t nvs_set_i64(nvs_handle_t,const char *,int64_t);
