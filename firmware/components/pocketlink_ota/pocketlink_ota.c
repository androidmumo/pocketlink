#include "pocketlink_ota.h"
#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "cJSON.h"
#include "esp_crt_bundle.h"
#include "esp_http_client.h"
#include "esp_ota_ops.h"
#include "esp_timer.h"
#include "mbedtls/base64.h"
#include "mbedtls/pk.h"
#include "mbedtls/sha256.h"

esp_err_t pl_ota_download(const pl_config *config, const pl_ota_offer *offer, void (*progress)(int), const esp_partition_t **written) {
    *written=NULL;
    const esp_partition_t *running=esp_ota_get_running_partition(), *target=esp_ota_get_next_update_partition(NULL);
    if (!pl_ota_partition(running) || !pl_ota_partition(target) || target->address==running->address) return ESP_ERR_NOT_SUPPORTED;
    char url[400], auth[80];
    snprintf(url,sizeof(url),"https://%s:%u/api/v1/device/firmware/%s",config->host,config->port,offer->sha256);
    snprintf(auth,sizeof(auth),"Bearer %s",config->credential);
    esp_http_client_config_t options={.url=url,.crt_bundle_attach=esp_crt_bundle_attach,.timeout_ms=10000,.disable_auto_redirect=true,.buffer_size=2048,.buffer_size_tx=1024};
    esp_http_client_handle_t client=esp_http_client_init(&options);if(!client)return ESP_ERR_NO_MEM;
    esp_http_client_set_header(client,"Authorization",auth);
    esp_err_t result=ESP_FAIL;esp_ota_handle_t ota=0;bool begun=false;
    mbedtls_sha256_context sha;mbedtls_sha256_init(&sha);
    if(esp_http_client_open(client,0)!=ESP_OK)goto done;
    if(esp_http_client_fetch_headers(client)!=offer->size || esp_http_client_get_status_code(client)!=200)goto done;
    if(esp_ota_begin(target,offer->size,&ota)!=ESP_OK)goto done;
    begun=true;
    if(mbedtls_sha256_starts(&sha,0))goto done;
    uint8_t buffer[4096];int received=0,last=-1;int64_t deadline=esp_timer_get_time()+180000000;
    while(received<offer->size){
        if(esp_timer_get_time()>deadline)goto done;
        int remaining=offer->size-received;
        int n=esp_http_client_read(client,(char *)buffer,remaining<(int)sizeof(buffer)?remaining:sizeof(buffer));
        if(n<=0 || esp_ota_write(ota,buffer,n)!=ESP_OK || mbedtls_sha256_update(&sha,buffer,n))goto done;
        received+=n;int percent=received*100/offer->size;if(progress&&percent!=last){last=percent;progress(percent);}
    }
    if(!esp_http_client_is_complete_data_received(client))goto done;
    unsigned char digest[32];char hex[65];if(mbedtls_sha256_finish(&sha,digest))goto done;
    for(unsigned i=0;i<32;i++)snprintf(hex+2*i,3,"%02x",digest[i]);
    if(strcmp(hex,offer->sha256)){result=ESP_ERR_INVALID_CRC;goto done;}
    result=esp_ota_end(ota);begun=false;if(result==ESP_OK)*written=target;
done:
    if(begun)esp_ota_abort(ota);
    mbedtls_sha256_free(&sha);esp_http_client_cleanup(client);memset(auth,0,sizeof(auth));return result;
}
