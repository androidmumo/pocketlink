#include "pocketlink_activity.h"
#include <assert.h>
#include <stdio.h>
int main(void) {
    pl_activity state = {0};
    assert(pl_activity_tick(&state) == ' ');
    assert(pl_activity_begin(&state)); /* Lock immediately, before dispatch. */
    for (unsigned i = 0; i < 100; i++) {
        assert(!pl_activity_begin(&state)); /* Repeated presses and HTTP submissions are rejected. */
        assert(pl_activity_tick(&state) == "|/-\\"[i % 4]);
    }
    pl_activity_end(&state); /* Both success and failure release the operation. */
    assert(!state.busy && pl_activity_tick(&state) == ' ');
    assert(pl_activity_begin(&state));
    assert(pl_activity_tick(&state) == '|');
    pl_activity_end(&state); /* A queue-send failure also leaves it reusable. */
    assert(pl_activity_begin(&state));
    puts("Operation reservation, repeated-input rejection, spinner and recovery: PASS");
}
