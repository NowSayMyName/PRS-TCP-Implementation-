package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
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

		dataAddr := net.UDPAddr{
			Port: dataPort,
			IP:   net.ParseIP(ipAddress),
		}

		dataConn, err := net.ListenUDP("udp", &dataAddr)
		if err != nil {
			fmt.Printf("Couldn't listen \n%v", err)
			return
		}

		err = acceptConnection(publicConn, ipAddress, dataPort)
		if err != nil {
			fmt.Printf("Couldn't accept connection \n%v\n", err)
			return
		}
		go handleConnection(dataConn)
	}
}

func handleConnection(dataConn *net.UDPConn) (err error) {
	transmitting := true
	buffer := make([]byte, 100)

	_, remoteAddr, err := dataConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive path \n%v", err)
		return err
	}

	fmt.Printf("SEND FILE : %s\n", buffer)
	go sendFile(&transmitting, string(buffer), dataConn, remoteAddr)
	// go listenOnDataPort(&transmitting, dataConn, remoteAddr, &windowSize)

	return
}

/** waits for a connection and sends the public port number*/
func acceptConnection(publicConn *net.UDPConn, ipAddress string, dataPort int) (err error) {
	buffer := make([]byte, 100)

	_, remoteAddr, err := publicConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN \n%v", err)
		return err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "SYN" {
		fmt.Printf(string(buffer[0:3])+" %v", err)
		return errors.New("Could not receive SYN")
	}

	str := "SYN-ACK" + strconv.Itoa(dataPort)
	fmt.Println(str)

	_, err = publicConn.WriteTo([]byte(str), remoteAddr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return err
	}

	_, err = publicConn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive ACK \n%v", err)
		return err
	}
	fmt.Printf("%s\n\n", buffer)

	if string(buffer[0:3]) != "ACK" {
		return errors.New("Couldn't receive ACK")
	}

	fmt.Printf("Connection started on port %d\n", dataPort)
	return nil
}

/** takes a path to a file and sends it to the given address*/
func sendFile(connected *bool, path string, dataConn *net.UDPConn, dataAddr net.Addr) (err error) {
	seqNum := 0

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error finding absolute path %v\n", err)
		return err
	}

	finalPath := pwd + "/" + path
	finalPath = strings.Replace(finalPath, "\n", "", -1)
	finalPath = strings.Replace(finalPath, "\r", "", -1)
	finalPath = strings.Replace(finalPath, "%", "", -1)
	finalPath = strings.Replace(finalPath, "\x00", "", -1)

	f, err := os.Open(finalPath)
	// f, err := os.Open("/Users/yoannrouxel-duval/go/src/github.com/NowSayMyName/PRS_TCP_Implementation/TCP_in_Go/server_side/newFile.mp3")
	if err != nil {
		fmt.Printf("Error opening file %v\n", err)
		return err
	}
	defer f.Close()

	channelWindow := make(chan bool)

	transmitting := true
	packets := map[int]time.Time{}

	firstRTT := 1000000
	go listenACKGlobal(&packets, dataConn, dataAddr, &transmitting, channelWindow, &firstRTT)

	bufferSize := 1400
	r := bufio.NewReader(f)

	readingBuffer := make([]byte, bufferSize)
	// var currentByte int64 = 0

	endOfFile := false
	for !endOfFile {
		// time.Sleep(1000 * time.Millisecond)
		//Reading the file
		// n, err := f.ReadAt(readingBuffer, currentByte)
		// currentByte += int64(n)
		// fmt.Printf("READ %d BYTES\n", currentByte)

		// n, err := io.ReadAtLeast(f, readingBuffer, bufferSize)

		n, err := io.ReadFull(r, readingBuffer)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Printf("REACHED EOF\n")
			endOfFile = true
		}
		if err != nil {
			fmt.Println("Error reading file:", err)
			return err
		}

		packets[seqNum] = time.Now()
		_ = <-channelWindow

		// go listenACK(n, readingBuffer, seqNum, dataConn, dataAddr, windowSize)
		go timeCheck2(&packets, readingBuffer[:n], seqNum, dataConn, dataAddr, &firstRTT)

		seqNum++
		if seqNum == 1000000 {
			seqNum = 0
		}
	}
	go timeCheck2(&packets, []byte("FIN"), seqNum, dataConn, dataAddr, &firstRTT)
	return
}

/*func listenACK(n int, buffer []byte, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, windowSize *int) (err error) {
	fmt.Printf("SENDING : " + strconv.Itoa(seqNum) + "\n")
	go sendPacket(n, buffer, seqNum, dataConn, dataAddr, windowSize, true)
	transmitionBuffer := make([]byte, 9)
	start := time.Now()

	acknowledged := false
	for !acknowledged {
		_, err := dataConn.Read(transmitionBuffer)
		if err != nil {
			fmt.Printf("Error reading packets %v\n", err)
			return err
		}
		fmt.Printf("RECEIVED : " + string(transmitionBuffer) + "\n")
		if string(transmitionBuffer[0:9]) == "ACK"+strconv.Itoa(seqNum) {
			acknowledged = true
			*windowSize++
			break
		}
		elapsed := time.Now().Sub(start)
		if elapsed > 1000 {
			fmt.Printf("RESENDING : " + strconv.Itoa(seqNum) + "\n")
			go sendPacket(n, buffer, seqNum, dataConn, dataAddr, windowSize, false)
			start = time.Now()
		}
	}
	return
}*/

func sendPacket(buffer []byte, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr) (err error) {
	//Sending fragment
	seq := strconv.Itoa(seqNum)
	// fmt.Printf("Sequence number: %d\n", seqNum)
	zeros := 6 - len(seq)
	for i := 0; i < zeros; i++ {
		seq = "0" + seq
	}
	// fmt.Println(string(buffer[0:n]))
	msg := append([]byte(seq), buffer...)

	_, err = dataConn.WriteTo(msg, dataAddr)
	if err != nil {
		fmt.Printf("Error sending packet %v\n", err)
		return err
	}
	return
}

/** retourne le nouveau RTT, avec beta = 1 - alpha (mais évite de répéter ce calcul) */
func getRTT(lastRTT int, measuredRTT int, alpha float32, beta float32) int {
	return int(alpha*float32(lastRTT) + beta*float32(measuredRTT))
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

/*
func listenACK2(seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, windowSize *int, acknowledged *bool) (err error) {
	transmitionBuffer := make([]byte, 9)
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
		}
	}
	return
}*/

/*
func remove(packets []int, value int) []int {
	for i := 0; i < len(packets); i++ {
		if packets[i] == value {
			// fmt.Printf("YES REMOVED " + strconv.Itoa(value) + "\n")
			return append(packets[:i], packets[i+1:]...)
		}
	}
	return packets
}

func contains(packets []int, value int) bool {
	for _, v := range packets {
		if v == value {
			// fmt.Printf("YES CONTAINS " + strconv.Itoa(value) + "\n")
			return true
		}
	}
	return false
}*/

func listenACKGlobal(packets *map[int]time.Time, dataConn *net.UDPConn, dataAddr net.Addr, transmitting *bool, channelWindow chan bool, srtt *int) (err error) {
	transmissionBuffer := make([]byte, 9)

	channelWindow <- true
	for *transmitting {
		_, err = dataConn.Read(transmissionBuffer)
		if err != nil {
			fmt.Printf("Error reading packets %v\n", err)
			return err
		}
		fmt.Printf("RECEIVED : " + string(transmissionBuffer) + "\n")
		if string(transmissionBuffer[0:3]) == "ACK" {
			packetNum, _ := strconv.Atoi(string(transmissionBuffer[3:9]))
			timeDiff := int(time.Now().Sub((*packets)[packetNum]) / time.Microsecond)
			if timeDiff > 10000000 {
				timeDiff = 10000000
			}

			// fmt.Printf("TIME DIFF : " + strconv.Itoa(timeDiff) + "\n")

			*srtt = getRTT(*srtt, timeDiff, 0.9, 0.1)
			fmt.Printf("SRTT : " + strconv.Itoa(*srtt) + "\n")

			delete(*packets, packetNum)
			for i := 0; i < 1; i++ {
				channelWindow <- true
			}
		}
	}
	return
}

func timeCheck2(packets *map[int]time.Time, buffer []byte, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, srtt *int) {
	fmt.Printf("SENDING : " + strconv.Itoa(seqNum) + "\n")
	for {
		go sendPacket(buffer, seqNum, dataConn, dataAddr)
		// time.Sleep(time.Duration(*srtt))
		time.Sleep(time.Duration(*srtt) * time.Microsecond)
		if _, ok := (*packets)[seqNum]; !ok {
			break
		}
		fmt.Printf("RESENDING : " + strconv.Itoa(seqNum) + "\n")
	}
}
