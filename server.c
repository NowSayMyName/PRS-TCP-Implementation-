#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/select.h>
#include <sys/time.h>
#include <netinet/in.h>

#include "server.h"

#define RCVSIZE 1024

int main (int argc, char *argv[]) {

  if (argc != 2) {
    printf("The correct way to start the program is \"./server <server_port>\"\n");
    return -1;
  }

  struct sockaddr_in address, client_addr;
  int port = atoi(argv[1]);
  int valid= 1;
  char buffer[RCVSIZE];

  //create socket
  int server_desc = socket(AF_INET, SOCK_DGRAM, 0);
  if (server_desc < 0) {
    perror("Cannot create socketUDP\n");
    return -1;
  }

  setsockopt(server_desc, SOL_SOCKET, SO_REUSEADDR, &valid, sizeof(int));

  address.sin_family= AF_INET;
  address.sin_port= htons(port);
  address.sin_addr.s_addr= htonl(INADDR_ANY);

  //initialize socket
  int bindResult = bind(server_desc, (struct sockaddr*) &address, sizeof(address));
  if (bindResult < 0) {
    perror("bindResult");
    close(server_desc);
    return -1;
  }

  while (1) {
    int acceptResult = acceptConnection(server_desc, client_addr, buffer, port);
    if (acceptResult < 0) {
      printf("Connexion error : %d", acceptResult);
    }
  }
  close(server_desc);
return 0;
}

int acceptConnection(int server_desc, const struct sockaddr_in client_addr, char* buffer, int port) {
  socklen_t alen= sizeof(client_addr);
  int receiveResult = recvfrom(server_desc, buffer, RCVSIZE, 0, (struct sockaddr*) &client_addr, &alen);
  if (receiveResult < 1) {
    return -1;
  }
  if (!strcmp(buffer, "SYN")) {
    return -2;
  }
  sprintf(buffer, "%d", "SYN");
  sprintf(buffer+7, "%d", port);
  int sendResult = sendto(server_desc, buffer, sizeof(buffer), 0, (struct sockaddr*)&client_addr, &alen);
  if (sendResult < 1) {
    return -3;
  }
  int receiveResult = recvfrom(server_desc, buffer, RCVSIZE, 0, (struct sockaddr*) &client_addr, &alen);
  if (receiveResult < 1) {
    return -4;
  }
  if (!strcmp(buffer, "ACK")) {
    return -5;
  }
  return 1;
}