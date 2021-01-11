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
	"sync"
	"time"
)

type packet struct {
	content []byte
	time    time.Time
}

func getArgs() (portNumber int) {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: go run serverClean.go  <port_number>\n")
		os.Exit(1)
	} else {
		portNumber, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Printf("Usage: go run serverClean.go <port_number>\n")
			os.Exit(1)
		} else {
			return portNumber
		}

	}
	return -1
}

func main() {
	portNumber := getArgs()

	publicAddr := net.UDPAddr{
		Port: portNumber,
		IP:   net.ParseIP("0.0.0.0"),
	}

	dataPort := portNumber
	publicConn, err := net.ListenUDP("udp", &publicAddr)
	fmt.Printf("Starting server on address: %s:%d\n\n", "0.0.0.0", portNumber)
	if err != nil {
		fmt.Printf("Couldn't listen %v\n", err)
		return
	}

	for {
		dataPort++

		dataAddr := net.UDPAddr{
			Port: dataPort,
			IP:   net.ParseIP("0.0.0.0"),
		}

		dataConn, err := net.ListenUDP("udp", &dataAddr)
		if err != nil {
			fmt.Printf("Couldn't listen \n%v", err)
			return
		}

		firstRTT, err := acceptConnection(publicConn, "0.0.0.0", dataPort)
		if err != nil {
			fmt.Printf("Couldn't accept connection \n%v\n", err)
			return
		}
		go handleConnection(dataConn, firstRTT)
	}
}

func handleConnection(dataConn *net.UDPConn, firstRTT int) (err error) {
	transmitting := true
	buffer := make([]byte, 100)

	_, remoteAddr, err := dataConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive path \n%v", err)
		return err
	}

	fmt.Printf("SEND FILE : %s\n", buffer)
	go sendFile(&transmitting, string(buffer), dataConn, remoteAddr, firstRTT)
	// go listenOnDataPort(&transmitting, dataConn, remoteAddr, &windowSize)

	return
}

/** waits for a connection and sends the public port number*/
func acceptConnection(publicConn *net.UDPConn, ipAddress string, dataPort int) (firstRTT int, err error) {
	buffer := make([]byte, 100)

	_, remoteAddr, err := publicConn.ReadFrom(buffer)
	if err != nil {
		fmt.Printf("Could not receive SYN \n%v", err)
		return -1, err
	}
	fmt.Printf("%s\n", buffer)

	if string(buffer[0:3]) != "SYN" {
		fmt.Printf(string(buffer[0:3])+" %v", err)
		return -1, errors.New("Could not receive SYN")
	}

	str := "SYN-ACK" + strconv.Itoa(dataPort)
	fmt.Println(str)

	startTime := time.Now()
	_, err = publicConn.WriteTo([]byte(str), remoteAddr)
	if err != nil {
		fmt.Printf("Could not send SYN-ACK \n%v", err)
		return -1, err
	}

	_, err = publicConn.Read(buffer)
	if err != nil {
		fmt.Printf("Could not receive ACK \n%v", err)
		return -1, err
	}
	fmt.Printf("%s\n\n", buffer)

	if string(buffer[0:3]) != "ACK" {
		return -1, errors.New("Couldn't receive ACK")
	}

	fmt.Printf("Connection started on port %d\n", dataPort)
	return int(time.Now().Sub(startTime) / time.Microsecond), err
}

/** takes a path to a file and sends it to the given address*/
func sendFile(connected *bool, path string, dataConn *net.UDPConn, dataAddr net.Addr, firstRTT int) (err error) {
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
	if err != nil {
		fmt.Printf("Error opening file %v\n", err)
		return err
	}
	defer f.Close()

	// variables de fonctionnement de transmission
	ssthresh := 256
	CWND := 50
	numberOfACKInWindow := 0
	firstRTT = 20000

	//toutes les channels de communication
	channelWindowGlobal := make(chan bool, 100)
	channelLoss := make(chan bool, 100)
	allACKChannel := make(chan int, 1000)

	packets := &map[int]packet{}
	channelEndOfFile := make(chan bool)

	var mutex = &sync.Mutex{}

	//variables de lecture du fichier
	bufferSize := 1494
	r := bufio.NewReader(f)
	readingBuffer := make([]byte, bufferSize)
	endOfFile := false
	lastSeqNum := -1

	// go routines d'écoute et de traitement d'ack/pertes
	go listenACK(connected, dataConn, allACKChannel)
	go handleACK(connected, mutex, allACKChannel, packets, channelWindowGlobal, &ssthresh, &CWND, &numberOfACKInWindow, &lastSeqNum, channelEndOfFile, &firstRTT)
	go handleLostPackets(connected, channelLoss, &ssthresh, &CWND, &numberOfACKInWindow)

	//Reading the file
	for !endOfFile {
		seqNum++

		n, err := io.ReadFull(r, readingBuffer)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Printf("REACHED EOF\n")
			endOfFile = true
		} else if err != nil {
			fmt.Println("Error reading file:", err)
			return err
		}

		content := createPacket(seqNum, append([]byte(nil), readingBuffer[:n]...))
		//on attend que la window permette d'envoyer un msg
		_ = <-channelWindowGlobal

		mutex.Lock()
		(*packets)[seqNum] = packet{content, time.Now()}
		mutex.Unlock()

		go packetHandling(mutex, packets, channelLoss, content, seqNum, dataConn, dataAddr, firstRTT)
	}

	lastSeqNum = seqNum
	_ = <-channelEndOfFile

	_, err = dataConn.WriteTo([]byte("FIN"), dataAddr)
	if err != nil {
		fmt.Printf("Error sending FIN")
		return
	}

	fmt.Printf("SENT %s\n", path)
	return
}

/** fonction d'écoute sur le port de communication, transmet tout ack reçu à la fonction de traitement via une channel */
func listenACK(transmitting *bool, dataConn *net.UDPConn, allACKChannel chan int) {
	transmissionBuffer := make([]byte, 9)

	for *transmitting {
		_, err := dataConn.Read(transmissionBuffer)
		if err != nil {
			fmt.Printf("Error reading packets %v\n", err)
			return
		}

		fmt.Printf("RECEIVED " + string(transmissionBuffer[0:9]) + "\n")

		//si le message est un ACK, on l'envoie se faire traiter
		if string(transmissionBuffer[0:3]) == "ACK" {
			packetNum, _ := strconv.Atoi(string(transmissionBuffer[3:9]))
			allACKChannel <- packetNum
		}
	}
}

/** change les variables de fonctionnement en cas de perte de paquets*/
func handleLostPackets(transmitting *bool, channelLoss chan bool, ssthresh *int, CWND *int, numberOfACKInWindow *int) {
	for *transmitting {
		_ = <-channelLoss
		fmt.Printf("LOSS\n")

		// fast recovery
		*CWND /= 2
		*ssthresh = *CWND
		*numberOfACKInWindow = 0
	}
}

/** traite tout ack reçu */
func handleACK(transmitting *bool, mutex *sync.Mutex, allACKChannel chan int, packets *map[int]packet, channelWindowGlobal chan bool, ssthresh *int, CWND *int, numberOfACKInWindow *int, endOfFile *int, channelEndOfFile chan bool, SRTT *int) (err error) {
	//fast retransmit variables
	highestReceivedSeqNum := 0
	timesReceived := 0

	//permet de lancer la fenêtre de départ
	for i := 0; i < *CWND; i++ {
		channelWindowGlobal <- false
	}

	for *transmitting {
		packetNum := <-allACKChannel

		//test for fast retransmit
		if highestReceivedSeqNum == packetNum {
			timesReceived++
		} else if highestReceivedSeqNum < packetNum {
			highestReceivedSeqNum = packetNum
			timesReceived = 1
		}

		fmt.Printf("PROCESSING SEQNUM : %d\n", highestReceivedSeqNum)

		//check si l'acquittement n'a pas déjà été reçu
		if timesReceived == 1 {
			// //slow start
			// if *CWND < *ssthresh {
			// 	fmt.Printf("LOCKING\n")
			// 	mutex.Lock()
			// 	fmt.Printf("LOCK ACQUIRED\n")

			// 	//on acquitte tous packets avec un numéro de séquence inférieur
			// 	for seqNum, packet := range *packets {
			// 		if seqNum <= highestReceivedSeqNum {
			// 			timeDiff := int(time.Now().Sub(packet.time) / time.Microsecond)
			// 			*SRTT = int(0.9*float32(*SRTT) + 0.1*float32(timeDiff))

			// 			fmt.Printf("SRTT : %d\n", *SRTT)
			// 			delete((*packets), seqNum)

			// 			fmt.Printf("DONE DELETING\n")
			// 			for j := 0; j < 2; j++ {
			// 				go func() { channelWindowGlobal <- false }()
			// 			}
			// 			fmt.Printf("DONE UPDATING WINDOW\n")

			// 			*CWND++
			// 			*numberOfACKInWindow++
			// 			fmt.Printf("WINDOW SIZE : %d\n", *CWND)
			// 		}
			// 	}

			// 	mutex.Unlock()
			// 	fmt.Printf("UNLOCKING\n")

			// 	//congestion avoidance
			// } else {
			// 	fmt.Printf("LOCKING\n")

			// 	mutex.Lock()
			// 	fmt.Printf("LOCK ACQUIRED\n")

			// 	//on acquitte tous packets avec un numéro de séquence inférieur
			// 	for seqNum, packet := range *packets {
			// 		if seqNum <= highestReceivedSeqNum {
			// 			timeDiff := int(time.Now().Sub(packet.time) / time.Microsecond)
			// 			*SRTT = int(0.9*float32(*SRTT) + 0.1*float32(timeDiff))

			// 			fmt.Printf("SRTT : %d\n", *SRTT)
			// 			delete((*packets), seqNum)

			// 			fmt.Printf("DONE DELETING\n")
			// 			go func() { channelWindowGlobal <- false }()
			// 			fmt.Printf("DONE UPDATING WINDOW\n")

			// 			*numberOfACKInWindow++
			// 		}
			// 	}

			// 	mutex.Unlock()
			// 	fmt.Printf("UNLOCKING\n")

			// 	if *numberOfACKInWindow >= *CWND {
			// 		go func() {
			// 			fmt.Printf("UPDATING WINDOW SIZE\n")
			// 			*CWND++
			// 			channelWindowGlobal <- false
			// 			*numberOfACKInWindow = 0
			// 			fmt.Printf("WINDOW SIZE : %d\n", *CWND)
			// 		}()
			// 	}
			// }
			fmt.Printf("LOCKING\n")

			mutex.Lock()
			fmt.Printf("LOCK ACQUIRED\n")

			//on acquitte tous packets avec un numéro de séquence inférieur
			for seqNum, packet := range *packets {
				if seqNum <= highestReceivedSeqNum {
					timeDiff := int(time.Now().Sub(packet.time) / time.Microsecond)
					*SRTT = int(0.9*float32(*SRTT) + 0.1*float32(timeDiff))

					fmt.Printf("SRTT : %d\n", *SRTT)
					delete((*packets), seqNum)

					fmt.Printf("DONE DELETING\n")
					go func() { channelWindowGlobal <- false }()
					fmt.Printf("DONE UPDATING WINDOW\n")

					*numberOfACKInWindow++
				}
			}

			mutex.Unlock()
			fmt.Printf("UNLOCKING\n")

			// si on recoit un ACK 3x, c'est que packet suivant celui acquitté est perdu
		} else if timesReceived == 3 {
		}

		if *endOfFile == highestReceivedSeqNum {
			fmt.Printf("ALL PACKETS HAVE BEEN RECEIVED\n")
			channelEndOfFile <- true
		}

		fmt.Printf("DONE PROCESSING SEQNUM : %d\n", highestReceivedSeqNum)
	}
	//s'il ne reste plus à acquitter c'est que tous le fichier est envoyé

	return
}

func createPacket(seqNum int, content []byte) []byte {
	seq := strconv.Itoa(seqNum)
	zeros := 6 - len(seq)
	for i := 0; i < zeros; i++ {
		seq = "0" + seq
	}
	return append([]byte(seq), content...)
}

/** s'occupe de créer le packet et de l'envoyer/renvoyer*/
func packetHandling(mutex *sync.Mutex, packets *map[int]packet, channelLoss chan bool, msg []byte, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, rtt int) {
	//Tant qu'on a pas reçu l'acquittement

	for {
		fmt.Printf("SENDING SEQNUM : %d\n", seqNum)

		_, err := dataConn.WriteTo(msg, dataAddr)
		if err != nil {
			fmt.Printf("Error sending packet %v\n", err)
			return
		}

		time.Sleep(time.Duration(rtt*3) * time.Microsecond)

		mutex.Lock()
		if _, ok := (*packets)[seqNum]; !ok {
			fmt.Printf("ENDING ROUTINE FOR SEQNUM : %d\n", seqNum)
			mutex.Unlock()
			break
		}
		mutex.Unlock()

		channelLoss <- true
	}
}
