#include "pocketlink_activity.h"
bool pl_activity_begin(pl_activity *state, bool visible) {
    if (state->busy) return false;
    state->busy = true;
    state->visible = visible;
    state->frame = 0;
    return true;
}
void pl_activity_end(pl_activity *state) { state->busy = false; state->visible = false; state->frame = 0; }
char pl_activity_tick(pl_activity *state) {
    if (!state->busy || !state->visible) return ' ';
    return "|/-\\"[state->frame++ % 4];
}
