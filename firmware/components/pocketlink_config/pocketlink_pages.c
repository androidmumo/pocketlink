#include "pocketlink_pages.h"
#include <string.h>
pl_page pl_page_find(const char *text, size_t lines, size_t requested,
                     pl_next_line next_line, void *context) {
    pl_page result = {0, 1, 0, 0};
    if (!text || !lines || !next_line) return result;
    size_t length = strlen(text), pos = 0, count = 0;
    do {
        size_t begin = pos;
        for (size_t line = 0; line < lines && pos < length; line++) {
            size_t n = next_line(text + pos, length - pos, context);
            if (!n || n > length - pos) return (pl_page){0, 1, 0, 0};
            pos += n;
        }
        if (count <= requested) result = (pl_page){count, 0, begin, pos};
        count++;
    } while (pos < length);
    result.count = count;
    return result;
}
