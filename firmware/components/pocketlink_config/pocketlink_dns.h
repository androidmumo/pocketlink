#pragma once
#include <stddef.h>
#include <stdint.h>
/* Return response size, or zero for unsupported/malformed queries. */
size_t pl_dns_answer(uint8_t *packet, size_t length, size_t capacity);
