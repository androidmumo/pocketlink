#pragma once
#include <stdbool.h>
/* Callers serialize access; claiming also covers time waiting in the work queue. */
typedef struct { bool busy; unsigned frame; } pl_activity;
bool pl_activity_begin(pl_activity *state);
void pl_activity_end(pl_activity *state);
char pl_activity_tick(pl_activity *state);
