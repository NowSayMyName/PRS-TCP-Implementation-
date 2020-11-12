package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	address := "127.0.0.1"
	controlPort := "5000"
	controlConn, dataConn, err := connectionToServer(address, controlPort)
	if err != nil {
		fmt.Printf("Could not connect %v", err)
	}
	defer controlConn.Close()
	defer dataConn.Close()

	f, err := os.Open("C:/Users/Melvil/go/src/github.com/MelvilB/PRS/PRS_TCP_Implementation/stuff/stuff/test123.txt")
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		log.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			fmt.Printf("Some error %v\n", err)
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
		_, err = fmt.Fprintf(dataConn, string(readingBuffer[0:n]))
		if err != nil {
			fmt.Printf("Some error %v\n", err)
			break
		}
		//Waiting for ACK
		acknowledged := false
		for !acknowledged {
			_, err = controlConn.Read(transmitionBuffer)
			if err != nil {
				fmt.Printf("Some error %v\n", err)
				log.Fatal(err)
			}
			fmt.Printf("waiting for ACK  \n")

			if string(transmitionBuffer[0:3]) == "ACK" {
				acknowledged = true
			}
		}
	}
	_, err = fmt.Fprintf(controlConn, "EOT")

}

/** renvoie le port utilisé par le serveur pour les messages de controles*/
func connectionToServer(address string, controlPort string) (controlConn *net.UDPConn, dataConn *net.UDPConn, err error) {
	addr, err := net.ResolveUDPAddr("udp", address+":"+controlPort)
	if err != nil {
		fmt.Printf("Could not resolve address \n%v", err)
		return nil, nil, err
	}

	controlConn, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Could not dial \n%v", err)
		return nil, nil, err
	}

	buffer := make([]byte, 100)

	fmt.Printf("SYN\n")
	_, err = fmt.Fprintf(controlConn, "SYN")
	if err != nil {
		fmt.Printf("Could not send SYN \n%v", err)
		return nil, nil, err
	}

	_, err = controlConn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN-ACK \n%v", err)
		return nil, nil, err
	}

	fmt.Printf("%s\n", buffer)

	if string(buffer[0:8]) != "SYN-ACK " {
		fmt.Printf(string(buffer[0:8]))
		return nil, nil, errors.New("Could not receive SYN-ACK")
	}

	fmt.Printf("ACK\n\n")
	_, err = fmt.Fprintf(controlConn, "ACK")
	if err != nil {
		fmt.Printf("Could not send ACK \n%v", err)
		return nil, nil, err
	}

	addr, err = net.ResolveUDPAddr("udp", address+":"+string(buffer[8:12]))
	if err != nil {
		fmt.Printf("Could not resolve address \n%v", err)
		return nil, nil, err
	}

	dataConn, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Could not dial \n%v", err)
		return nil, nil, err
	}

	return controlConn, dataConn, nil
}

func readControlPort(controlConn *net.UDPConn, windowSize *int) (err error) {
	buffer := make([]byte, 100)

	for {
		_, err := controlConn.Read(buffer)

		if err != nil {
			fmt.Printf("Reading error \n%v", err)
			return err
		}

		if string(buffer[0:3]) == "ACK" {
			*windowSize++
		}
	}
}
