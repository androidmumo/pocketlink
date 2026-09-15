/* PocketLink: one worker owns networking/NVS; callbacks only enqueue work.
 * The local portal is restricted to the AP interface, exact Host/Origin and a
 * random session token. No Wi-Fi password or device credential is logged. */
#include <ctype.h>
#include <assert.h>
#include <errno.h>
#include <inttypes.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <unistd.h>
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"
#include "freertos/queue.h"
#include "freertos/semphr.h"
#include "bsp_display.h"
#include "bsp_button.h"
#include "lvgl.h"
#include "cJSON.h"
#include "esp_crt_bundle.h"
#include "esp_event.h"
#include "esp_http_client.h"
#include "esp_http_server.h"
#include "esp_log.h"
#include "esp_mac.h"
#include "esp_netif.h"
#include "esp_random.h"
#include "esp_sntp.h"
#include "esp_timer.h"
#include "esp_wifi.h"
#include "lwip/inet.h"
#include "nvs.h"
#include "nvs_flash.h"
#include "pocketlink_config.h"
#include "pocketlink_inbox.h"
#include "pocketlink_dns.h"

#define AP_IP "192.168.4.1"
#define AP_ORIGIN "http://" AP_IP
#define LINK_UP BIT0
#define AP_LIFETIME_US (10LL * 60 * 1000000)
#define BODY_LIMIT 2048
#define REPLY_LIMIT (40 * 1024)
#define TEXT_LIMIT 2048

/* One atomic blob keeps binding and its inbox together across power loss. */
typedef struct {
    pl_config config;
    pl_inbox inbox;
} saved_state;
typedef enum { CMD_CONFIG, CMD_SCAN, CMD_KEY } command_kind;
typedef struct {
    command_kind kind;
    pl_config config;
    char code[65];
    bsp_btn_t key;
    bsp_btn_ev_t event;
} command;
static saved_state saved;
static SemaphoreHandle_t mutex;
static QueueHandle_t commands;
static EventGroupHandle_t events;
static nvs_handle_t storage;
static httpd_handle_t http;
static bool ap_active, busy, url_qr;
static char status_text[160] = "正在启动";
static char token[33], ap_name[33], ap_password[17], serial[32];
static char scan_names[16][33];
static unsigned scan_count;
static int64_t ap_deadline;
static lv_obj_t *title, *status_label, *qr, *message_box, *message_label, *hint;
extern const char portal_start[] asm("_binary_portal_html_start");

static void lock(void) { xSemaphoreTake(mutex, portMAX_DELAY); }
static void unlock(void) { xSemaphoreGive(mutex); }
static bool portal_active(void) { lock(); bool value = ap_active; unlock(); return value; }
static void set_status(const char *s) {
    lock(); snprintf(status_text, sizeof(status_text), "%s", s); unlock();
    if (bsp_lvgl_lock(1000)) {
        lv_label_set_text(status_label, s);
        bsp_lvgl_unlock();
    }
}
static bool persist(const saved_state *value) {
    if (nvs_set_blob(storage, "state", value, sizeof(*value)) != ESP_OK || nvs_commit(storage) != ESP_OK) {
        set_status("保存失败，请勿断电；重试前检查存储");
        return false;
    }
    lock(); saved = *value; unlock();
    return true;
}
static void render(void) {
    if (!bsp_lvgl_lock(1000)) return;
    if (ap_active) {
        char data[160];
        lv_obj_remove_flag(qr, LV_OBJ_FLAG_HIDDEN);
        lv_obj_add_flag(message_box, LV_OBJ_FLAG_HIDDEN);
        lv_label_set_text(title, url_qr ? "第二步：打开配置页" : "第一步：扫码连接热点");
        if (url_qr) snprintf(data, sizeof(data), AP_ORIGIN);
        else snprintf(data, sizeof(data), "WIFI:T:WPA;S:%s;P:%s;;", ap_name, ap_password);
        if (lv_qrcode_update(qr, data, strlen(data)) != LV_RESULT_OK) {
            lv_obj_add_flag(qr, LV_OBJ_FLAG_HIDDEN);
            lv_label_set_text(title, "二维码生成失败，请手动连接");
        }
        lv_label_set_text_fmt(hint, "%s\n密码 %s\n确定切换 · 10分钟后关闭", ap_name, ap_password);
    } else {
        lv_obj_add_flag(qr, LV_OBJ_FLAG_HIDDEN);
        lv_obj_remove_flag(message_box, LV_OBJ_FLAG_HIDDEN);
        lv_label_set_text(title, saved.inbox.id ? "收到文字消息" : "PocketLink");
        lv_label_set_text(message_label, saved.inbox.id ? saved.inbox.text : "等待消息\n\n长按确定键开始配网");
        lv_obj_scroll_to_y(message_box, 0, LV_ANIM_OFF);
        lv_label_set_text(hint, "上下：滚动 · 确定：已读\n长按确定：重新配网");
    }
    bsp_lvgl_unlock();
}
static void button(bsp_btn_t key, bsp_btn_ev_t event, void *user) {
    (void)user;
    if (event != BSP_BTN_CLICK && event != BSP_BTN_LONG) return;
    command cmd = {.kind = CMD_KEY, .key = key, .event = event};
    xQueueSend(commands, &cmd, 0);
}
static void wifi_event(void *arg, esp_event_base_t base, int32_t id, void *data) {
    (void)arg; (void)data;
    if (base == IP_EVENT && id == IP_EVENT_STA_GOT_IP) xEventGroupSetBits(events, LINK_UP);
    if (base == WIFI_EVENT && id == WIFI_EVENT_STA_DISCONNECTED) xEventGroupClearBits(events, LINK_UP);
}
static bool connect_wifi(const pl_config *config) {
    esp_wifi_disconnect();
    xEventGroupClearBits(events, LINK_UP);
    wifi_config_t station = {0};
    memcpy(station.sta.ssid, config->ssid, strlen(config->ssid));
    memcpy(station.sta.password, config->password, strlen(config->password));
    station.sta.threshold.authmode = config->password[0] ? WIFI_AUTH_WPA2_PSK : WIFI_AUTH_OPEN;
    station.sta.pmf_cfg.capable = true;
    if (esp_wifi_set_config(WIFI_IF_STA, &station) != ESP_OK || esp_wifi_connect() != ESP_OK) return false;
    return (xEventGroupWaitBits(events, LINK_UP, pdFALSE, pdFALSE, pdMS_TO_TICKS(20000)) & LINK_UP) != 0;
}
static bool clock_ready(void) {
    time_t now = time(NULL);
    if (now > 1767225600) return true;
    set_status("已联网，正在同步时间以验证证书");
    for (unsigned i = 0; i < 100; i++) {
        vTaskDelay(pdMS_TO_TICKS(100));
        if (time(NULL) > 1767225600) return true;
    }
    return false;
}
/* Redirects disabled: never forward a credential to an arbitrary endpoint. */
static cJSON *relay_request(const pl_config *config, const char *path, const char *body, int *status) {
    *status = 0;
    int64_t deadline = esp_timer_get_time() + 20000000;
    char url[340], auth[80];
    snprintf(url, sizeof(url), "https://%s:%u%s", config->host, config->port, path);
    esp_http_client_config_t options = {
        .url = url, .crt_bundle_attach = esp_crt_bundle_attach,
        .timeout_ms = 10000, .disable_auto_redirect = true,
        .buffer_size = 2048, .buffer_size_tx = 1024,
    };
    esp_http_client_handle_t client = esp_http_client_init(&options);
    if (!client) return NULL;
    if (config->credential[0]) {
        snprintf(auth, sizeof(auth), "Bearer %s", config->credential);
        esp_http_client_set_header(client, "Authorization", auth);
    }
    esp_http_client_set_method(client, body ? HTTP_METHOD_POST : HTTP_METHOD_GET);
    esp_http_client_set_header(client, "Content-Type", "application/json");
    size_t length = body ? strlen(body) : 0;
    char *reply = NULL;
    cJSON *json = NULL;
    if (esp_http_client_open(client, length) != ESP_OK) goto done;
    for (size_t sent = 0; sent < length;) {
        int n = esp_http_client_write(client, body + sent, length - sent);
        if (n <= 0) goto done;
        sent += n;
    }
    int64_t declared = esp_http_client_fetch_headers(client);
    *status = esp_http_client_get_status_code(client);
    if (declared < 0 || declared > REPLY_LIMIT || *status < 200 || *status >= 300) goto done;
    reply = malloc(REPLY_LIMIT + 1);
    if (!reply) goto done;
    size_t received = 0;
    while (received < REPLY_LIMIT) {
        if (esp_timer_get_time() >= deadline) goto done;
        int n = esp_http_client_read(client, reply + received, REPLY_LIMIT - received);
        if (n < 0) goto done;
        if (!n) break;
        received += n;
    }
    if (!esp_http_client_is_complete_data_received(client)) goto done;
    reply[received] = 0;
    if (pl_json_shallow(reply)) json = cJSON_ParseWithLength(reply, received + 1);
done:
    free(reply);
    esp_http_client_cleanup(client);
    memset(auth, 0, sizeof(auth));
    return json;
}
static bool acknowledge(const saved_state *state, const char *kind, int *status) {
    char body[100];
    snprintf(body, sizeof(body), "{\"message_id\":%" PRId64 ",\"state\":\"%s\"}", state->inbox.id, kind);
    cJSON *response = relay_request(&state->config, "/api/v1/device/ack", body, status);
    bool ok = cJSON_IsTrue(cJSON_GetObjectItemCaseSensitive(response, "acknowledged"));
    cJSON_Delete(response);
    return ok;
}
static void receive_text(void) {
    if (!saved.config.credential[0] || !(xEventGroupGetBits(events) & LINK_UP)) return;
    int status = 0;
    saved_state next = saved;
    if (next.inbox.id) {
        if (next.inbox.receipt == 2) return; /* Keep this message until explicitly read. */
        if (!acknowledge(&next, next.inbox.receipt == 3 ? "read" : "received", &status)) {
            if (status == 404) {
                memset(&next.inbox, 0, sizeof(next.inbox));
                if (persist(&next)) { set_status("消息权限已撤销，本地消息已移除"); render(); }
                return;
            }
            set_status(status == 401 ? "凭证失效，请重新配对" : "回执未确认，稍后重试");
            return;
        }
        pl_inbox_acknowledged(&next.inbox);
        if (persist(&next)) { set_status("已连接服务器"); render(); }
        return;
    }
    cJSON *response = relay_request(&saved.config, "/api/v1/device/inbox", NULL, &status);
    if (!response) { set_status(status == 401 ? "凭证失效，请重新配对" : "服务器暂不可用，正在重试"); return; }
    const cJSON *messages = cJSON_GetObjectItemCaseSensitive(response, "messages");
    const cJSON *first = cJSON_GetArrayItem(messages, 0);
    if (first) {
        const cJSON *id = cJSON_GetObjectItemCaseSensitive(first, "id");
        const cJSON *text = cJSON_GetObjectItemCaseSensitive(first, "text");
        if (cJSON_IsNumber(id) && id->valuedouble > 0 && id->valuedouble <= 9007199254740991.0 &&
            id->valuedouble == (double)(int64_t)id->valuedouble && cJSON_IsString(text) && strlen(text->valuestring) <= TEXT_LIMIT) {
            if (pl_inbox_accept(&next.inbox, (int64_t)id->valuedouble, text->valuestring) && persist(&next)) { set_status("新消息已保存，确定键标记已读"); render(); }
        } else set_status("消息格式不支持，未确认接收");
    } else if (cJSON_IsArray(messages)) set_status("已连接服务器，等待消息");
    else set_status("服务器返回格式错误");
    cJSON_Delete(response);
}
static void random_hex(char *out, size_t bytes) {
    static const char digits[] = "0123456789abcdef";
    for (size_t i = 0; i < bytes; i++) { unsigned v = esp_random() & 255; out[2*i] = digits[v >> 4]; out[2*i+1] = digits[v & 15]; }
    out[2*bytes] = 0;
}
static esp_err_t json_reply(httpd_req_t *req, const char *status, cJSON *value) {
    char *body = cJSON_PrintUnformatted(value);
    cJSON_Delete(value);
    if (!body) return ESP_FAIL;
    httpd_resp_set_status(req, status);
    httpd_resp_set_type(req, "application/json; charset=utf-8");
    httpd_resp_set_hdr(req, "Cache-Control", "no-store");
    esp_err_t err = httpd_resp_send(req, body, HTTPD_RESP_USE_STRLEN);
    free(body);
    return err;
}
static esp_err_t error_reply(httpd_req_t *req, const char *status, const char *message) {
    cJSON *value = cJSON_CreateObject(); cJSON_AddStringToObject(value, "error", message);
    return json_reply(req, status, value);
}
static bool local_request(httpd_req_t *req, bool host_check) {
    struct sockaddr_in address = {0}; socklen_t length = sizeof(address);
    if (!portal_active() || getsockname(httpd_req_to_sockfd(req), (struct sockaddr *)&address, &length) ||
        address.sin_addr.s_addr != inet_addr(AP_IP)) return false;
    if (!host_check) return true;
    char host[64];
    return httpd_req_get_hdr_value_str(req, "Host", host, sizeof(host)) == ESP_OK &&
        (!strcmp(host, AP_IP));
}
static esp_err_t portal_get(httpd_req_t *req) {
    if (!local_request(req, true)) return error_reply(req, "403 Forbidden", "请连接设备热点并打开 192.168.4.1");
    httpd_resp_set_type(req, "text/html; charset=utf-8");
    httpd_resp_set_hdr(req, "Cache-Control", "no-store");
    httpd_resp_set_hdr(req, "X-Frame-Options", "DENY");
    httpd_resp_set_hdr(req, "Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; form-action 'none'; frame-ancestors 'none'; base-uri 'none'");
    return httpd_resp_send(req, portal_start, HTTPD_RESP_USE_STRLEN);
}
static esp_err_t status_get(httpd_req_t *req) {
    if (!local_request(req, true)) return error_reply(req, "403 Forbidden", "请使用设备热点");
    cJSON *value = cJSON_CreateObject();
    lock();
    cJSON_AddStringToObject(value, "token", token);
    cJSON_AddStringToObject(value, "status", status_text);
    cJSON_AddStringToObject(value, "host", saved.config.host);
    cJSON_AddNumberToObject(value, "port", saved.config.port);
    cJSON_AddStringToObject(value, "ssid", saved.config.ssid);
    cJSON_AddBoolToObject(value, "configured", saved.config.credential[0] != 0);
    cJSON_AddBoolToObject(value, "busy", busy);
    cJSON *networks = cJSON_AddArrayToObject(value, "scan");
    for (unsigned i = 0; i < scan_count; i++) cJSON_AddItemToArray(networks, cJSON_CreateString(scan_names[i]));
    unlock();
    return json_reply(req, "200 OK", value);
}
static bool copy_field(cJSON *root, const char *key, char *out, size_t size) {
    cJSON *value = cJSON_GetObjectItemCaseSensitive(root, key);
    if (!cJSON_IsString(value) || strlen(value->valuestring) >= size) return false;
    strcpy(out, value->valuestring);
    return true;
}
static esp_err_t portal_post(httpd_req_t *req) {
    char supplied[33], origin[64], type[32];
    if (!local_request(req, true) ||
        httpd_req_get_hdr_value_str(req, "Origin", origin, sizeof(origin)) != ESP_OK || strcmp(origin, AP_ORIGIN) ||
        httpd_req_get_hdr_value_str(req, "Content-Type", type, sizeof(type)) != ESP_OK || strcmp(type, "application/json") ||
        httpd_req_get_hdr_value_str(req, "X-PocketLink-Token", supplied, sizeof(supplied)) != ESP_OK) {
        return error_reply(req, "403 Forbidden", "会话无效，请重新打开设备页面");
    }
    lock(); bool valid_token = strlen(supplied) == 32; unsigned diff = 0;
    for (unsigned i = 0; i < 32 && valid_token; i++) diff |= supplied[i] ^ token[i];
    bool occupied = busy; unlock();
    if (!valid_token || diff) return error_reply(req, "403 Forbidden", "会话已过期，请刷新页面");
    if (occupied) return error_reply(req, "409 Conflict", "设备正在处理，请稍候");
    if (!req->content_len || req->content_len > BODY_LIMIT) return error_reply(req, "400 Bad Request", "请求大小无效");
    char body[BODY_LIMIT + 1]; size_t used = 0;
    int64_t deadline = esp_timer_get_time() + 5000000;
    while (used < req->content_len) {
        if (esp_timer_get_time() >= deadline) return ESP_FAIL;
        int n = httpd_req_recv(req, body + used, req->content_len - used);
        if (n <= 0) return ESP_FAIL;
        used += n;
    }
    body[used] = 0;
    if (!pl_json_shallow(body) || memchr(body, 0, used) || strstr(body, "\\u0000")) return error_reply(req, "400 Bad Request", "输入包含无效字符");
    const char *end = NULL;
    cJSON *value = cJSON_ParseWithOpts(body, &end, true);
    command cmd = {.kind = !strcmp(req->uri, "/scan") ? CMD_SCAN : CMD_CONFIG};
    bool valid = cJSON_IsObject(value);
    if (cmd.kind == CMD_CONFIG) {
        cmd.config.version = PL_CONFIG_VERSION;
        cJSON *port = cJSON_GetObjectItemCaseSensitive(value, "port");
        valid = valid && cJSON_GetArraySize(value) == 5 &&
            copy_field(value, "ssid", cmd.config.ssid, sizeof(cmd.config.ssid)) &&
            copy_field(value, "password", cmd.config.password, sizeof(cmd.config.password)) &&
            copy_field(value, "host", cmd.config.host, sizeof(cmd.config.host)) &&
            copy_field(value, "code", cmd.code, sizeof(cmd.code)) &&
            cJSON_IsNumber(port) && port->valuedouble >= 1 && port->valuedouble <= 65535 && port->valuedouble == port->valueint;
        if (valid) {
            cmd.config.port = port->valueint;
            for (char *p = cmd.config.host; *p; p++) *p = tolower((unsigned char)*p);
            lock();
            bool same_server = !strcmp(cmd.config.host, saved.config.host) && cmd.config.port == saved.config.port;
            if (!cmd.code[0] && same_server) strcpy(cmd.config.credential, saved.config.credential);
            unlock();
            valid = pl_config_valid(&cmd.config) &&
                (cmd.code[0] ? pl_secret_valid(cmd.code) : cmd.config.credential[0] != 0);
        }
    } else valid = valid && cJSON_GetArraySize(value) == 0;
    cJSON_Delete(value);
    memset(body, 0, sizeof(body));
    if (!valid) return error_reply(req, "400 Bad Request", "请检查网络、密码、服务器和配对码");
    lock(); busy = true; unlock();
    if (xQueueSend(commands, &cmd, 0) != pdTRUE) {
        lock(); busy = false; unlock();
        memset(&cmd, 0, sizeof(cmd));
        return error_reply(req, "503 Service Unavailable", "设备忙，请重试");
    }
    memset(&cmd, 0, sizeof(cmd));
    return json_reply(req, "202 Accepted", cJSON_CreateObject());
}
static esp_err_t redirect_portal(httpd_req_t *req) {
    if (!local_request(req, false)) return ESP_FAIL;
    httpd_resp_set_status(req, "302 Found");
    httpd_resp_set_hdr(req, "Location", AP_ORIGIN "/");
    httpd_resp_set_hdr(req, "Cache-Control", "no-store");
    return httpd_resp_send(req, NULL, 0);
}
static esp_err_t start_http(void) {
    httpd_config_t config = HTTPD_DEFAULT_CONFIG(); config.stack_size = 8192;
    config.max_open_sockets = 3; config.lru_purge_enable = true; config.recv_wait_timeout = 3;
    config.uri_match_fn = httpd_uri_match_wildcard;
    esp_err_t result = httpd_start(&http, &config);
    if (result != ESP_OK) return result;
    const httpd_uri_t routes[] = {
        {.uri="/", .method=HTTP_GET, .handler=portal_get},
        {.uri="/status", .method=HTTP_GET, .handler=status_get},
        {.uri="/configure", .method=HTTP_POST, .handler=portal_post},
        {.uri="/scan", .method=HTTP_POST, .handler=portal_post},
        {.uri="/*", .method=HTTP_GET, .handler=redirect_portal},
    };
    for (unsigned i=0; i<sizeof(routes)/sizeof(*routes); i++) {
        result = httpd_register_uri_handler(http, &routes[i]);
        if (result != ESP_OK) { httpd_stop(http); http = NULL; return result; }
    }
    return ESP_OK;
}
/* Minimal bounded DNS responder: only one uncompressed A/IN question. */
static void dns_task(void *arg) {
    (void)arg;
    int fd = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    struct sockaddr_in local = {.sin_family = AF_INET, .sin_port = htons(53), .sin_addr.s_addr = inet_addr(AP_IP)};
    if (fd < 0 || bind(fd, (struct sockaddr *)&local, sizeof(local))) {
        if (fd >= 0) close(fd);
        vTaskDelete(NULL); return;
    }
    for (;;) {
        uint8_t packet[512]; struct sockaddr_in peer; socklen_t length = sizeof(peer);
        int n = recvfrom(fd, packet, sizeof(packet), 0, (struct sockaddr *)&peer, &length);
        if (n <= 0 || !portal_active()) continue;
        size_t response_length = pl_dns_answer(packet, (size_t)n, sizeof(packet));
        if (response_length) sendto(fd, packet, response_length, 0, (struct sockaddr *)&peer, length);
    }
}
static void start_portal(void) {
    if (portal_active()) return;
    lock(); random_hex(ap_password, 8); random_hex(token, 16); unlock();
    wifi_config_t ap = {0};
    strcpy((char *)ap.ap.ssid, ap_name); strcpy((char *)ap.ap.password, ap_password);
    ap.ap.ssid_len = strlen(ap_name); ap.ap.channel = 1; ap.ap.max_connection = 1;
    ap.ap.authmode = WIFI_AUTH_WPA2_PSK;
    /* Configure the password before starting AP beacons; never briefly expose a default open AP. */
    esp_wifi_stop();
    xEventGroupClearBits(events, LINK_UP);
    if (esp_wifi_set_mode(WIFI_MODE_APSTA) != ESP_OK || esp_wifi_set_config(WIFI_IF_AP, &ap) != ESP_OK || esp_wifi_start() != ESP_OK) {
        esp_wifi_set_mode(WIFI_MODE_STA); esp_wifi_start();
        set_status("热点启动失败，长按确定重试"); return;
    }
    lock(); ap_active = true; busy = false; scan_count = 0; unlock();
    if (start_http() != ESP_OK) {
        lock(); ap_active = false; unlock(); esp_wifi_set_mode(WIFI_MODE_STA);
        set_status("配置页面启动失败，请长按确定重试"); return;
    }
    static bool dns_started;
    if (!dns_started) dns_started = xTaskCreate(dns_task, "portal_dns", 3072, NULL, 3, NULL) == pdPASS;
    ap_deadline = esp_timer_get_time() + AP_LIFETIME_US; url_qr = false;
    set_status("连接后打开 192.168.4.1；可忽略无互联网提示"); render();
}
static void stop_portal(void) {
    lock(); ap_active = false; busy = false; memset(token, 0, sizeof(token)); unlock();
    if (http) { httpd_stop(http); http = NULL; }
    esp_wifi_set_mode(WIFI_MODE_STA);
    memset(ap_password, 0, sizeof(ap_password)); render();
}
static void scan_wifi(void) {
    set_status("正在扫描附近 2.4 GHz 网络");
    wifi_scan_config_t scan = {.show_hidden = false};
    if (esp_wifi_scan_start(&scan, true) != ESP_OK) { set_status("扫描失败，请手动输入网络名称"); return; }
    uint16_t count = 16;
    wifi_ap_record_t *records = calloc(count, sizeof(*records));
    if (!records) { esp_wifi_clear_ap_list(); set_status("内存不足，请手动输入网络名称"); return; }
    esp_err_t err = esp_wifi_scan_get_ap_records(&count, records);
    lock(); scan_count = 0;
    if (err == ESP_OK) for (unsigned i = 0; i < count; i++) {
        records[i].ssid[32] = 0;
        bool duplicate = false;
        for (unsigned j = 0; j < scan_count; j++) if (!strcmp(scan_names[j], (char *)records[i].ssid)) duplicate = true;
        if (!duplicate && records[i].ssid[0] && pl_utf8_valid((char *)records[i].ssid)) snprintf(scan_names[scan_count++], 33, "%s", records[i].ssid);
    }
    unlock(); free(records); set_status("扫描完成，请在手机上选择网络");
}
static void configure(command *cmd) {
    set_status("正在连接 Wi-Fi，请保持设备通电");
    if (!connect_wifi(&cmd->config)) { set_status("Wi-Fi 连接失败，请检查密码与 2.4 GHz 网络"); return; }
    if (!clock_ready()) { set_status("时间同步失败，请检查网络后重试"); return; }
    set_status("正在通过 HTTPS 验证服务器与设备绑定");
    int status = 0;
    cJSON *response;
    if (cmd->code[0]) {
        cJSON *body = cJSON_CreateObject();
        cJSON_AddStringToObject(body, "sn", serial); cJSON_AddStringToObject(body, "code", cmd->code);
        char *encoded = cJSON_PrintUnformatted(body); cJSON_Delete(body);
        if (!encoded) { set_status("内存不足，请稍后重试"); return; }
        response = relay_request(&cmd->config, "/api/v1/device/pair", encoded, &status);
        memset(encoded, 0, strlen(encoded)); free(encoded);
        const cJSON *credential = cJSON_GetObjectItemCaseSensitive(response, "credential");
        if (!cJSON_IsString(credential) || !pl_secret_valid(credential->valuestring)) {
            cJSON_Delete(response);
            set_status("配对未完成；响应丢失时需在管理页撤销后重配"); return;
        }
        strcpy(cmd->config.credential, credential->valuestring);
    } else {
        response = relay_request(&cmd->config, "/api/v1/device/me", NULL, &status);
        if (response && !cJSON_IsString(cJSON_GetObjectItemCaseSensitive(response, "id"))) {
            cJSON_Delete(response); response = NULL;
        }
    }
    if (!response) { set_status(status == 401 ? "凭证失效，请输入新的配对码" : "服务器连接失败，请检查地址、端口和证书"); return; }
    cJSON_Delete(response);
    saved_state next = saved;
    if (strcmp(next.config.credential, cmd->config.credential) || strcmp(next.config.host, cmd->config.host) || next.config.port != cmd->config.port) {
        memset(&next.inbox, 0, sizeof(next.inbox));
    }
    next.config = cmd->config;
    if (!persist(&next)) return;
    set_status("配置已保存，设备已绑定；热点即将关闭");
    vTaskDelay(pdMS_TO_TICKS(4000)); stop_portal();
    set_status("已连接服务器");
}
void app_main(void) {
    mutex = xSemaphoreCreateMutex(); commands = xQueueCreate(4, sizeof(command)); events = xEventGroupCreate();
    assert(mutex && commands && events);
    ESP_ERROR_CHECK(bsp_display_init()); assert(bsp_lvgl_init());
    bsp_display_backlight(70);
    assert(bsp_lvgl_lock(-1));
    lv_obj_t *screen = lv_screen_active();
    lv_obj_set_style_text_font(screen, &lv_font_source_han_sans_sc_14_cjk, 0);
    lv_obj_set_style_bg_color(screen, lv_color_hex(0xf1f6f3), 0);
    title = lv_label_create(screen); lv_obj_set_pos(title, 10, 8); lv_obj_set_width(title, 220);
    qr = lv_qrcode_create(screen); lv_qrcode_set_size(qr, 180); lv_qrcode_set_quiet_zone(qr, true);
    lv_qrcode_set_dark_color(qr, lv_color_black()); lv_qrcode_set_light_color(qr, lv_color_white());
    lv_obj_set_pos(qr, 30, 35); lv_obj_add_flag(qr, LV_OBJ_FLAG_HIDDEN);
    message_box = lv_obj_create(screen); lv_obj_set_pos(message_box, 5, 35); lv_obj_set_size(message_box, 230, 190);
    message_label = lv_label_create(message_box); lv_obj_set_width(message_label, 198);
    status_label = lv_label_create(screen); lv_obj_set_pos(status_label, 10, 225); lv_obj_set_width(status_label, 220);
    hint = lv_label_create(screen); lv_obj_set_pos(hint, 10, 267); lv_obj_set_width(hint, 220);
    lv_obj_set_style_text_font(hint, &lv_font_source_han_sans_sc_14_cjk, 0);
    bsp_lvgl_unlock();
    ESP_ERROR_CHECK(bsp_button_init(button, NULL));
    if (nvs_flash_init() != ESP_OK || nvs_flash_init_partition("pocketcfg") != ESP_OK ||
        nvs_open_from_partition("pocketcfg", "pocketlink", NVS_READWRITE, &storage) != ESP_OK) {
        set_status("存储初始化失败；未擦除任何分区"); return;
    }
    size_t size = sizeof(saved);
    esp_err_t loaded = nvs_get_blob(storage, "state", &saved, &size);
    if (loaded != ESP_ERR_NVS_NOT_FOUND && (loaded != ESP_OK || size != sizeof(saved) ||
        !pl_config_valid(&saved.config) || !saved.config.credential[0] || !pl_inbox_valid(&saved.inbox))) {
        set_status("配置版本或数据无效；未覆盖原数据"); return;
    }
    if (loaded == ESP_ERR_NVS_NOT_FOUND) {
        memset(&saved, 0, sizeof(saved)); saved.config.version = PL_CONFIG_VERSION;
        strcpy(saved.config.host, "pocketlink.mcloc.cn"); saved.config.port = 443;
    }
    ESP_ERROR_CHECK(esp_netif_init()); ESP_ERROR_CHECK(esp_event_loop_create_default());
    esp_netif_t *station = esp_netif_create_default_wifi_sta();
    esp_netif_t *ap = esp_netif_create_default_wifi_ap(); assert(station && ap);
    ESP_ERROR_CHECK(esp_event_handler_register(WIFI_EVENT, ESP_EVENT_ANY_ID, wifi_event, NULL));
    ESP_ERROR_CHECK(esp_event_handler_register(IP_EVENT, IP_EVENT_STA_GOT_IP, wifi_event, NULL));
    wifi_init_config_t wifi = WIFI_INIT_CONFIG_DEFAULT(); ESP_ERROR_CHECK(esp_wifi_init(&wifi));
    ESP_ERROR_CHECK(esp_wifi_set_storage(WIFI_STORAGE_RAM)); ESP_ERROR_CHECK(esp_wifi_set_mode(WIFI_MODE_STA));
    ESP_ERROR_CHECK(esp_wifi_start());
    uint8_t mac[6]; ESP_ERROR_CHECK(esp_read_mac(mac, ESP_MAC_WIFI_STA));
    snprintf(serial, sizeof(serial), "PL-%02X%02X%02X%02X%02X%02X", mac[0],mac[1],mac[2],mac[3],mac[4],mac[5]);
    snprintf(ap_name, sizeof(ap_name), "PocketLink-%02X%02X%02X", mac[3],mac[4],mac[5]);
    esp_sntp_setoperatingmode(SNTP_OPMODE_POLL); esp_sntp_setservername(0, "ntp.aliyun.com");
    esp_sntp_setservername(1, "pool.ntp.org"); esp_sntp_init();
    render();
    if (saved.config.ssid[0]) {
        set_status("正在连接已保存网络");
        if (!connect_wifi(&saved.config)) set_status("网络暂不可用，自动重试；长按确定配网");
    } else start_portal();
    int64_t next_poll = 0, next_reconnect = 0;
    for (;;) {
        command cmd;
        if (xQueueReceive(commands, &cmd, pdMS_TO_TICKS(100)) == pdTRUE) {
            if (cmd.kind == CMD_KEY) {
                if (cmd.key == BSP_BTN_OK && cmd.event == BSP_BTN_LONG) start_portal();
                else if (cmd.event == BSP_BTN_CLICK && ap_active && cmd.key == BSP_BTN_OK) { url_qr = !url_qr; render(); }
                else if (cmd.event == BSP_BTN_CLICK && !ap_active && saved.inbox.id && cmd.key == BSP_BTN_OK) {
                    saved_state next = saved; pl_inbox_read(&next.inbox);
                    if (persist(&next)) { set_status("已读回执等待服务器确认"); next_poll = 0; }
                } else if (cmd.event == BSP_BTN_CLICK && !ap_active && bsp_lvgl_lock(500)) {
                    if (cmd.key == BSP_BTN_UP || cmd.key == BSP_BTN_DOWN)
                        lv_obj_scroll_by(message_box, 0, cmd.key == BSP_BTN_UP ? 70 : -70, LV_ANIM_OFF);
                    bsp_lvgl_unlock();
                }
            } else if (ap_active) {
                if (cmd.kind == CMD_SCAN) scan_wifi(); else configure(&cmd);
                lock(); busy = false; unlock();
            }
            memset(&cmd, 0, sizeof(cmd));
        }
        int64_t now = esp_timer_get_time();
        if (ap_active && now >= ap_deadline) { stop_portal(); set_status("配网已超时关闭；长按确定重新开始"); }
        if (!ap_active && saved.config.ssid[0] && !(xEventGroupGetBits(events) & LINK_UP) && now >= next_reconnect) {
            set_status("网络已断开，正在重连"); connect_wifi(&saved.config); next_reconnect = esp_timer_get_time() + 15000000;
        }
        if (!ap_active && (xEventGroupGetBits(events) & LINK_UP) && now >= next_poll) {
            if (clock_ready()) receive_text();
            next_poll = esp_timer_get_time() + 5000000;
        }
    }
}
