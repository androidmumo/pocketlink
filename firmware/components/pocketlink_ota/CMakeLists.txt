idf_component_register(SRCS "pocketlink_ota.c" "ota_boot.c" "ota_manifest.c" INCLUDE_DIRS "."
    REQUIRES app_update esp_http_client mbedtls json nvs_flash pocketlink_config esp_timer
    EMBED_TXTFILES "trust.pem")
