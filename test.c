  
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

int main (int argc, char *argv[]) {

    int sendFrag(char filepath[],int buffer_size, int server_desc, const struct sockaddr_in serv_addr){
        //char filepath[] = "/home/mbonnefoy/Téléchargements/test.pdf";
        unsigned char buffer[buffer_size];

        FILE *file;
        file = fopen(filepath, "rb");

        while(feof(file) == 0){
            fread(buffer, buffer_size, 1, file);
            char buffer[] = "ACK";
            int sendResult = sendto(server_desc, buffer, sizeof(buffer), 0, (struct sockaddr*) &serv_addr, sizeof(serv_addr));
        }
        fclose(file);
        return 0;
    }
}