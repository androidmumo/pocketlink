#include "pocketlink_dns.h"
#include <assert.h>
#include <string.h>
int main(void) {
    const uint8_t query[] = {0x12,0x34,1,0,0,1,0,0,0,0,0,0,1,'a',0,0,1,0,1};
    uint8_t packet[512];
    memcpy(packet,query,sizeof(query));
    assert(pl_dns_answer(packet,sizeof(query),sizeof(packet))==sizeof(query)+16);
    assert(packet[0]==0x12 && packet[1]==0x34 && packet[7]==1);
    assert(packet[sizeof(query)+12]==192 && packet[sizeof(query)+15]==1);
    for(size_t n=0;n<sizeof(query);n++) {
        memcpy(packet,query,sizeof(query)); assert(!pl_dns_answer(packet,n,sizeof(packet)));
    }
    memcpy(packet,query,sizeof(query)); assert(!pl_dns_answer(packet,sizeof(query),sizeof(query)+15));
    memcpy(packet,query,sizeof(query)); packet[16]=28; assert(!pl_dns_answer(packet,sizeof(query),sizeof(packet)));
    memcpy(packet,query,sizeof(query)); packet[12]=0xc0; assert(!pl_dns_answer(packet,sizeof(query),sizeof(packet)));
    memcpy(packet,query,sizeof(query)); packet[12]=63; assert(!pl_dns_answer(packet,sizeof(query),sizeof(packet)));
    memcpy(packet,query,sizeof(query)); packet[5]=2; assert(!pl_dns_answer(packet,sizeof(query),sizeof(packet)));
    /* Exercise all declared lengths against a deterministic malformed corpus. */
    uint32_t seed=1;
    for(unsigned sample=0;sample<5000;sample++) {
        for(unsigned i=0;i<sizeof(packet);i++) {seed=1664525*seed+1013904223;packet[i]=(uint8_t)(seed>>24);}
        size_t n=sample%513; size_t result=pl_dns_answer(packet,n,sizeof(packet));
        assert(result==0 || (result==n+16 && result<=sizeof(packet)));
    }
    return 0;
}
