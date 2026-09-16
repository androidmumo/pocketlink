#include "pocketlink_ota.h"
#include "esp_ota_ops.h"
typedef struct { uint32_t address; int64_t sequence; } pending_update;
bool pl_ota_partition(const esp_partition_t *partition) {
    return partition && partition->size==0x300000 && partition->type==ESP_PARTITION_TYPE_APP &&
        ((partition->address==0x10000 && partition->subtype==ESP_PARTITION_SUBTYPE_APP_OTA_0) ||
         (partition->address==0x370000 && partition->subtype==ESP_PARTITION_SUBTYPE_APP_OTA_1));
}
esp_err_t pl_ota_activate(nvs_handle_t storage,const pl_ota_offer *offer,const esp_partition_t *written){
    if(!pl_ota_partition(written))return ESP_ERR_INVALID_ARG;
    pending_update pending={.address=written->address,.sequence=offer->sequence};
    esp_err_t e=nvs_set_blob(storage,"ota_pending",&pending,sizeof(pending));if(e==ESP_OK)e=nvs_commit(storage);if(e!=ESP_OK)return e;
    e=esp_ota_set_boot_partition(written);
    if(e!=ESP_OK){nvs_erase_key(storage,"ota_pending");nvs_commit(storage);}return e;
}
int64_t pl_ota_sequence(nvs_handle_t storage){int64_t sequence=0;nvs_get_i64(storage,"ota_seq",&sequence);return sequence;}
esp_err_t pl_ota_confirm(nvs_handle_t storage){
    const esp_partition_t *running=esp_ota_get_running_partition();if(!pl_ota_partition(running))return ESP_ERR_NOT_SUPPORTED;
    pending_update pending;size_t length=sizeof(pending);esp_err_t e=nvs_get_blob(storage,"ota_pending",&pending,&length);
    esp_ota_img_states_t state;bool verifying=esp_ota_get_state_partition(running,&state)==ESP_OK && state==ESP_OTA_IMG_PENDING_VERIFY;
    if(e==ESP_ERR_NVS_NOT_FOUND)return verifying?ESP_ERR_INVALID_STATE:ESP_OK;
    if(e!=ESP_OK || length!=sizeof(pending) || pending.sequence<1)return ESP_ERR_INVALID_STATE;
    if(pending.address!=running->address){nvs_erase_key(storage,"ota_pending");return nvs_commit(storage);}
    if(verifying && (e=esp_ota_mark_app_valid_cancel_rollback())!=ESP_OK)return e;
    if((e=nvs_set_i64(storage,"ota_seq",pending.sequence))!=ESP_OK)return e;
    if((e=nvs_erase_key(storage,"ota_pending"))!=ESP_OK)return e;
    return nvs_commit(storage);
}
void pl_ota_boot_failed(void){
    esp_ota_img_states_t state;const esp_partition_t *running=esp_ota_get_running_partition();
    if(esp_ota_get_state_partition(running,&state)==ESP_OK && state==ESP_OTA_IMG_PENDING_VERIFY)esp_ota_mark_app_invalid_rollback_and_reboot();
}

bool pl_ota_available(void){return pl_ota_partition(esp_ota_get_running_partition()) && pl_ota_partition(esp_ota_get_next_update_partition(NULL));}
