#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

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
  char filepath[] = "/home/mbonnefoy/Téléchargements/test.pdf";

  FILE *file;
  char *file_data;
  file = fopen(filepath, "rb");

  //Allocate memory
  file_data=(char *)malloc(sizeof(file_data));

  //Read file contents into buffer
  fread(file_data, sizeof(file_data), 1, file);
  fclose(file);
  printf(file_data)



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
  connect(server_desc, serv_addr);

  close(server_desc);
  return 0;
}

int connect(int server_desc, const struct sockaddr_in serv_addr) {

  char buffer[] = "SYN";
  int sendResult = sendto(server_desc, buffer, sizeof(buffer), 0, (struct sockaddr*) &serv_addr, sizeof(serv_addr));
  int received = 0;

  while (received == 0) {
    int bufferResult = recvfrom(server_desc, buffer, RCVSIZE, 0, (struct sockaddr*) &serv_addr, sizeof(serv_addr));
    strncat(buffer, buffer, 7);
    char data_port = buffer;
    
    if (strcmp(buffer, "SYN-ACK")) {
      received = 1;
      char buffer[] = "ACK";
      int sendResult = sendto(server_desc, buffer, sizeof(buffer), 0, (struct sockaddr*) &serv_addr, sizeof(serv_addr));
    }else{
      return -1;
    }
  }

}

// int sendMessage(const void* buffer, int server_desc, const struct sockaddr_in serv_addr*) {
//   return sendto(server_desc, buffer, sizeof(buffer), 0, (struct sockaddr*)&serv_addr, sizeof(serv_addr));
// }