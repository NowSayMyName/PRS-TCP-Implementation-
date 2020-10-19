#include <arpa/inet.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define RCVSIZE 1024

int createSocket(struct sockaddr_in server, char * address, int port) {
  int valid= 1;
  int server_desc = socket(AF_INET, SOCK_DGRAM, 0);

  if (server_desc < 0) {
    perror("Cannot create socketUDP\n");
    return -1;
  }

  int optResult = setsockopt(server_desc, SOL_SOCKET, SO_REUSEADDR, &valid, sizeof(int));
  if (optResult > 0) {
    perror("optResult\n");
    return -2;
  }

  server.sin_family= AF_INET;
  server.sin_port= htons(port);
  if (address == NULL) {
    server.sin_addr.s_addr= htonl(INADDR_ANY);

    int bindResult = bind(server_desc, (struct sockaddr*) &server, sizeof(server));
    if (bindResult < 0) {
      perror("bindResult");
      close(server_desc);
      return -4;
    }
  } else {
    int addressResult = inet_aton(address, &server.sin_addr);
    if (addressResult <= 0) {
      printf("Invalid address");
      return -3;
    }
  }
  return server_desc;
}

char *substring(char *src,int pos,int len) { 
  char *dest=NULL;                        
  if (len>0) {                  
    /* allocation et mise à zéro */          
    dest = calloc(len+1, 1);      
    /* vérification de la réussite de l'allocation*/  
    if(NULL != dest) {
        strncat(dest,src+pos,len);            
    }
  }                                       
  return dest;                            
}

/** renvoie le port utilisé par le serveur pour les messages de controles, sinon des valeurs <0*/
int connectionToServer(int server_desc, struct sockaddr_in serv_addr, char* buffer, int buffer_size) {
  socklen_t alen = sizeof(serv_addr);
  sprintf(buffer, "%s", "SYN");
  printf("%s\n", buffer);

  int sendResult = sendto(server_desc, buffer, sizeof(buffer), 0, (struct sockaddr*) &serv_addr, alen);
  if (sendResult < 1) {
    return -1;
  }

  int receiveResult = recvfrom(server_desc, buffer, buffer_size, 0, (struct sockaddr*) &serv_addr, &alen);
    if (receiveResult < 1) {
    return -2;
  }
  printf("%s\n", buffer);

  if (!strcmp(substring(buffer, 0, 9), "SYN-ACK ")) {
    return -3;
  }

  sprintf(buffer, "%s", "ACK");
  printf("%s\n", buffer);
  
  sendResult = sendto(server_desc, buffer, sizeof(buffer), 0, (struct sockaddr*) &serv_addr, alen);
  if (sendResult < 1) {
    return -4;
  }

  return atoi(substring(buffer, 8, 4));
}

/** waits for a connection and sends the control port number*/
int acceptConnection(int server_desc, struct sockaddr_in client_addr, int port, char* buffer, int buffer_size) {
  socklen_t alen= sizeof(client_addr);
  int receiveResult = recvfrom(server_desc, buffer, buffer_size, 0, (struct sockaddr*) &client_addr, &alen);
  printf("%s\n", buffer);
  if (receiveResult < 1) {
    return -1;
  }
  if (!strcmp(buffer, "SYN\n")) {
    return -2;
  }
  sprintf(buffer, "%s", "SYN-ACK ");
  sprintf(buffer+8, "%d", port);
  printf("%s\n", buffer);

  int sendResult = sendto(server_desc, buffer, 12, 0, (struct sockaddr*)&client_addr, alen);
  if (sendResult < 1) {
    return -3;
  }

  receiveResult = recvfrom(server_desc, buffer, buffer_size, 0, (struct sockaddr*) &client_addr, &alen);
  printf("%s\n", buffer);
  if (receiveResult < 1) {
    return -4;
  }
  if (!strcmp(buffer, "ACK\n")) {
    return -5;
  }
  return 1;
}

/** fragment and send a file**/
int sendFrag(char filepath[],int buffer_size, int server_desc, const struct sockaddr_in serv_addr){
    //char filepath[] = "/home/yrouxel/Téléchargements/test.pdf";
    unsigned char buffer[buffer_size];

    FILE *file;
    file = fopen(filepath, "rb");

    while(feof(file) == 0){
        fread(buffer, buffer_size, 1, file);
        int sendResult = sendto(server_desc, buffer, sizeof(buffer), 0, (struct sockaddr*) &serv_addr, sizeof(serv_addr));
    }
    fclose(file);
    return 0;
}