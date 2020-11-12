package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	f, err := os.Open("C:/Users/Melvil/Desktop/INSA/PRS/PRS_TCP_Implementation_/TCP_in_Go/test.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	r := bufio.NewReader(f)
	b := make([]byte, 100)
	for {
		fmt.Println("[   NEW PACKET   ]")
		n, err := r.Read(b)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading file:", err)
			break
		}
		fmt.Println(string(b[0:n]))
	}
}
