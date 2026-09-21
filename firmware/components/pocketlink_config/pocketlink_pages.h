#pragma once
#include <stddef.h>
/* The callback returns the byte length of one complete, UTF-8-safe visual line. */
typedef size_t (*pl_next_line)(const char *text, size_t remaining, void *context);
typedef struct { size_t index, count, begin, end; } pl_page;
pl_page pl_page_find(const char *text, size_t lines, size_t requested,
                     pl_next_line next_line, void *context);
