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

		firstRTT, err := acceptConnection(publicConn, ipAddress, dataPort)
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
	executionTimeStart := time.Now()
	seqNum := 1

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

	info, err := os.Stat(finalPath)
	if err != nil {
		return err
	}
	fileSize := float32(info.Size()) / float32(1000000)

	f, err := os.Open(finalPath)
	if err != nil {
		fmt.Printf("Error opening file %v\n", err)
		return err
	}
	defer f.Close()

	// variables de fonctionnement de transmission
	ssthresh := 256
	CWND := 4
	numberOfACKInWindow := 0
	firstRTT = 20000

	//toutes les channels de communication
	channelWindowGlobal := make(chan bool, 100)
	channelLoss := make(chan int, 100)
	allACKChannel := make(chan int, 1000)
	ackChannels := &map[int](chan int){}

	//mutex de protection de la map ackChannels
	var mutex = &sync.Mutex{}

	//variables de lecture du fichier
	bufferSize := 1494
	r := bufio.NewReader(f)
	readingBuffer := make([]byte, bufferSize)
	endOfFile := false

	// go routines d'écoute et de traitement d'ack/pertes
	go listenACK(connected, dataConn, allACKChannel)
	go handleACK(connected, mutex, allACKChannel, ackChannels, channelWindowGlobal, &ssthresh, &CWND, &numberOfACKInWindow, &endOfFile)
	go handleLostPackets(connected, channelLoss, &ssthresh, &CWND, &numberOfACKInWindow)

	//Reading the file
	for !endOfFile {
		n, err := io.ReadFull(r, readingBuffer)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Printf("REACHED EOF\n")
			endOfFile = true
		} else if err != nil {
			fmt.Println("Error reading file:", err)
			return err
		}

		//on attend que la window permette d'envoyer un msg
		_ = <-channelWindowGlobal
		go packetHandling(mutex, ackChannels, channelLoss, append([]byte(nil), readingBuffer[:n]...), seqNum, dataConn, dataAddr, &firstRTT)

		seqNum++
		if seqNum == 1000000 {
			seqNum = 1
		}
		// time.Sleep(time.Duration(500) * time.Millisecond)
	}

	//on attend que tous les paquets sont bien reçu (acquittés) avant d'envoyer la fin de fichier
	finished := false
	for !finished {
		finished = <-channelWindowGlobal
	}

	_, err = dataConn.WriteTo([]byte("FIN"), dataAddr)
	if err != nil {
		fmt.Printf("Error sending FIN")
		return
	}

	executionTime := float32(time.Now().Sub(executionTimeStart)) / float32(time.Second)
	fmt.Printf("SENT %s\n", path)
	fmt.Printf(">>>TIME :  %v s\n ", executionTime)
	fmt.Printf(">>>FILE SIZE :  %v Mo\n ", fileSize)
	fmt.Printf(">>>FLOW RATE :  %v Mo/s\n ", fileSize/executionTime)
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

		fmt.Printf("RECEIVED : " + string(transmissionBuffer) + "\n")

		//si le message est un ACK, on l'envoie se faire traiter
		if string(transmissionBuffer[0:3]) == "ACK" {
			packetNum, _ := strconv.Atoi(string(transmissionBuffer[3:9]))
			allACKChannel <- packetNum
		}
	}
}

/** change les variables de fonctionnement en cas de perte de paquets*/
func handleLostPackets(transmitting *bool, channelLoss chan int, ssthresh *int, CWND *int, numberOfACKInWindow *int) {
	for *transmitting {
		_ = <-channelLoss

		// fast recovery
		*CWND /= 2
		*ssthresh = *CWND
		*numberOfACKInWindow = 0
	}
}

/** traite tout ack reçu */
func handleACK(transmitting *bool, mutex *sync.Mutex, allACKChannel chan int, ackChannels *map[int](chan int), channelWindowGlobal chan bool, ssthresh *int, CWND *int, numberOfACKInWindow *int, endOfFile *bool) (err error) {
	//fast retransmit variables
	highestReceivedSeqNum := 0
	timesReceived := 0

	//permet de lancer la fenêtre de départ
	for i := 0; i < *CWND; i++ {
		channelWindowGlobal <- true
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

		//check si l'acquittement n'a pas déjà été reçu
		if timesReceived == 1 {
			//slow start
			if *CWND < *ssthresh {
				mutex.Lock()

				//on acquitte tous packets avec un numéro de séquence inférieur
				for key, ackChannel := range *ackChannels {
					if key <= highestReceivedSeqNum {
						ackChannel <- 0
						fmt.Printf("YOU RECEIVED ACK, SEQNUM %d\n", key)
						delete((*ackChannels), key)

						for i := 0; i < 2; i++ {
							channelWindowGlobal <- false
						}

						*CWND++
						*numberOfACKInWindow++
						fmt.Printf("WINDOW SIZE : %d\n", *CWND)
					}
				}

				mutex.Unlock()
				//congestion avoidance
			} else {
				mutex.Lock()

				//on acquitte tous packets avec un numéro de séquence inférieur
				for key, ackChannel := range *ackChannels {
					if key <= highestReceivedSeqNum {
						ackChannel <- 0
						fmt.Printf("YOU RECEIVED ACK, SEQNUM %d\n", key)
						delete((*ackChannels), key)

						*numberOfACKInWindow++
						channelWindowGlobal <- false
					}
				}

				mutex.Unlock()

				if *numberOfACKInWindow >= *CWND {
					*CWND++
					channelWindowGlobal <- false
					*numberOfACKInWindow = 0
					fmt.Printf("WINDOW SIZE : %d\n", *CWND)
				}
			}

			//pas bon ça, source de paquets supplémentaires parfois
			//s'il ne reste plus à acquitter c'est que tous le fichier est envoyé
			if *endOfFile && len((*ackChannels)) == 0 {
				channelWindowGlobal <- true
			}
			// si on recoit un ACK 3x, c'est que packet suivant celui acquitté est perdu
		} //else if timesReceived == 3 {
		// 	fmt.Printf("PACKET : %d DROPPED\n", highestReceivedSeqNum+1)

		// 	mutex.Lock()
		// 	(*ackChannels)[highestReceivedSeqNum+1] <- -1
		// 	mutex.Unlock()
		// }
	}
	return
}

/** s'occupe de créer le packet et de l'envoyer/renvoyer*/
func packetHandling(mutex *sync.Mutex, ackChannels *map[int](chan int), channelLoss chan int, content []byte, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, srtt *int) {
	ackChannel := make(chan int, 100)

	//création de la channel de communication
	mutex.Lock()
	(*ackChannels)[seqNum] = ackChannel
	mutex.Unlock()

	//concaténation du numéro de séquence et du msg
	seq := strconv.Itoa(seqNum)
	zeros := 6 - len(seq)
	for i := 0; i < zeros; i++ {
		seq = "0" + seq
	}
	msg := append([]byte(seq), content...)

	fmt.Printf("SENDING : " + strconv.Itoa(seqNum) + "\n")

	var lastTime time.Time
	ack := -1

	//Tant qu'on a pas reçu l'acquittement
	for ack != 0 {
		lastTime = time.Now()
		lastTimeInt := lastTime.Nanosecond()

		_, err := dataConn.WriteTo(msg, dataAddr)
		if err != nil {
			fmt.Printf("Error sending packet %v\n", err)
			return
		}

		// envoie une demande de retransmission dans le futur, celle ci ne sera pas traitée si on recoit un 0 (ACK) ou un (-1)fast retransmit d'abord
		go func(ackChannel chan int, srtt *int, lastTimeInt int) {
			time.Sleep(time.Duration(int(float32(*srtt)*3)) * time.Microsecond)
			ackChannel <- lastTimeInt
		}(ackChannel, srtt, lastTimeInt)

		//on ne veut pas traiter une demande de retransmission faite par une go routine lancée avant de recevoir une demande de fast retransmit
		for ack != 0 && ack != lastTimeInt && ack != -1 {
			ack = <-ackChannel

			if ack == lastTimeInt || ack == -1 {
				channelLoss <- seqNum
				fmt.Printf("RESENDING : " + strconv.Itoa(seqNum) + "\n")
			}
		}
	}

	timeDiff := int(time.Now().Sub(lastTime) / time.Microsecond)
	*srtt = int(0.9*float32(*srtt) + 0.1*float32(timeDiff))

	fmt.Printf("SRTT : %d\n", *srtt)
	fmt.Printf("ENDING SEQNUM %d ROUTINE\n", seqNum)
}
