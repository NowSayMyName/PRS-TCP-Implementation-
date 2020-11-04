package main

import (
	"fmt"
	"net"
)

func main() {
	p := make([]byte, 2048)
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
