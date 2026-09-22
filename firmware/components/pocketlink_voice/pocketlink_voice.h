#pragma once
#include <stdbool.h>
#include <stddef.h>
#include "esp_err.h"
#include "pocketlink_config.h"
esp_err_t pl_voice_start(const pl_config *config,const char *room);
void pl_voice_stop(void);
void pl_voice_hold(bool held); /* Nonblocking, including release while worker is busy. */
void pl_voice_status(char *text,size_t size);
