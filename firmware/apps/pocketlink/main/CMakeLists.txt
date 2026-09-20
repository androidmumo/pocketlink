idf_component_register(SRCS "main.c" INCLUDE_DIRS "."
    REQUIRES pocketlink_font bsp pocketlink_config pocketlink_ota esp_wifi esp_event esp_netif nvs_flash
    esp_http_server esp_http_client mbedtls json esp_timer lwip
    EMBED_TXTFILES "portal.html")
