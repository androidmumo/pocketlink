#include "pocketlink_inbox.h"
#include "pocketlink_config.h"
#include <string.h>
bool pl_inbox_valid(const pl_inbox *box) {
    return box && box->id >= 0 && box->receipt <= PL_READ_PENDING &&
        ((box->id == 0) == (box->receipt == PL_EMPTY)) &&
        memchr(box->text, 0, sizeof(box->text)) && pl_utf8_valid(box->text);
}
bool pl_inbox_accept(pl_inbox *box, int64_t id, const char *text) {
    if (!box || box->id || id <= 0 || !text || strlen(text) > PL_TEXT_LIMIT || !pl_utf8_valid(text)) return false;
    box->id = id; strcpy(box->text, text); box->receipt = PL_RECEIVED_PENDING;
    return true;
}
bool pl_inbox_read(pl_inbox *box) {
    if (!box || !box->id) return false;
    box->receipt = PL_READ_PENDING;
    return true;
}
void pl_inbox_acknowledged(pl_inbox *box) {
    if (!box) return;
    if (box->receipt == PL_READ_PENDING) memset(box, 0, sizeof(*box));
    else if (box->receipt == PL_RECEIVED_PENDING) box->receipt = PL_RECEIVED;
}
