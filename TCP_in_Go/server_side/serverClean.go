package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
)

func main() {
	publicAddr := net.UDPAddr{
		Port: 5000,
		IP:   net.ParseIP("192.168.0.12"),
	}

	dataPort := 5001
	publicConn, err := net.ListenUDP("udp", &publicAddr)
	if err != nil {
		fmt.Printf("Couldn't listen %v\n", err)
		return
	}

	for {
		dataConn, err := acceptConnection(publicConn, dataPort)
		if err != nil {
			fmt.Printf("Couldn't accept connection \n%v\n", err)
			return
		}

		fmt.Printf("HERE\n")

		go handleConnection(dataConn)
	}
}

func handleConnection(dataConn *net.UDPConn) (err error) {
	windowSize := 1
	transmitting := true
	buffer := make([]byte, 100)

	_, remoteAddr, err := dataConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive path \n%v", err)
		return err
	}

	fmt.Printf("%s\n", buffer)
	go sendFile(&transmitting, string(buffer), dataConn, remoteAddr, &windowSize)
	go listenOnDataPort(&transmitting, dataConn, remoteAddr, &windowSize)

	return
}

/** waits for a connection and sends the public port number*/
func acceptConnection(publicConn *net.UDPConn, dataPort int) (dataConn *net.UDPConn, err error) {
	buffer := make([]byte, 100)

	_, remoteAddr, err := publicConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN \n%v", err)
		return nil, err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "SYN" {
		fmt.Printf(string(buffer[0:3])+" %v", err)
		return nil, errors.New("Could not receive SYN")
	}

	str := "SYN-ACK" + strconv.Itoa(dataPort)
	fmt.Println(str)

	_, err = publicConn.WriteTo([]byte(str), remoteAddr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return nil, err
	}

	_, err = publicConn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive ACK \n%v", err)
		return nil, err
	}
	fmt.Printf("%s\n\n", buffer)

	if string(buffer[0:3]) != "ACK" {
		return nil, errors.New("Couldn't receive ACK")
	}

	dataAddr := net.UDPAddr{
		Port: dataPort,
		IP:   net.ParseIP("192.168.0.12"),
	}

	dataConn, err = net.ListenUDP("udp", &dataAddr)
	if err != nil {
		fmt.Printf("Couldn't listen \n%v", err)
		return nil, err
	}

	return dataConn, nil
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
