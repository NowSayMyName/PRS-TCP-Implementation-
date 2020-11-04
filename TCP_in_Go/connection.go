package TCP_in_Go

import (
	"bufio"
	"fmt"
	"net"
)

/** renvoie le port utilisé par le serveur pour les messages de controles, sinon des valeurs <0*/
func int connectionToServer(conn *net.UDPConn) {
	buffer := make([]byte, *bufSize)

	_, err = fmt.Fprintf(conn, "SYN")
	if err != nil {
		fmt.Printf("Could not send SYN %v", err)
		return
	}
  
	_, err = conn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN-ACK %v", err)
		return
	}

	printf("%s\n", buffer);
	runes := []rune(buffer)
	
	if (string(runes[0:7]) != "SYN-ACK ") {
		fmt.Printf(string(runes[0:7]) + " %v", err)
	 	return
	}
	
	_, err = fmt.Fprintf(conn, "ACK")
	if err != nil {
		fmt.Printf("Could not send ACK %v", err)
		return
	}
  
	fmt.Printf(string(runes[9:12]))
	return strconv.Atoi(string(runes[9:12]))
  }