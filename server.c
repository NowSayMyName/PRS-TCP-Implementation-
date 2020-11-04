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

  struct sockaddr_in server_ctrl, server_data, client_addr;
  int port = atoi(argv[1]);
  int data_port = port + 1;

  char buffer[RCVSIZE];

  //create socket
  int server_desc_ctrl = createSocket(server_ctrl, NULL, port);
  int server_desc_data = createSocket(server_data, NULL, data_port);
  FILE *file;
  file = fopen("/home/mbonnefoy/Téléchargements/testResult.pdf", "w");
  if(file == NULL)
  {
    printf("Unable to create file.\n");
    return -1;
  }

  while (1) {
    int acceptResult = acceptConnection(server_desc_ctrl, client_addr, data_port, buffer, RCVSIZE);
    if (acceptResult < 0) {
      printf("Connexion error : %d\n", acceptResult);
      return -1;
    }  
    int forkResult = fork();
    if (forkResult == 0) {
      //talk on data port
      int transmitting = 1;
      while (transmitting) {
        int receiveResult = recvfrom(server_desc_data, buffer, RCVSIZE, 0, (struct sockaddr*) &client_addr, sizeof(client_addr));
        if (receiveResult < 1) {
          return -2;
        }
        if(strcmp(buffer,"[(DATA END)] ")!=0){
          transmitting = 0;
        }else{
          fwrite(buffer,RCVSIZE,1,file);
        }
      }
      fclose(file);
    } else if (forkResult > 0) {

    } else if (forkResult < 0) {
      printf("FORK ERROR :%d\n", forkResult);
    }
  }
  
  close(server_desc_ctrl);
  close(server_desc_data);

  return 0;
}