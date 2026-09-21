#include "pocketlink_pages.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>
/* Two UTF-8 characters per test line, honoring explicit newlines. */
static size_t line(const char *s, size_t remaining, void *context) {
    (void)context;
    size_t n = 0;
    for (unsigned i = 0; i < 2 && n < remaining; i++) {
        unsigned char c = (unsigned char)s[n];
        if (c == '\n') return n + 1;
        n += c < 128 ? 1 : c < 224 ? 2 : c < 240 ? 3 : 4;
    }
    return n;
}
int main(void) {
    pl_page p = pl_page_find("短消息", 8, 99, line, NULL);
    assert(p.count == 1 && p.index == 0 && p.begin == 0 && p.end == strlen("短消息"));
    const char *cases[] = {"", "abcd", "abcde", "中文甲乙丙丁戊己庚", "a\n\n\nb", "甲😀乙\nabc", "\n\n\n\n\n"};
    for (size_t c = 0; c < sizeof(cases)/sizeof(*cases); c++) {
        const char *s = cases[c]; size_t pos = 0;
        pl_page first = pl_page_find(s, 2, 0, line, NULL);
        for (size_t i = 0; i < first.count; i++) {
            p = pl_page_find(s, 2, i, line, NULL);
            assert(p.index == i && p.begin == pos && p.end <= strlen(s));
            assert(!s[p.begin] || ((unsigned char)s[p.begin] & 0xc0) != 0x80);
            pos = p.end;
        }
        assert(pos == strlen(s));
        p = pl_page_find(s, 2, 999, line, NULL);
        assert(p.index == first.count - 1 && p.end == strlen(s));
    }
    assert(pl_page_find("abcd", 2, 0, line, NULL).count == 1);
    assert(pl_page_find("abcde", 2, 0, line, NULL).count == 2);
    puts("UTF-8 pagination, exact boundaries, blank lines and page clamping: PASS");
}
