#pragma once
#include <stdbool.h>
#include <stdint.h>
#define PL_TEXT_LIMIT 2048
typedef enum { PL_EMPTY, PL_RECEIVED_PENDING, PL_RECEIVED, PL_READ_PENDING } pl_receipt;
typedef struct { int64_t id; char text[PL_TEXT_LIMIT + 1]; uint8_t receipt; } pl_inbox;
bool pl_inbox_valid(const pl_inbox *inbox);
bool pl_inbox_accept(pl_inbox *inbox, int64_t id, const char *text);
bool pl_inbox_read(pl_inbox *inbox);
void pl_inbox_acknowledged(pl_inbox *inbox);
