package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type doubleChannel struct {
	ackChannel    chan int
	windowChannel chan bool
}

type packet struct {
	content  []byte
	lastTime time.Time
}

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
	CWND := 4
	numberOfACKInWindow := 0
	firstRTT = 20000

	//toutes les channels de communication
	channelWindowGlobal := make(chan bool, 100)
	channelWindowNewPackets := make(chan bool, 100)
	channelLoss := make(chan bool, 100)
	allACKChannel := make(chan int, 1000)
	doubleChannels := &map[int]doubleChannel{}

	packets := &map[int]packet{}

	channelPacketsAvailable := make(chan bool, 100)
	packetsToBeSent := []int{}
	channelEndOfFile := make(chan bool)

	//mutex de protection de la map ackChannels
	var mutexChannels = &sync.Mutex{}
	var mutexPackets = &sync.Mutex{}

	//variables de lecture du fichier
	bufferSize := 1494
	r := bufio.NewReader(f)
	readingBuffer := make([]byte, bufferSize)
	endOfFile := false
	lastSeqNum := -1

	// go routines d'écoute et de traitement d'ack/pertes
	go listenACK(connected, dataConn, allACKChannel)
	go handleACK(connected, mutexChannels, allACKChannel, doubleChannels, channelWindowGlobal, &ssthresh, &CWND, &numberOfACKInWindow, &lastSeqNum, channelEndOfFile)
	go handleLostPackets(connected, channelLoss, &packetsToBeSent, &ssthresh, &CWND, &numberOfACKInWindow)
	go handleWindowPriority(connected, mutexChannels, mutexPackets, doubleChannels, channelWindowGlobal, channelWindowNewPackets, channelPacketsAvailable, &packetsToBeSent)

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

		//on attend que la window permette d'envoyer un msg
		_ = <-channelWindowNewPackets

		mutexPackets.Lock()
		packetsToBeSent = append(packetsToBeSent, seqNum)
		sort.Ints(packetsToBeSent)

		(*packets)[seqNum] = packet{content: createPacket(seqNum, readingBuffer[:n])}
		mutexPackets.Unlock()
	}

	lastSeqNum = seqNum
	fmt.Printf("LAST SEQNUM : %d\n", lastSeqNum)
	_ = <-channelEndOfFile

	_, err = dataConn.WriteTo([]byte("FIN"), dataAddr)
	if err != nil {
		fmt.Printf("Error sending FIN")
		return
	}

	fmt.Printf("SENT %s\n", path)
	return
}

func sendNextPacket(transmitting *bool, packetsToBeSent *[]int, packets *map[int]packet, channelWindowGlobal chan bool) {
	for *transmitting {
		_ = <-channelWindowGlobal

		packet := (*packets)[(*packetsToBeSent)[0]]
		packet.lastTime = time.Now()
		sendPacket(packet)
	}
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

		//si le message est un ACK, on l'envoie se faire traiter
		if string(transmissionBuffer[0:3]) == "ACK" {
			packetNum, _ := strconv.Atoi(string(transmissionBuffer[3:9]))
			allACKChannel <- packetNum
		}
	}
}

/** change les variables de fonctionnement en cas de perte de paquets*/
func handleLostPackets(transmitting *bool, channelLoss chan bool, packetsToBeSent *[]int, ssthresh *int, CWND *int, numberOfACKInWindow *int) {
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
func handleACK(transmitting *bool, mutex *sync.Mutex, allACKChannel chan int, packetsToBeSent *[]int, packets *map[int]packet, channelWindowGlobal chan bool, ssthresh *int, CWND *int, numberOfACKInWindow *int, endOfFile *int, channelEndOfFile chan bool) (err error) {
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
			//slow start
			if *CWND < *ssthresh {
				mutex.Lock()

				//on acquitte tous packets avec un numéro de séquence inférieur
				for i := 0; i < len(*packetsToBeSent); i++ {
					packet := (*packetsToBeSent)[i]
					if packet <= highestReceivedSeqNum {
						*packetsToBeSent = (*packetsToBeSent)[1:len(*packetsToBeSent)]
						delete((*packets), packet)

						for j := 0; j < 2; j++ {
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
				for i := 0; i < len(*packetsToBeSent); i++ {
					packet := (*packetsToBeSent)[i]
					if packet <= highestReceivedSeqNum {
						*packetsToBeSent = (*packetsToBeSent)[1:len(*packetsToBeSent)]
						delete((*packets), packet)

						channelWindowGlobal <- false

						*numberOfACKInWindow++
					}
				}

				mutex.Unlock()

				if *numberOfACKInWindow >= *CWND {
					go func() {
						fmt.Printf("UPDATING WINDOW SIZE\n")
						*CWND++
						channelWindowGlobal <- false
						*numberOfACKInWindow = 0
						fmt.Printf("WINDOW SIZE : %d\n", *CWND)
					}()
				}
			}
			// si on recoit un ACK 3x, c'est que packet suivant celui acquitté est perdu
		} else if timesReceived == 3 {
			mutex.Lock()
			(*doubleChannels)[highestReceivedSeqNum+1].ackChannel <- -1
			mutex.Unlock()

			timesReceived = 0
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
func packetHandling(mutex *sync.Mutex, channelLoss chan bool, channelSendRequests chan int, channelWindowGlobal chan bool, content []byte, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, srtt *int) {

	var lastTime time.Time
	ack := -1

	//Tant qu'on a pas reçu l'acquittement
	for ack != 0 {
		//waiting for autorisation to send
		fmt.Printf("SEQNUM %d REQUESTING BEING SENT\n", seqNum)
		channelSendRequests <- seqNum
		fmt.Printf("SEQNUM %d WAITING FOR AUTHORISATION\n", seqNum)
		canSend := <-dB.windowChannel
		if !canSend {
			fmt.Printf("SEQNUM %d UNAUTHORISED\n", seqNum)
			break
		}
		fmt.Printf("SEQNUM %d GRANTED AUTHORISATION\n", seqNum)

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
		}(dB.ackChannel, srtt, lastTimeInt)

		//on ne veut pas traiter une demande de retransmission faite par une go routine lancée avant de recevoir une demande de fast retransmit
		for {
			ack = <-dB.ackChannel

			if ack == 0 {
				channelWindowGlobal <- false

				// fmt.Printf("%d RECEIVED ACK\n", seqNum)
				break
			} else if ack == lastTimeInt {
				channelLoss <- true
				channelWindowGlobal <- false

				fmt.Printf("SEQNUM " + strconv.Itoa(seqNum) + " TIMED OUT\n")
				fmt.Printf("RESENDING : " + strconv.Itoa(seqNum) + "\n")
				break
			} else if ack == -1 {
				channelLoss <- true
				channelWindowGlobal <- false

				fmt.Printf("SEQNUM " + strconv.Itoa(seqNum) + " FAST RETRANSMIT\n")
				fmt.Printf("RESENDING : " + strconv.Itoa(seqNum) + "\n")
				break
			}
		}
	}

	timeDiff := int(time.Now().Sub(lastTime) / time.Microsecond)
	*srtt = int(0.9*float32(*srtt) + 0.1*float32(timeDiff))

	fmt.Printf("SRTT : %d\n", *srtt)
	// fmt.Printf("ENDING SEQNUM %d ROUTINE\n", seqNum)
}
