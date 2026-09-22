idf_component_register(SRCS "pocketlink_voice.c" INCLUDE_DIRS "." REQUIRES bsp pocketlink_config esp_websocket_client esp_timer mbedtls json)
