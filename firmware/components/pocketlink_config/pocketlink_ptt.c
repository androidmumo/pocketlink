#include "pocketlink_ptt.h"
uint32_t pl_ptt_hold(pl_ptt *s,bool held){s->held=held;if(held)return 0;s->armed=true;s->pending=0;uint32_t old=s->stream;s->stream=0;return old;}
uint32_t pl_ptt_request(pl_ptt *s,int64_t now){if(!s->held||!s->armed||s->pending||s->stream)return 0;s->request++;if(!s->request)s->request++;s->pending=s->request;s->requested_at=now;s->armed=false;return s->pending;}
bool pl_ptt_grant(pl_ptt *s,uint32_t request,uint32_t stream,int64_t now){if(!stream||!s->held||!s->pending||request!=s->pending)return false;s->stream=stream;s->pending=0;s->started_at=now;return true;}
bool pl_ptt_deny(pl_ptt *s,uint32_t request){if(!request||request!=s->pending)return false;s->pending=0;return true;}
bool pl_ptt_timeout(pl_ptt *s,int64_t now){if(!s->pending||now-s->requested_at<3000000)return false;s->pending=0;return true;}
uint32_t pl_ptt_expire(pl_ptt *s,int64_t now){if(!s->stream||now-s->started_at<29000000)return 0;uint32_t old=s->stream;s->stream=0;return old;}
bool pl_ptt_floor(pl_ptt *s,uint32_t stream){if(!s->stream||s->stream==stream)return false;s->stream=0;return true;}
