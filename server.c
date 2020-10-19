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
  char buffer[RCVSIZE];

  int server_desc = createSocket(server, NULL, atoi(argv[1]));
  if (server_desc < 0) {
    printf("socket error :%d\n", server_desc);
    return -1;
  }

  int dataport = port + 1;
  while (1) {
    int acceptResult = acceptConnection(server_desc, client_addr, dataport, buffer, RCVSIZE);
    
    if (acceptResult < 0) {
      printf("Connexion error : %d\n", acceptResult);
      return -1;
    }
    printf("RECEIVED : %s \n",acceptResult);
    int forkResult = fork();
    if (forkResult == 0) {
        /*setsockopt(server_desc, SOL_SOCKET, SO_REUSEADDR, &valid, sizeof(int));

        clientHandler.sin_family= AF_INET;
        clientHandler.sin_port= htons(dataport);
        clientHandler.sin_addr.s_addr= htonl(INADDR_ANY);*/
      //talk on data port
    } else if (forkResult > 0) {
      dataport++;
    } else {
      printf("FORK ERROR :%d\n", forkResult);
    }

  }
  close(server_desc);
  return 0;
}