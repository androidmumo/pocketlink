#include "pocketlink_voice.h"
#include "bsp_audio.h"
#include "pocketlink_ptt.h"
#include "esp_websocket_client.h"
#include "esp_crt_bundle.h"
#include "esp_timer.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/queue.h"
#include "freertos/semphr.h"
#include "cJSON.h"
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define FRAME 648
/* Queues contain at most 160ms of audio; callbacks never touch UI or codecs. */
typedef struct { int len,kind; int64_t at; uint8_t data[1025]; } packet;
static esp_websocket_client_handle_t client;
static QueueHandle_t incoming;
static SemaphoreHandle_t finished;
static atomic_bool running,held,connected,fault;
static portMUX_TYPE guard=portMUX_INITIALIZER_UNLOCKED;
static char message[192]="尚未连接";
static char url[400],headers[100];
static void state(const char *text){portENTER_CRITICAL(&guard);snprintf(message,sizeof(message),"%s",text);portEXIT_CRITICAL(&guard);}
void pl_voice_status(char *text,size_t size){portENTER_CRITICAL(&guard);snprintf(text,size,"%s",message);portEXIT_CRITICAL(&guard);}
void pl_voice_hold(bool value){atomic_store(&held,value);}
static uint32_t read32(const uint8_t *p){return ((uint32_t)p[0]<<24)|((uint32_t)p[1]<<16)|((uint32_t)p[2]<<8)|p[3];}
static void write32(uint8_t *p,uint32_t v){p[0]=v>>24;p[1]=v>>16;p[2]=v>>8;p[3]=v;}
static void ws_event(void *arg,esp_event_base_t base,int32_t id,void *raw){
 (void)arg;(void)base;
 if(id==WEBSOCKET_EVENT_CONNECTED){atomic_store(&connected,true);state("已连接，按住确定讲话");return;}
 if(id==WEBSOCKET_EVENT_DISCONNECTED||id==WEBSOCKET_EVENT_ERROR||id==WEBSOCKET_EVENT_CLOSED){atomic_store(&connected,false);atomic_store(&held,false);atomic_store(&fault,true);return;}
 if(id!=WEBSOCKET_EVENT_DATA)return;
 esp_websocket_event_data_t *e=raw;
 if(e->op_code==9||e->op_code==10)return;
 /* Server emits bounded, unfragmented controls and PCM frames. */
 if((e->op_code!=1&&e->op_code!=2)||!e->fin||e->payload_offset!=0||e->data_len!=e->payload_len||e->data_len<1||e->data_len>1024){atomic_store(&fault,true);return;}
 packet p={.len=e->data_len,.kind=e->op_code,.at=esp_timer_get_time()};memcpy(p.data,e->data_ptr,p.len);p.data[p.len]=0;
 if(xQueueSend(incoming,&p,0)!=pdTRUE)atomic_store(&fault,true);
}
static bool control(const char *type,uint32_t value){char text[96];snprintf(text,sizeof(text),"{\"type\":\"%s\",\"%s\":%lu}",type,strcmp(type,"request")==0?"request_id":"stream",(unsigned long)value);return esp_websocket_client_send_text(client,text,strlen(text),pdMS_TO_TICKS(100))==(int)strlen(text);}
static uint32_t number(cJSON *v,const char *key){cJSON *n=cJSON_GetObjectItemCaseSensitive(v,key);if(!cJSON_IsNumber(n)||n->valuedouble<0||n->valuedouble>4294967295.0||n->valuedouble!=(double)(uint32_t)n->valuedouble)return 0;return (uint32_t)n->valuedouble;}
static void task(void *arg){
 (void)arg;pl_ptt ptt={.armed=true};uint32_t rx=0,seq=0,rxseq=0;
 uint8_t frame[FRAME];int16_t silence[320]={0};packet p;
 while(atomic_load(&running)&&!atomic_load(&fault)){
  uint32_t released=pl_ptt_hold(&ptt,atomic_load(&held));
  if(released){if(!control("release",released))break;bsp_audio_write(silence,sizeof(silence));}
  if(atomic_load(&connected)){uint32_t request=pl_ptt_request(&ptt,esp_timer_get_time());if(request){state("正在申请讲话");if(!control("request",request))break;}}
  if(pl_ptt_timeout(&ptt,esp_timer_get_time()))state("申请超时，请松开后重试");
  released=pl_ptt_expire(&ptt,esp_timer_get_time());if(released){if(!control("release",released))break;state("已到单次时限，请松开");}
  for(int processed=0;processed<4&&xQueueReceive(incoming,&p,0)==pdTRUE;processed++){
   if(p.kind==1){cJSON *v=cJSON_Parse((char *)p.data);cJSON *kind=v?cJSON_GetObjectItemCaseSensitive(v,"type"):NULL;
    if(!cJSON_IsString(kind)){cJSON_Delete(v);atomic_store(&fault,true);break;}
    uint32_t stream=number(v,"stream");
    if(strcmp(kind->valuestring,"grant")==0){if(!atomic_load(&held)||!pl_ptt_grant(&ptt,number(v,"request_id"),stream,esp_timer_get_time())){if(stream&&!control("release",stream))atomic_store(&fault,true);}else{seq=0;state("正在讲话，松开确定结束");for(int i=0;i<6&&atomic_load(&held);i++)if(bsp_audio_read(frame+8,640)!=ESP_OK){atomic_store(&fault,true);break;}}}
    else if(strcmp(kind->valuestring,"busy")==0){if(pl_ptt_deny(&ptt,number(v,"request_id"))){state("有人正在讲话，请松开后重试");}}
    else if(strcmp(kind->valuestring,"floor")==0){rx=stream;rxseq=0;if(pl_ptt_floor(&ptt,stream)){state("讲话已结束，请松开");}else if(!ptt.stream&&!ptt.pending){cJSON *name=cJSON_GetObjectItemCaseSensitive(v,"sender");if(stream&&cJSON_IsString(name))state(name->valuestring);else state("已连接，按住确定讲话");}}
    else atomic_store(&fault,true);
    cJSON_Delete(v);
   }else if(p.len==FRAME&&rx&&!ptt.stream&&read32(p.data)==rx&&read32(p.data+4)>rxseq&&esp_timer_get_time()-p.at<=160000){rxseq=read32(p.data+4);if(bsp_audio_write(p.data+8,640)!=ESP_OK){atomic_store(&fault,true);break;}}
  }
  if(atomic_load(&fault))break;
  if(ptt.stream&&atomic_load(&held)){
   if(bsp_audio_read(frame+8,640)!=ESP_OK)break;
   if(!atomic_load(&held))continue;
   write32(frame,ptt.stream);write32(frame+4,++seq);
   if(esp_websocket_client_send_bin(client,(char *)frame,FRAME,pdMS_TO_TICKS(100))!=FRAME)break;
  }else vTaskDelay(pdMS_TO_TICKS(5));
 }
 atomic_store(&connected,false);atomic_store(&held,false);atomic_store(&fault,true);
 bsp_audio_write(silence,sizeof(silence));state("对讲已断开，切换房间或重新进入");xSemaphoreGive(finished);vTaskDelete(NULL);
}
void pl_voice_stop(void){
 if(!client){return;}
 atomic_store(&held,false);atomic_store(&running,false);
 xSemaphoreTake(finished,portMAX_DELAY);
 esp_websocket_client_stop(client);esp_websocket_client_destroy(client);client=NULL;
 vQueueDelete(incoming);incoming=NULL;vSemaphoreDelete(finished);finished=NULL;
 memset(headers,0,sizeof(headers));
}
esp_err_t pl_voice_start(const pl_config *config,const char *room){
 pl_voice_stop();
 if(bsp_audio_init()!=ESP_OK||bsp_audio_set_format(16000,16,1)!=ESP_OK){state("音频初始化失败");return ESP_FAIL;}bsp_audio_set_volume(55);
 incoming=xQueueCreate(8,sizeof(packet));finished=xSemaphoreCreateBinary();if(!incoming||!finished){if(incoming)vQueueDelete(incoming);if(finished)vSemaphoreDelete(finished);return ESP_ERR_NO_MEM;}
 snprintf(url,sizeof(url),"wss://%s:%u/api/v1/device/voice/%s",config->host,config->port,room);snprintf(headers,sizeof(headers),"Authorization: Bearer %s\r\n",config->credential);
 esp_websocket_client_config_t options={.uri=url,.headers=headers,.subprotocol="pocketlink.voice.v1",.crt_bundle_attach=esp_crt_bundle_attach,.disable_auto_reconnect=true,.network_timeout_ms=2000,.task_stack=8192,.buffer_size=1024,.ping_interval_sec=20};
 client=esp_websocket_client_init(&options);if(!client){vQueueDelete(incoming);vSemaphoreDelete(finished);return ESP_ERR_NO_MEM;}
 atomic_store(&held,false);atomic_store(&connected,false);atomic_store(&fault,false);atomic_store(&running,true);state("正在连接对讲房间");
 esp_websocket_register_events(client,WEBSOCKET_EVENT_ANY,ws_event,NULL);
 if(xTaskCreate(task,"pl_voice",8192,NULL,5,NULL)!=pdPASS){atomic_store(&running,false);esp_websocket_client_destroy(client);client=NULL;vQueueDelete(incoming);vSemaphoreDelete(finished);return ESP_ERR_NO_MEM;}
 if(esp_websocket_client_start(client)!=ESP_OK){pl_voice_stop();return ESP_FAIL;}return ESP_OK;
}
