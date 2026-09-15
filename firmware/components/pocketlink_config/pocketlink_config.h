#pragma once
#include <stdbool.h>
#include <stdint.h>
#define PL_CONFIG_VERSION 1
/* One versioned NVS blob: never persist partially updated credentials. */
typedef struct {
    uint32_t version;
    char ssid[33];
    char password[65];
    char host[254];
    uint16_t port;
    char credential[65];
} pl_config;
bool pl_host_valid(const char *host);
bool pl_secret_valid(const char *secret);
bool pl_config_valid(const pl_config *config);
bool pl_utf8_valid(const char *text);
bool pl_json_shallow(const char *text);
