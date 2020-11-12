package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

func main() {
	conn, port, err := connectionToServer("127.0.0.1", "5000")
	if err != nil {
		fmt.Printf("Could not connect %v", err)
	}
	defer conn.Close()
	fmt.Printf("Could not connect %v", port)

	f, err := os.Open("C:/Users/Melvil/Desktop/INSA/PRS/PRS_TCP_Implementation_/TCP_in_Go/test.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	r := bufio.NewReader(f)
	readingBuffer := make([]byte, 100)
	transmitionBuffer := make([]byte, 100)

	for {
		//Reading the file
		fmt.Println("[   NEW PACKET   ]")
		n, err := r.Read(readingBuffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading file:", err)
			break
		}
		//Sending fragment
		fmt.Println(string(readingBuffer[0:n]))
		_, err = fmt.Fprintf(conn, string(readingBuffer[0:n]))

		//Waiting for ACK
		acknowledged := false
		for !acknowledged {
			_, err = conn.Read(transmitionBuffer)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("%s\n", transmitionBuffer)
			runes := []rune(string(transmitionBuffer))

			if string(runes[0:3]) == "ACK " {
				acknowledged = true

			}
		}
	}
}

/** renvoie le port utilisé par le serveur pour les messages de controles*/
func connectionToServer(addr string, port string) (conn net.Conn, controlPort int, err error) {
	conn, err = net.Dial("udp", addr+":"+port)
	if err != nil {
		fmt.Printf("Could not dial \n%v", err)
		return nil, 0, err
	}

	buffer := make([]byte, 100)

	fmt.Printf("SYN\n")
	_, err = fmt.Fprintf(conn, "SYN")
	if err != nil {
		fmt.Printf("Could not send SYN \n%v", err)
		return nil, 0, err
	}

	_, err = conn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN-ACK \n%v", err)
		return nil, 0, err
	}

	fmt.Printf("%s\n", buffer)
	runes := []rune(string(buffer))

	if string(runes[0:7]) != "SYN-ACK " {
		fmt.Printf(string(runes[0:7])+" %v", err)
		return nil, 0, errors.New("Could not receive SYN-ACK")
	}

	fmt.Printf("ACK\n")
	_, err = fmt.Fprintf(conn, "ACK")
	if err != nil {
		fmt.Printf("Could not send ACK \n%v", err)
		return nil, 0, err
	}

	controlPort, _ = strconv.Atoi(string(runes[9:12]))
	fmt.Printf(string(runes[9:12]))
	return conn, controlPort, nil
}
