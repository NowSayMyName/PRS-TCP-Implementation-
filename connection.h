#include <arpa/inet.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int acceptConnection(int server_desc, struct sockaddr_in client_addr, char* buffer, int port);
int connectionToServer(int server_desc, struct sockaddr_in serv_addr, char* buffer);

