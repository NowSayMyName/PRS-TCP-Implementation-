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
	publicAddr := net.UDPAddr{
		Port: 5000,
		IP:   net.ParseIP("127.0.0.1"),
	}

	dataPort := 5001
	publicConn, err := net.ListenUDP("udp", &publicAddr)
	if err != nil {
		fmt.Printf("Couldn't listen %v\n", err)
		return
	}
	for {
		remoteAddr, dataConn, err := acceptConnection(publicConn, dataPort)
		if err != nil {
			fmt.Printf("Couldn't accept connection \n%v\n", err)
			return
		}

		// windowSize := 1

		// transmitting := true
	}
}

/** waits for a connection and sends the public port number*/
func acceptConnection(publicConn *net.UDPConn, dataPort int) (publicAddr net.Addr, dataConn *net.UDPConn, err error) {
	buffer := make([]byte, 100)

	_, publicAddr, err = publicConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN-ACK \n%v", err)
		return nil, nil, err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "SYN" {
		fmt.Printf(string(buffer[0:3])+" %v", err)
		return nil, nil, errors.New("Could not receive SYN")
	}

	str := "SYN-ACK" + strconv.Itoa(dataPort)
	fmt.Println(str)

	_, err = publicConn.WriteTo([]byte(str), publicAddr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return nil, nil, err
	}

	_, err = publicConn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive ACK \n%v", err)
		return nil, nil, err
	}
	fmt.Printf("%s\n\n", buffer)

	if string(buffer[0:3]) != "ACK" {
		return nil, nil, errors.New("Couldn't receive ACK")
	}

	dataAddr := net.UDPAddr{
		Port: dataPort,
		IP:   net.ParseIP("127.0.0.1"),
	}

	dataConn, err = net.ListenUDP("udp", &dataAddr)
	if err != nil {
		fmt.Printf("Couldn't listen \n%v", err)
		return nil, nil, err
	}

	return publicAddr, dataConn, nil
}

func sendFile(path string, dataConn *net.UDPConn, dataAddr net.Addr) {
	f, err := os.Open(path)
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
		_, err = dataConn.WriteTo(readingBuffer[0:n], dataAddr)
		if err != nil {
			fmt.Printf("Some error %v\n", err)
			break
		}
		//Waiting for ACK
		acknowledged := false
		for !acknowledged {
			_, err = dataConn.Read(transmitionBuffer)
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
	_, err = fmt.Fprintf(dataConn, "EOT")
}
