package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

func main() {
	addr := net.UDPAddr{
		Port: 5000,
		IP:   net.ParseIP("127.0.0.1"),
	}
	controlPort := 5001
	controlConn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Couldn't listen %v\n", err)
		return
	}
	for {
		controlAddr, dataConn, err := acceptConnection(controlConn, controlPort)
		if err != nil {
			fmt.Printf("Couldn't accept connection \n%v", err)
			return
		}

		transmitting := true
		transmitionBuffer := make([]byte, 100)
		for transmitting {
			_, err = dataConn.Read(transmitionBuffer)
			if err != nil {
				fmt.Printf("Couldn't read data \n%v", err)
			}
			fmt.Println(string(transmitionBuffer))
			runes := []rune(string(transmitionBuffer))
			if string(transmitionBuffer) != "" {
				_, err = controlConn.WriteTo([]byte("ACK"), controlAddr)
				if err != nil {
					fmt.Printf("Couldn't write to control \n%v", err)
				}
			}

			if string(runes[0:3]) == "EOT" {
				transmitting = false
				break
			}
		}
	}
}

/** waits for a connection and sends the control port number*/
func acceptConnection(controlConn *net.UDPConn, dataPort int) (controlAddr net.Addr, dataConn *net.UDPConn, err error) {
	buffer := make([]byte, 100)

	_, controlAddr, err = controlConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN-ACK \n%v", err)
		return nil, nil, err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "SYN" {
		fmt.Printf(string(buffer[0:3])+" %v", err)
		return nil, nil, errors.New("Could not receive SYN")
	}

	str := "SYN-ACK " + strconv.Itoa(dataPort)
	fmt.Println(str)

	_, err = controlConn.WriteTo([]byte(str), controlAddr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return nil, nil, err
	}

	_, err = controlConn.Read(buffer)
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

	return controlAddr, dataConn, nil
}

func receiveData(controlConn *net.UDPConn, controlAddr *net.Addr, dataConn *net.UDPConn) (err error) {
	buffer := make([]byte, 100)

	for {
		_, err := dataConn.Read(buffer)

		if err != nil {
			fmt.Printf("Coulnd't read data \n%v", err)
			return err
		}

		if string(buffer) != "" {
			_, err = controlConn.WriteTo([]byte("ACK"), *controlAddr)
			if err != nil {
				fmt.Printf("Couldn't write to control \n%v", err)
				return err
			}
		}
	}
}
