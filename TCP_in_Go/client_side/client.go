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
	address := "127.0.0.1"
	controlPort := "5000"
	conn, _, err := connectionToServer(address + ":" + controlPort)
	if err != nil {
		fmt.Printf("Could not connect %v", err)
	}
	defer conn.Close()

	f, err := os.Open("stuff/stuff/test123.txt")
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

			if string(runes[0:3]) == "ACK" {
				acknowledged = true

			}
		}
	}
	_, err = fmt.Fprintf(conn, "EOT")

}

/** renvoie le port utilisé par le serveur pour les messages de controles*/
func connectionToServer(address string) (conn *net.UDPConn, controlPort int, err error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		fmt.Printf("Could not resolve address \n%v", err)
		return nil, 0, err
	}

	conn, err = net.DialUDP("udp", nil, addr)
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

	if string(buffer[0:8]) != "SYN-ACK " {
		fmt.Printf(string(buffer[0:8]))
		return nil, 0, errors.New("Could not receive SYN-ACK")
	}

	fmt.Printf("ACK\n")
	_, err = fmt.Fprintf(conn, "ACK")
	if err != nil {
		fmt.Printf("Could not send ACK \n%v", err)
		return nil, 0, err
	}

	controlPort, _ = strconv.Atoi(string(buffer[8:12]))
	return conn, controlPort, nil
}

func readControlPort(conn *net.UDPConn, windowSize *int) (err error) {
	for {
		buffer := make([]byte, 100)
		_, err := conn.Read(buffer)

		if err != nil {
			fmt.Printf("Reading error \n%v", err)
			return err
		}

		if string(buffer[0:3]) == "ACK" {
			*windowSize++
		}
	}
}
