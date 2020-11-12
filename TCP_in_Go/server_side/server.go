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
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}
	for {
		dataConn, err := acceptConnection(ser, 5001)
		if err != nil {
			fmt.Printf("Some error %v\n", err)
			return
		}

		transmitting := true
		transmitionBuffer := make([]byte, 100)
		for transmitting {
			_, err = dataConn.Read(transmitionBuffer)
			if err != nil {
				fmt.Printf("Some error %v\n", err)
			}
			fmt.Println(string(transmitionBuffer))
			runes := []rune(string(transmitionBuffer))
			if string(transmitionBuffer) != "" {
				_, err = controlConn.WriteTo([]byte(string("ACK")), &addr)
				if err != nil {
					fmt.Printf("Some error %v\n", err)
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
func acceptConnection(controlConn *net.UDPConn, dataPort int) (dataConn *net.UDPConn, err error) {
	buffer := make([]byte, 100)

	_, addr, err := controlConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN-ACK \n%v", err)
		return nil, err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "SYN" {
		fmt.Printf(string(buffer[0:3])+" %v", err)
		return nil, errors.New("Could not receive SYN")
	}

	str := "SYN-ACK " + strconv.Itoa(dataPort)
	fmt.Println(str)

	_, err = controlConn.WriteTo([]byte(str), addr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return nil, err
	}

	_, err = controlConn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive ACK \n%v", err)
		return nil, err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "ACK" {
		return nil, errors.New("Couldn't receive ACK")
	}

	dataAddr := net.UDPAddr{
		Port: dataPort,
		IP:   net.ParseIP("127.0.0.1"),
	}

	dataConn, err = net.ListenUDP("udp", &dataAddr)

	return dataConn, nil
}
