#include <arpa/inet.h>

int acceptConnection(int server_desc, const struct sockaddr_in client_addr, char* buffer, int port) {
