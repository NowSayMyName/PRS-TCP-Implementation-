#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/select.h>
#include <sys/time.h>
#include <netinet/in.h>

#define RCVSIZE 1024

int main (int argc, char *argv[]) {

  if (argc != 2) {
    printf("The correct way to start the program is \"./server <server_port>\"\n");
    return -1;
  }

  struct sockaddr_in adresseUDP, client;
  int port = atoi(argv[1]);
  int valid= 1;
  socklen_t alen= sizeof(client);
  char buffer[RCVSIZE];

  //create socket
  int server_desc = socket(AF_INET, SOCK_DGRAM, 0);
  if (server_desc < 0) {
    perror("Cannot create socketUDP\n");
    return -1;
  }

  setsockopt(server_desc, SOL_SOCKET, SO_REUSEADDR, &valid, sizeof(int));

  adresseUDP.sin_family= AF_INET;
  adresseUDP.sin_port= htons(port);
  adresseUDP.sin_addr.s_addr= htonl(INADDR_ANY);

  //initialize socket
  int bindResult = bind(server_desc, (struct sockaddr*) &adresseUDP, sizeof(adresseUDP));
  if (bindResult < 0) {
    perror("bindResult");
    close(server_desc);
    return -1;
  }

  // fd_set socket_set;
  // FD_ZERO (&socket_set);
  // FD_SET (server_desc, &socket_set);$

  //   if (select (FD_SETSIZE, &socket_set, NULL, NULL, NULL) < 0){
  //     perror ("select");
  //     exit (EXIT_FAILURE);
  //   }

  //   if (FD_ISSET (server_desc, &socket_set)) {

  while (1) {
    int acceptResult = acceptConnection(server_desc, serv_addr, buffer, port);
  }
  close(server_desc);
return 0;
}

int acceptConnection(int server_desc, const struct sockaddr_in serv_addr, char* buffer, int port) {
  int receiveResult = recvfrom(server_desc, buffer, RCVSIZE, 0, (struct sockaddr*) &serv_addr, sizeof(serv_addr));
  if (receiveResult < 1) {
    return -1;
  }
  if (!strcmp(buffer, "SYN")) {
    return -1;
  }
  char msg[] = "SYN-ACK ";
  sprintf(msg+7, "%d", port);
  int sendResult = sendto(server_desc, msg, sizeof(msg), 0, (struct sockaddr*)&server_desc, sizeof(server_desc));
  if (sendResult < 1) {
    return -1;
  }
  int receiveResult = recvfrom(server_desc, buffer, RCVSIZE, 0, (struct sockaddr*) &serv_addr, sizeof(serv_addr));
  if (receiveResult < 1) {
    return -1;
  }
  if (!strcmp(buffer, "ACK")) {
    return -1;
  }
  return 1;
}