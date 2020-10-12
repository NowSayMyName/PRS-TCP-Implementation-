all: main

mainServer: server.o
	gcc -Wall server.o -o mainServer

server.o: server.c
	gcc -Wall server.c -I./include -c -o server.o

mainClient: client.o
	gcc -Wall client.o -o mainClient

client.o: client.c
	gcc -Wall client.c -I./include -c -o client.o

clean:
	rm -f *.o