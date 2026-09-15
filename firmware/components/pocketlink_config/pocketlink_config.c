#include "pocketlink_config.h"
#include <string.h>
static bool alpha(char c) { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'); }
static bool digit(char c) { return c >= '0' && c <= '9'; }
bool pl_secret_valid(const char *s) {
    static const char alphabet[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
    if (!s || strlen(s) != 43) return false;
    for (unsigned i = 0; i < 43; i++) {
        const char *digit = strchr(alphabet, s[i]);
        if (!digit || (i == 42 && (digit - alphabet) % 4 != 0)) return false;
    }
    return true;
}
/* Reject overlong encodings, surrogates, truncated sequences and invalid bytes. */
bool pl_utf8_valid(const char *s) {
    const unsigned char *p = (const unsigned char *)s;
    while (*p) {
        unsigned value = *p++, count, minimum;
        if (value < 128) continue;
        if (value >= 0xc2 && value <= 0xdf) { value &= 31; count = 1; minimum = 128; }
        else if (value >= 0xe0 && value <= 0xef) { value &= 15; count = 2; minimum = 2048; }
        else if (value >= 0xf0 && value <= 0xf4) { value &= 7; count = 3; minimum = 65536; }
        else return false;
        while (count--) {
            if ((*p & 0xc0) != 0x80) return false;
            value = (value << 6) | (*p++ & 63);
        }
        if (value < minimum || value > 0x10ffff || (value >= 0xd800 && value <= 0xdfff)) return false;
    }
    return true;
}
/* The portal accepts flat JSON; bound nesting before entering cJSON on a small stack. */
bool pl_json_shallow(const char *s) {
    unsigned depth = 0; bool quoted = false, escaped = false;
    for (; *s; s++) {
        if (quoted) {
            if (escaped) escaped = false;
            else if (*s == '\\') escaped = true;
            else if (*s == '"') quoted = false;
        } else if (*s == '"') quoted = true;
        else if (*s == '{' || *s == '[') { if (++depth > 4) return false; }
        else if (*s == '}' || *s == ']') { if (!depth) return false; --depth; }
    }
    return !quoted && !depth;
}
bool pl_host_valid(const char *s) {
    if (!s || !*s || strlen(s) > 253) return false;
    unsigned label = 0;
    char previous = 0;
    for (; *s; previous = *s++) {
        if (*s == '.') {
            if (!label || previous == '-') return false;
            label = 0;
        } else {
            if (!alpha(*s) && !digit(*s) && *s != '-') return false;
            if (!label && *s == '-') return false;
            if (++label > 63) return false;
        }
    }
    return label && previous != '-';
}
bool pl_config_valid(const pl_config *c) {
    if (!c || c->version != PL_CONFIG_VERSION || !c->port ||
        !memchr(c->ssid, 0, sizeof(c->ssid)) || !c->ssid[0] ||
        !memchr(c->password, 0, sizeof(c->password)) ||
        !memchr(c->host, 0, sizeof(c->host)) ||
        !memchr(c->credential, 0, sizeof(c->credential))) return false;
    size_t n = strlen(c->password);
    if (n && (n < 8 || n > 63)) return false;
    return pl_utf8_valid(c->ssid) && pl_utf8_valid(c->password) && pl_host_valid(c->host) && (!c->credential[0] || pl_secret_valid(c->credential));
}
