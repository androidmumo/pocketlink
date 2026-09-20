#pragma once
#include <stdbool.h>
#include <sys/socket.h>

bool pl_portal_local_address(const struct sockaddr *address, socklen_t length);
bool pl_portal_host(const char *host);
bool pl_portal_origin(const char *origin);
