#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/select.h>
#include <sys/time.h>
#include <netinet/in.h>

#include "connection.h"

#define RCVSIZE 1024

int main (int argc, char *argv[]) {
  if (argc != 2) {
    printf("The correct way to start the program is \"./server <server_port>\"\n");
    return -1;
  }

  struct sockaddr_in server, clientHandler, client_addr;
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

  server.sin_family= AF_INET;
  server.sin_port= htons(port);
  server.sin_addr.s_addr= htonl(INADDR_ANY);

  //initialize socket
  int bindResult = bind(server_desc, (struct sockaddr*) &server, sizeof(server));
  if (bindResult < 0) {
    perror("bindResult");
    close(server_desc);
    return -1;
  }

  int dataport = port + 1;
  while (1) {
    int acceptResult = acceptConnection(server_desc, client_addr, dataport, buffer, RCVSIZE);
    
    if (acceptResult < 0) {
      printf("Connexion error : %d\n", acceptResult);
      return -1;
    }
    printf("RECEIVED : %s\n",buffer);

    /*int forkResult = fork();
    if (forkResult == 0) {
      //talk on data port
    } else if (forkResult > 0) {
      dataport++;
    } else {
      printf("FORK ERROR :%d\n", forkResult);
    }*/

  }
  close(server_desc);
  return 0;
}