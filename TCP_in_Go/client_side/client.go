package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

func main() {
	conn, port, err := connectionToServer("127.0.0.1", "5000")
	if err != nil {
		fmt.Printf("Could not connect %v", err)
	}
	defer conn.Close()
	fmt.Printf("Could not connect %v", port)
}

/** renvoie le port utilisé par le serveur pour les messages de controles*/
func connectionToServer(addr string, port string) (conn net.Conn, controlPort int, err error) {
	conn, err = net.Dial("udp", addr+":"+port)
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
	runes := []rune(string(buffer))

	if string(runes[0:7]) != "SYN-ACK " {
		fmt.Printf(string(runes[0:7])+" %v", err)
		return nil, 0, errors.New("Could not receive SYN-ACK")
	}

	fmt.Printf("ACK\n")
	_, err = fmt.Fprintf(conn, "ACK")
	if err != nil {
		fmt.Printf("Could not send ACK \n%v", err)
		return nil, 0, err
	}

	controlPort, _ = strconv.Atoi(string(runes[9:12]))
	fmt.Printf(string(runes[9:12]))
	return conn, controlPort, nil
}
