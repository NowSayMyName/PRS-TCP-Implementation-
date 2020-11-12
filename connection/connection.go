package connection

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

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

	fmt.Printf(string(runes[9:12]))
	return conn, 7, nil
}

/** waits for a connection and sends the control port number*/
func acceptConnection(conn *net.UDPConn, controlPort int) (err error) {
	buffer := make([]byte, 100)

	_, err = conn.Read(buffer)
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

	fmt.Printf(str + "\n")
	_, err = fmt.Fprintf(conn, str)
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

	if string(buffer) != "ACK" {
		return errors.New("Couldn't receive ACK")
	}
	return
}
