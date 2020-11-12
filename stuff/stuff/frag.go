package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	f, err := os.Open("C:/Users/Melvil/go/src/github.com/MelvilB/PRS/PRS_TCP_Implementation/stuff/stuff/test123.txt")
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
