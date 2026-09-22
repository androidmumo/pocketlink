#pragma once
#include <stdbool.h>
#include <stdint.h>
typedef struct { bool held,armed; uint32_t request,pending,stream; int64_t requested_at,started_at; } pl_ptt;
uint32_t pl_ptt_hold(pl_ptt *state,bool held);
uint32_t pl_ptt_request(pl_ptt *state,int64_t now);
bool pl_ptt_grant(pl_ptt *state,uint32_t request,uint32_t stream,int64_t now);
bool pl_ptt_deny(pl_ptt *state,uint32_t request);
bool pl_ptt_timeout(pl_ptt *state,int64_t now);
uint32_t pl_ptt_expire(pl_ptt *state,int64_t now);
bool pl_ptt_floor(pl_ptt *state,uint32_t stream);
