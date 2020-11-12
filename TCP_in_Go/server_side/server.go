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
		err := acceptConnection(ser, 5001)
		if err != nil {
			fmt.Printf("Some error %v\n", err)
			return
		}
	}
}

/** waits for a connection and sends the control port number*/
func acceptConnection(conn *net.UDPConn, controlPort int) (err error) {
	buffer := make([]byte, 100)

	_, addr, err := conn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN-ACK \n%v", err)
		return err
	}
	fmt.Printf("%s\n", buffer)

	runes := []rune(string(buffer))

	if string(runes[0:3]) != "SYN" {
		fmt.Printf(string(runes[0:3])+" %v", err)
		return errors.New("Could not receive SYN")
	}

	str := "SYN-ACK " + strconv.Itoa(controlPort)
	fmt.Println(str)

	_, err = conn.WriteTo([]byte(str), addr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return err
	}

	_, err = conn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive ACK \n%v", err)
		return err
	}
	fmt.Printf("%s\n", buffer)

	runes = []rune(string(buffer))

	if string(runes[0:3]) != "ACK" {
		return errors.New("Couldn't receive ACK")
	}
	return
}
