#include "pocketlink_inbox.h"
#include <assert.h>
#include <string.h>
int main(void) {
    pl_inbox box = {0}; assert(pl_inbox_valid(&box)); assert(!pl_inbox_read(&box));
    assert(pl_inbox_accept(&box, 7, "离线消息"));
    assert(!pl_inbox_accept(&box, 8, "must not replace unread message"));
    /* Power loss before sending the receipt leaves a retryable durable message. */
    pl_inbox rebooted = box; assert(pl_inbox_valid(&rebooted));
    assert(rebooted.receipt == PL_RECEIVED_PENDING);
    pl_inbox_acknowledged(&rebooted); assert(rebooted.id == 7);
    pl_inbox_acknowledged(&rebooted); assert(rebooted.id == 7);
    assert(pl_inbox_read(&rebooted)); box = rebooted;
    assert(box.receipt == PL_READ_PENDING); assert(box.id == 7);
    pl_inbox_acknowledged(&box); assert(box.id == 0 && pl_inbox_valid(&box));
    /* Reading before the received ACK is allowed: server read implies received. */
    assert(pl_inbox_accept(&box, 8, "read immediately")); assert(pl_inbox_read(&box));
    pl_inbox_acknowledged(&box); assert(pl_inbox_valid(&box));
    assert(!pl_inbox_accept(&box, 0, "invalid"));
    assert(!pl_inbox_accept(&box, 1, "\xed\xa0\x80"));
    char large[PL_TEXT_LIMIT+2]; memset(large,'a',sizeof(large)); large[sizeof(large)-1]=0;
    assert(!pl_inbox_accept(&box, 1, large));
    box.id=1; assert(!pl_inbox_valid(&box));
    box.receipt=PL_RECEIVED; memset(box.text,'a',sizeof(box.text)); assert(!pl_inbox_valid(&box));
    return 0;
}
