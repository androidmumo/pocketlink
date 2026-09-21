#include "pocketlink_activity.h"
bool pl_activity_begin(pl_activity *state) {
    if (state->busy) return false;
    state->busy = true;
    state->frame = 0;
    return true;
}
void pl_activity_end(pl_activity *state) { state->busy = false; state->frame = 0; }
char pl_activity_tick(pl_activity *state) {
    if (!state->busy) return ' ';
    return "|/-\\"[state->frame++ % 4];
}
