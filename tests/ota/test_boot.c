#include <assert.h>
#include <string.h>
#include <stdio.h>
#include "pocketlink_ota.h"
#include "esp_ota_ops.h"
static const esp_partition_t a={0x10000,0x300000,0,16},b={0x370000,0x300000,0,17};
static const esp_partition_t *running=&a,*selected=&a;
static unsigned char pending[32];static size_t pending_size;
static int64_t sequence;static int state,commit_fail,select_fail,rollbacks;
const esp_partition_t *esp_ota_get_running_partition(void){return running;}
const esp_partition_t *esp_ota_get_next_update_partition(const esp_partition_t *p){(void)p;return running==&a?&b:&a;}
esp_err_t esp_ota_get_state_partition(const esp_partition_t *p,int *s){(void)p;*s=state;return 0;}
esp_err_t esp_ota_mark_app_valid_cancel_rollback(void){state=0;return 0;}
esp_err_t esp_ota_mark_app_invalid_rollback_and_reboot(void){rollbacks++;return 0;}
esp_err_t esp_ota_set_boot_partition(const esp_partition_t *p){if(select_fail)return -1;selected=p;return 0;}
esp_err_t nvs_set_blob(int h,const char *k,const void *v,size_t n){(void)h;(void)k;assert(n<=sizeof(pending));memcpy(pending,v,n);pending_size=n;return 0;}
esp_err_t nvs_get_blob(int h,const char *k,void *v,size_t *n){(void)h;(void)k;if(!pending_size)return ESP_ERR_NVS_NOT_FOUND;assert(*n>=pending_size);memcpy(v,pending,pending_size);*n=pending_size;return 0;}
esp_err_t nvs_commit(int h){(void)h;return commit_fail?-1:0;}
esp_err_t nvs_erase_key(int h,const char *k){(void)h;(void)k;pending_size=0;return 0;}
esp_err_t nvs_get_i64(int h,const char *k,int64_t *v){(void)h;(void)k;*v=sequence;return 0;}
esp_err_t nvs_set_i64(int h,const char *k,int64_t v){(void)h;(void)k;sequence=v;return 0;}
int main(void){
 pl_ota_offer offer={.sequence=2};
 assert(pl_ota_available());assert(pl_ota_confirm(1)==0);assert(sequence==0);
 esp_partition_t wrong=b;wrong.address=0x356000;assert(!pl_ota_partition(&wrong));assert(pl_ota_activate(1,&offer,&wrong)==ESP_ERR_INVALID_ARG);
 commit_fail=1;assert(pl_ota_activate(1,&offer,&b)!=0);assert(selected==&a);commit_fail=0;
 /* Power loss before boot selection leaves the old application and removes stale intent. */
 assert(pl_ota_confirm(1)==0);assert(pending_size==0&&sequence==0);
 select_fail=1;assert(pl_ota_activate(1,&offer,&b)!=0);assert(!pending_size);select_fail=0;
 assert(pl_ota_activate(1,&offer,&b)==0);assert(selected==&b&&sequence==0);
 running=&b;state=ESP_OTA_IMG_PENDING_VERIFY;
 pl_ota_boot_failed();assert(rollbacks==1&&sequence==0);
 /* Simulate bootloader returning to A after failed first boot. */
 running=&a;state=0;assert(pl_ota_confirm(1)==0);assert(!pending_size&&sequence==0);
 assert(pl_ota_activate(1,&offer,&b)==0);running=&b;state=ESP_OTA_IMG_PENDING_VERIFY;
 assert(pl_ota_confirm(1)==0);assert(sequence==2&&!pending_size&&state==0);
 pl_ota_boot_failed();assert(rollbacks==1);assert(pl_ota_confirm(1)==0);
 offer.sequence=3;assert(pl_ota_activate(1,&offer,&a)==0);running=&a;state=1;
 assert(pl_ota_confirm(1)==0);assert(sequence==3&&state==0);
 state=1;assert(pl_ota_confirm(1)==ESP_ERR_INVALID_STATE);
 puts("OTA slot protection, activation failure, rollback and boot confirmation: PASS");
}
