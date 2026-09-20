#include "pocketlink_portal.h"
#include <netinet/in.h>
#include <string.h>

bool pl_portal_local_address(const struct sockaddr *address, socklen_t length) {
    if (!address || length < sizeof(struct sockaddr)) return false;
    if (address->sa_family == AF_INET && length >= sizeof(struct sockaddr_in)) {
        struct sockaddr_in ipv4;
        memcpy(&ipv4, address, sizeof(ipv4));
        const unsigned char expected[4] = {192, 168, 4, 1};
        return memcmp(&ipv4.sin_addr, expected, sizeof(expected)) == 0;
    }
    if (address->sa_family == AF_INET6 && length >= sizeof(struct sockaddr_in6)) {
        struct sockaddr_in6 ipv6;
        memcpy(&ipv6, address, sizeof(ipv6));
        /* ESP-IDF's dual-stack HTTP listener returns IPv4-mapped IPv6 addresses. */
        const unsigned char expected[16] = {0,0,0,0,0,0,0,0,0,0,255,255,192,168,4,1};
        return memcmp(&ipv6.sin6_addr, expected, sizeof(expected)) == 0;
    }
    return false;
}

bool pl_portal_host(const char *host) {
    return host && (!strcmp(host, "192.168.4.1") || !strcmp(host, "192.168.4.1:80"));
}

bool pl_portal_origin(const char *origin) {
    return origin && (!strcmp(origin, "http://192.168.4.1") || !strcmp(origin, "http://192.168.4.1:80"));
}
