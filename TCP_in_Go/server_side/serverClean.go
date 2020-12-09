package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

func getArgs() (ipaddress string, portNumber int) {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: go run serverClean.go <server_address> <port_number>\n")
		os.Exit(1)
	} else {
		portNumber, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Usage: go run serverClean.go <server_address> <port_number>\n")
			os.Exit(1)
		} else {
			return os.Args[1], portNumber
		}

	}
	return "", -1
}

func main() {
	ipAddress, portNumber := getArgs()

	//stopCh := make(chan struct{})

	publicAddr := net.UDPAddr{
		Port: portNumber,
		IP:   net.ParseIP(ipAddress),
	}

	dataPort := portNumber
	publicConn, err := net.ListenUDP("udp", &publicAddr)
	fmt.Printf("Starting server on address: %s:%d\n\n", ipAddress, portNumber)
	if err != nil {
		fmt.Printf("Couldn't listen %v\n", err)
		return
	}

	for {
		dataPort++
		dataConn, err := acceptConnection(publicConn, ipAddress, dataPort)
		if err != nil {
			fmt.Printf("Couldn't accept connection \n%v\n", err)
			return
		}
		go handleConnection(dataConn)
	}
}

func handleConnection(dataConn *net.UDPConn) (err error) {
	windowSize := 1
	transmitting := true
	buffer := make([]byte, 100)

	_, remoteAddr, err := dataConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive path \n%v", err)
		return err
	}

	fmt.Printf("SEND FILE : %s\n", buffer)
	go sendFile(&transmitting, string(buffer), dataConn, remoteAddr, &windowSize)
	// go listenOnDataPort(&transmitting, dataConn, remoteAddr, &windowSize)

	return
}

/** waits for a connection and sends the public port number*/
func acceptConnection(publicConn *net.UDPConn, ipAddress string, dataPort int) (dataConn *net.UDPConn, err error) {
	buffer := make([]byte, 100)

	dataAddr := net.UDPAddr{
		Port: dataPort,
		IP:   net.ParseIP(ipAddress),
	}

	_, remoteAddr, err := publicConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN \n%v", err)
		return nil, err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "SYN" {
		fmt.Printf(string(buffer[0:3])+" %v", err)
		return nil, errors.New("Could not receive SYN")
	}

	str := "SYN-ACK" + strconv.Itoa(dataPort)
	fmt.Println(str)

	_, err = publicConn.WriteTo([]byte(str), remoteAddr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return nil, err
	}

	_, err = publicConn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive ACK \n%v", err)
		return nil, err
	}
	fmt.Printf("%s\n\n", buffer)

	if string(buffer[0:3]) != "ACK" {
		return nil, errors.New("Couldn't receive ACK")
	}

	dataConn, err = net.ListenUDP("udp", &dataAddr)
	if err != nil {
		fmt.Printf("Couldn't listen \n%v", err)
		return nil, err
	}

	fmt.Printf("Connection started on port %d\n", dataPort)
	return dataConn, nil
}

/** takes a path to a file and sends it to the given address*/
func sendFile(connected *bool, path string, dataConn *net.UDPConn, dataAddr net.Addr, windowSize *int) (err error) {
	seqNum := 0

	// pwd, err := os.Getwd()
	// if err != nil {
	// 	fmt.Printf("Error finding absolute path %v\n", err)
	// 	return err
	// }

	// finalPath := pwd + "/" + path
	// finalPath = strings.Replace(finalPath, "\n", "", -1)
	// finalPath = strings.Replace(finalPath, "\r", "", -1)
	// finalPath = strings.Replace(finalPath, "%", "", -1)
	// finalPath = strings.Replace(finalPath, "\x00", "", -1)

	// fmt.Printf("%s\n", finalPath)

	// clean := strings.Map(func(r rune) rune {
	// 	if unicode.IsGraphic(r) {
	// 		return r
	// 	}
	// 	return -1
	// }, finalPath)

	// fmt.Printf("%q\n", clean)
	// fmt.Println(len(clean))

	// clean = strings.Map(func(r rune) rune {
	// 	if unicode.IsPrint(r) {
	// 		return r
	// 	}
	// 	return -1
	// }, finalPath)

	// fmt.Printf("%q\n", clean)
	// fmt.Println(len(clean))

	f, err := os.Open("/Users/yoannrouxel-duval/go/src/github.com/NowSayMyName/PRS_TCP_Implementation/TCP_in_Go/server_side/newFile.mp3")
	if err != nil {
		fmt.Printf("Error opening file %v\n", err)
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	readingBuffer := make([]byte, 100)
	endOfFile := false
	for !endOfFile {
		//Reading the file
		fmt.Println("[   NEW PACKET   ]")
		n, err := r.Read(readingBuffer)
		if err == io.EOF {
			endOfFile = true
		}
		if err != nil {
			fmt.Println("Error reading file:", err)
			return err
		}

		for *windowSize == 0 {
		}
		acknowledged := false

		go listenACK(n, seqNum, dataConn, dataAddr, windowSize, &acknowledged)
		go sendPacket(n, seqNum, dataConn, dataAddr, windowSize)

		seqNum++
		if seqNum == 1000000 {
			seqNum = 0
		}
	}
	_, err = dataConn.WriteTo([]byte("FIN"), dataAddr)
	if err != nil {
		fmt.Printf("Error sending FIN")
	}

	return
}

func listenACK(n int, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, windowSize *int, acknowledged *bool) (err error) {
	transmitionBuffer := make([]byte, 100)
	start := time.Now()
	for !*acknowledged {
		_, err := dataConn.Read(transmitionBuffer)
		if err != nil {
			fmt.Printf("Error reading packets %v\n", err)
			return err
		}
		fmt.Printf("RECEIVED : " + string(transmitionBuffer) + "\n")
		if string(transmitionBuffer[0:9]) == "ACK"+strconv.Itoa(seqNum) {
			*acknowledged = true
			*windowSize++
			break
		}
		elapsed := start.Sub(time.Now())
		if elapsed > 1 {
			go sendPacket(n, seqNum, dataConn, dataAddr, windowSize)
			start := time.Now()
		}
	}
	return
}

func sendPacket(n int, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, windowSize *int) (err error) {
	readingBuffer := make([]byte, 100)
	//Sending fragment
	seq := strconv.Itoa(seqNum)
	fmt.Printf("Sequence number: %d\n", seqNum)
	zeros := 6 - len(seq)
	for i := 0; i < zeros; i++ {
		seq = "0" + seq
	}
	byteSeq := []byte(seq)
	fmt.Println(string(readingBuffer[0:n]))
	msg := append(byteSeq, readingBuffer...)

	_, err = dataConn.WriteTo(msg, dataAddr)
	if err != nil {
		fmt.Printf("Error sending packet %v\n", err)
		return err
	}
	*windowSize--
	return
}

/*
func timeCheck(n int, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, windowSize *int) {
	start := time.Now()
	for {
		select {
		case <-stopCh:
			return
		default:
			elapsed := time.Since(start)
			elapsed = elapsed / 1000000 //to get time in milliseconds
			if elapsed > 500 {
				go sendPacket(n, seqNum, dataConn, dataAddr, windowSize)
				return
			}
		}
	}
}
*/
