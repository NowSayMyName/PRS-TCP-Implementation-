package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
)

func getArgs() (ipaddress string, portNumber int) {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: go run serverClean.go <server_address> <port_number>\n")
		os.Exit(1)
	} else {
		portNumber, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Usage: go run serverClean.go <server_address> <port_number>\n")
			os.Exit(1)
		} else {
			return os.Args[1], portNumber
		}

	}
	return "", -1
}

func main() {
	ipAddress, portNumber := getArgs()

	publicAddr := net.UDPAddr{
		Port: portNumber,
		IP:   net.ParseIP(ipAddress),
	}

	dataPort := portNumber
	publicConn, err := net.ListenUDP("udp", &publicAddr)
	fmt.Printf("Starting server on address: %s:%d\n\n", ipAddress, portNumber)
	if err != nil {
		fmt.Printf("Couldn't listen %v\n", err)
		return
	}

	for {
		dataPort++
		err := acceptConnection(publicConn, dataPort)
		if err != nil {
			fmt.Printf("Couldn't accept connection \n%v\n", err)
			return
		}
		go handleConnection(dataPort, ipAddress)
	}
}

func handleConnection(dataPort int, ipAddress string) (err error) {
	dataAddr := net.UDPAddr{
		Port: dataPort,
		IP:   net.ParseIP(ipAddress),
	}

	dataConn, err := net.ListenUDP("udp", &dataAddr)
	if err != nil {
		fmt.Printf("Couldn't listen \n%v", err)
		return err
	}

	fmt.Printf("Connection started on port %d\n", dataPort)

	windowSize := 1
	transmitting := true
	buffer := make([]byte, 100)

	_, remoteAddr, err := dataConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive path \n%v", err)
		return err
	}

	fmt.Printf("SEND FILE : %s\n", buffer)
	go sendFile(&transmitting, string(buffer), dataConn, remoteAddr, &windowSize)
	go listenOnDataPort(&transmitting, dataConn, remoteAddr, &windowSize)

	return
}

/** waits for a connection and sends the public port number*/
func acceptConnection(publicConn *net.UDPConn, dataPort int) (err error) {
	buffer := make([]byte, 100)

	_, remoteAddr, err := publicConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN \n%v", err)
		return err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "SYN" {
		fmt.Printf(string(buffer[0:3])+" %v", err)
		return errors.New("Could not receive SYN")
	}

	str := "SYN-ACK" + strconv.Itoa(dataPort)
	fmt.Println(str)

	_, err = publicConn.WriteTo([]byte(str), remoteAddr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return err
	}

	_, err = publicConn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive ACK \n%v", err)
		return err
	}
	fmt.Printf("%s\n\n", buffer)

	if string(buffer[0:3]) != "ACK" {
		return errors.New("Couldn't receive ACK")
	}

	return nil
}

/** takes a path to a file and sends it to the given address*/
func sendFile(connected *bool, path string, dataConn *net.UDPConn, dataAddr net.Addr, windowSize *int) (err error) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error creating file %v\n", err)
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	readingBuffer := make([]byte, 100)

	for {
		//Reading the file
		fmt.Println("[   NEW PACKET   ]")
		n, err := r.Read(readingBuffer)
		if err != nil {
			fmt.Printf("Error reading file %v\n", err)
			return err
		}

		for *windowSize == 0 {
		}

		//Sending fragment
		fmt.Println(string(readingBuffer[0:n]))
		_, err = dataConn.WriteTo(readingBuffer[0:n], dataAddr)
		if err != nil {
			fmt.Printf("Error sending packet %v\n", err)
			return err
		}
		*windowSize--
	}
	_, err = dataConn.WriteTo([]byte("FIN"), dataAddr)
	if err != nil {
		fmt.Printf("Error sending FIN")
	}

	return
}

func listenOnDataPort(connected *bool, dataConn *net.UDPConn, dataAddr net.Addr, windowSize *int) (err error) {
	transmitionBuffer := make([]byte, 100)

	for *connected {
		_, err := dataConn.Read(transmitionBuffer)
		if err != nil {
			fmt.Printf("Error reading packets %v\n", err)
			return err
		}

		if string(transmitionBuffer[0:3]) == "ACK" {
			*windowSize++
		}
	}
	return
}
