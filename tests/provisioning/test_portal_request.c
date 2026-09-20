#include "pocketlink_portal.h"
#include <arpa/inet.h>
#include <assert.h>
#include <stdio.h>
#include <string.h>

int main(void) {
    struct sockaddr_in v4 = {.sin_family = AF_INET};
    assert(inet_pton(AF_INET, "192.168.4.1", &v4.sin_addr) == 1);
    assert(pl_portal_local_address((struct sockaddr *)&v4, sizeof(v4)));
    assert(!pl_portal_local_address((struct sockaddr *)&v4, sizeof(v4)-1));
    assert(inet_pton(AF_INET, "192.168.1.10", &v4.sin_addr) == 1);
    assert(!pl_portal_local_address((struct sockaddr *)&v4, sizeof(v4)));
    struct sockaddr_in6 v6 = {.sin6_family = AF_INET6};
    assert(inet_pton(AF_INET6, "::ffff:192.168.4.1", &v6.sin6_addr) == 1);
    assert(pl_portal_local_address((struct sockaddr *)&v6, sizeof(v6)));
    assert(!pl_portal_local_address((struct sockaddr *)&v6, sizeof(v4)));
    const char *denied[] = {"::ffff:192.168.1.10", "::192.168.4.1", "::1", "fe80::1", "::"};
    for (unsigned i=0; i<sizeof(denied)/sizeof(*denied); ++i) {
        assert(inet_pton(AF_INET6, denied[i], &v6.sin6_addr) == 1);
        assert(!pl_portal_local_address((struct sockaddr *)&v6, sizeof(v6)));
    }
    assert(!pl_portal_local_address(NULL, 0));
    struct sockaddr unknown = {.sa_family = AF_UNSPEC};
    assert(!pl_portal_local_address(&unknown, sizeof(unknown)));
    assert(pl_portal_host("192.168.4.1"));
    assert(pl_portal_host("192.168.4.1:80"));
    assert(!pl_portal_host("192.168.4.1:8080"));
    assert(!pl_portal_host("192.168.4.1.evil.test"));
    assert(!pl_portal_host(NULL));
    assert(pl_portal_origin("http://192.168.4.1"));
    assert(pl_portal_origin("http://192.168.4.1:80"));
    assert(!pl_portal_origin("https://192.168.4.1"));
    assert(!pl_portal_origin("http://192.168.4.1:8080"));
    assert(!pl_portal_origin("http://192.168.4.1.evil.test"));
    assert(!pl_portal_origin(NULL));
    puts("Portal IPv4/dual-stack access, truncation and Host/Origin boundaries: PASS");
}
