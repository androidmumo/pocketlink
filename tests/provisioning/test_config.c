#include "pocketlink_config.h"
#include <assert.h>
#include <string.h>
int main(void) {
    pl_config c = {.version=PL_CONFIG_VERSION, .ssid="家庭网络", .host="pocketlink.mcloc.cn", .port=443};
    assert(pl_config_valid(&c));
    const char *bad[] = {"", "https://evil", "example.com/path", "user@host", "a\r\nHost:b", "a..b", "-a.b", "a-.b", "a.b.", "a b", "[::1]"};
    for (unsigned i=0;i<sizeof(bad)/sizeof(*bad);i++) assert(!pl_host_valid(bad[i]));
    assert(pl_host_valid("192.168.1.2"));
    memset(c.ssid,'a',sizeof(c.ssid)); assert(!pl_config_valid(&c));
    strcpy(c.ssid,"wifi"); strcpy(c.password,"short"); assert(!pl_config_valid(&c));
    strcpy(c.password,"12345678"); assert(pl_config_valid(&c));
    memset(c.credential,'a',42); c.credential[42]='A'; c.credential[43]=0; assert(pl_config_valid(&c));
    c.credential[42]='B'; assert(!pl_config_valid(&c));
    c.credential[0]=0; c.port=0; assert(!pl_config_valid(&c));
    assert(pl_secret_valid("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"));
    assert(!pl_secret_valid("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="));
    assert(!pl_secret_valid("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB"));
    assert(!pl_secret_valid("short"));
    assert(pl_utf8_valid("你好🌐"));
    assert(!pl_utf8_valid("\xc0\xaf")); assert(!pl_utf8_valid("\xed\xa0\x80"));
    assert(!pl_utf8_valid("\xf4\x90\x80\x80")); assert(!pl_utf8_valid("\xe4\xb8"));
    assert(pl_json_shallow("{\"password\":\"[{}]\"}"));
    assert(!pl_json_shallow("[[[[[]]]]]")); assert(!pl_json_shallow("{\"a\":\"open}"));
    return 0;
}
