package main

import (
	"errors"
	"fmt"
	"net"
)

func main() {
	address := "127.0.0.1"
	controlPort := "5000"

	_, _, err := connectionToServer(address, controlPort)
	if err != nil {
		fmt.Printf("Connection error \n%v", err)
	}
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

	if string(buffer[0:7]) != "SYN-ACK" {
		fmt.Printf(string(buffer[0:7]))
		return nil, nil, errors.New("Could not receive SYN-ACK")
	}

	fmt.Printf("ACK\n\n")
	_, err = fmt.Fprintf(controlConn, "ACK")
	if err != nil {
		fmt.Printf("Could not send ACK \n%v", err)
		return nil, nil, err
	}

	addr, err = net.ResolveUDPAddr("udp", address+":"+string(buffer[7:11]))
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
