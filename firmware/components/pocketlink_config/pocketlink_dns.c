#include "pocketlink_dns.h"
#include <string.h>
size_t pl_dns_answer(uint8_t *packet, size_t length, size_t capacity) {
    if (!packet || length > capacity || length < 17 || capacity - length < 16 ||
        (packet[2] & 0xf8) || packet[4] || packet[5] != 1) return 0;
    size_t p = 12;
    while (p < length && packet[p] && packet[p] <= 63) p += packet[p] + 1;
    if (p + 5 != length || packet[p] || packet[p+1] || packet[p+2] != 1 || packet[p+3] || packet[p+4] != 1) return 0;
    packet[2] = 0x81; packet[3] = 0x80; packet[6] = 0; packet[7] = 1;
    memset(packet + 8, 0, 4);
    const uint8_t answer[] = {0xc0,0x0c,0,1,0,1,0,0,0,0,0,4,192,168,4,1};
    memcpy(packet + length, answer, sizeof(answer));
    return length + sizeof(answer);
}
