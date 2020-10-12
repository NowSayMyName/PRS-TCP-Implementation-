all: main

mainTest: test.o
	gcc -Wall test.o -o mainTest

test.o: test.c
	gcc -Wall test.c -I./include -c -o test.o

clean:
	rm -f *.o