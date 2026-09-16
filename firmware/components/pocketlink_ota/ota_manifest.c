#include "pocketlink_ota.h"
#include <string.h>
#include "cJSON.h"
#include "mbedtls/base64.h"
#include "mbedtls/pk.h"
#include "mbedtls/sha256.h"
extern const char trust_start[] __asm__("_binary_trust_pem_start");
static bool number(const cJSON *value, double low, double high) {
    return cJSON_IsNumber(value) && value->valuedouble >= low && value->valuedouble <= high && value->valuedouble == (double)(int64_t)value->valuedouble;
}
static const char *string(const cJSON *obj, const char *key) {
    const cJSON *value = cJSON_GetObjectItemCaseSensitive(obj, key);
    return cJSON_IsString(value) ? value->valuestring : "";
}
esp_err_t pl_ota_parse(const char *manifest, size_t length, int64_t installed, pl_ota_offer *offer) {
    if (!manifest || !offer || !length || length > 4096 || memchr(manifest, 0, length)) return ESP_ERR_INVALID_ARG;
    char bounded[4097]; memcpy(bounded, manifest, length); bounded[length] = 0;
    if (!pl_json_shallow(bounded)) return ESP_ERR_INVALID_ARG;
    esp_err_t result = ESP_ERR_INVALID_CRC;
    cJSON *envelope = cJSON_ParseWithLengthOpts(bounded, length + 1, NULL, true), *metadata = NULL;
    unsigned char payload[1025], signature[80], digest[32]; size_t payload_size = 0, signature_size = 0;
    mbedtls_pk_context key; mbedtls_pk_init(&key);
    if (!cJSON_IsObject(envelope) || cJSON_GetArraySize(envelope) != 2) goto done;
    const char *encoded = string(envelope, "payload"), *signed_hash = string(envelope, "signature");
    if (mbedtls_base64_decode(payload, sizeof(payload)-1, &payload_size, (const unsigned char *)encoded, strlen(encoded)) ||
        mbedtls_base64_decode(signature, sizeof(signature), &signature_size, (const unsigned char *)signed_hash, strlen(signed_hash))) goto done;
    payload[payload_size] = 0;
    if (memchr(payload, 0, payload_size) || !pl_json_shallow((char *)payload)) goto done;
    if (mbedtls_pk_parse_public_key(&key, (const unsigned char *)trust_start, strlen(trust_start)+1) ||
        mbedtls_sha256(payload, payload_size, digest, 0) ||
        mbedtls_pk_verify(&key, MBEDTLS_MD_SHA256, digest, sizeof(digest), signature, signature_size)) goto done;
    metadata = cJSON_ParseWithLengthOpts((char *)payload, payload_size+1, NULL, true);
    const char *version = string(metadata,"version"), *sha = string(metadata,"sha256");
    const cJSON *format = cJSON_GetObjectItemCaseSensitive(metadata,"format");
    const cJSON *sequence = cJSON_GetObjectItemCaseSensitive(metadata,"sequence");
    const cJSON *size = cJSON_GetObjectItemCaseSensitive(metadata,"size");
    if (!cJSON_IsObject(metadata) || cJSON_GetArraySize(metadata)!=7 || !number(format,1,1) ||
        strcmp(string(metadata,"board"),"ai-passport-esp32c3") || strcmp(string(metadata,"app"),"pocketlink") ||
        !number(sequence,1,2147483647) || !number(size,288,0x300000) || !*version || strlen(version)>32 || strlen(sha)!=64) goto done;
    for (unsigned i=0; version[i]; i++) if (!((version[i]>='a'&&version[i]<='z') || (version[i]>='A'&&version[i]<='Z') || (version[i]>='0'&&version[i]<='9') || (i && strchr("._-",version[i])))) goto done;
    for (unsigned i=0;i<64;i++) if (!((sha[i]>='0'&&sha[i]<='9') || (sha[i]>='a'&&sha[i]<='f'))) goto done;
    if (sequence->valuedouble<=installed) { result=ESP_ERR_INVALID_VERSION; goto done; }
    memset(offer,0,sizeof(*offer)); offer->sequence=(int64_t)sequence->valuedouble; offer->size=(int)size->valuedouble;
    strcpy(offer->version,version); strcpy(offer->sha256,sha); result=ESP_OK;
done:
    mbedtls_pk_free(&key); cJSON_Delete(metadata); cJSON_Delete(envelope); return result;
}
