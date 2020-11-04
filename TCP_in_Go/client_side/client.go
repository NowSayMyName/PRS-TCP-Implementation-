package main

import (
	"fmt"
)

func main() {
	conn, port, err := connectionToServer("127.0.0.1:5000")
	if err != nil {
		fmt.Printf("Could not connect %v", err)
	}
	defer conn.Close()
	fmt.Printf("Could not connect %v", port)

}
