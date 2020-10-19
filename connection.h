#include <arpa/inet.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int createSocket(struct sockaddr_in server, char * address, int port);
int acceptConnection(int server_desc, struct sockaddr_in client_addr, int port, char* buffer, int buffer_size);
int connectionToServer(int server_desc, struct sockaddr_in serv_addr, char* buffer, int buffer_size);
char *substring(char *src,int pos,int len);
int sendFrag(char filepath[],int buffer_size, int server_desc, const struct sockaddr_in serv_addr);

