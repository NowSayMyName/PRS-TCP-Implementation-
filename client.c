#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#include "connection.h"

#define RCVSIZE 1024

int main (int argc, char *argv[]) {
  if (argc != 3) {
    printf("The correct way to start the program is \"./client <server_ip> <server_control_port>\"\n");
    return -1;
  }

  struct sockaddr_in serv_addr;
  int addressResult = inet_aton(argv[1], &serv_addr.sin_addr);

  if (addressResult <= 0) {
    printf("Invalid address");
    return -1;
  }

  int control_port = atoi(argv[2]);

  int valid = 1;
  char buffer[RCVSIZE];

  //create socket
  int server_desc = socket(AF_INET, SOCK_DGRAM, 0);

  // handle error
  if (server_desc < 0) {
    perror("cannot create socket\n");
    return -1;
  }

  setsockopt(server_desc, SOL_SOCKET, SO_REUSEADDR, &valid, sizeof(int));

  serv_addr.sin_family= AF_INET;
  serv_addr.sin_port= htons(control_port);
  int connectResult = connectionToServer(server_desc, serv_addr, buffer, RCVSIZE);

  if (connectResult < 0) {
    printf("Connexion error : %d\n", connectResult);
  } else {
    serv_addr.sin_port = htons(connectResult);
    printf("Data port : %d\n", connectResult);
  }
  char filepath[] = "/home/mbonnefoy/Téléchargements/test.pdf";
  int buffer_size = 200;
  int fragResult = sendFrag(filepath, buffer_size, server_desc, serv_addr);
  if(fragResult < 0){
    printf("Fragmentation error : %d\n", fragResult);
  }
  close(server_desc);
  return 0;
}