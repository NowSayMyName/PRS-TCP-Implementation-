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

type packet struct {
	content  []byte
	timeSent time.Time
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

	_, err = publicConn.WriteTo([]byte(str), remoteAddr)
	startTime := time.Now()
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

	f, err := os.Open(finalPath)
	if err != nil {
		fmt.Printf("Error opening file %v\n", err)
		return err
	}
	defer f.Close()

	channelWindow := make(chan bool)

	packets := map[int]*packet{}

	go listenACKGlobal(&packets, dataConn, dataAddr, connected, channelWindow, &firstRTT)

	bufferSize := 1400
	r := bufio.NewReader(f)

	readingBuffer := make([]byte, bufferSize)

	endOfFile := false
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

		_ = <-channelWindow

		// fmt.Printf(string(readingBuffer[:n]))
		go packetHandling(&packets, &packet{content: readingBuffer[:n]}, seqNum, dataConn, dataAddr, &firstRTT)
		//append([]byte(nil), readingBuffer[:n]...)

		seqNum++
		if seqNum == 1000000 {
			seqNum = 1
		}
	}

	//ici il faudrait attendre que TOUS les acquittements soient bien arrivés
	finished := false
	for !finished {
		finished = <-channelWindow
	}

	_, err = dataConn.WriteTo([]byte("FIN"), dataAddr)
	if err != nil {
		fmt.Printf("Error sending FIN")
		return
	}

	fmt.Printf("SENT %s\n", path)
	return
}

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

func listenACKGlobal(packets *map[int]*packet, dataConn *net.UDPConn, dataAddr net.Addr, transmitting *bool, channelWindow chan bool, srtt *int) (err error) {
	transmissionBuffer := make([]byte, 9)

	//fast retransmit variables
	lastReceivedSeqNum := 0
	timesReceived := 1

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

			//test for fast retransmit
			if lastReceivedSeqNum == packetNum {
				timesReceived++
			} else {
				lastReceivedSeqNum = packetNum
				timesReceived = 0
			}

			//check si l'acquittement n'a pas déjà été reçu
			if timesReceived == 1 {
				for key := range *packets {
					if key <= packetNum {
						timeDiff := int(time.Now().Sub((*packets)[key].timeSent) / time.Microsecond)
						if timeDiff > 10000000 {
							timeDiff = 10000000
						}

						// fmt.Printf("TIME DIFF : " + strconv.Itoa(timeDiff) + "\n")

						*srtt = int(0.9*float32(*srtt) + 0.1*float32(timeDiff))
						fmt.Printf("SRTT : " + strconv.Itoa(*srtt) + "\n")

						delete(*packets, key)
						if len(*packets) == 0 {
							channelWindow <- true
						} else {
							for i := 0; i < 2; i++ {
								channelWindow <- false
							}
						}
					} else {
						break
					}
				}
				// si on recoit un ACK 3x, c'est que packet suivant celui acquitté est perdu
			} else if timesReceived == 3 {
				if lostPacket, ok := (*packets)[lastReceivedSeqNum+1]; ok {
					go packetHandling(packets, lostPacket, lastReceivedSeqNum+1, dataConn, dataAddr, srtt)
				}
			}
		}
	}
	return
}

func packetHandling(packets *map[int]*packet, buffer *packet, seqNum int, dataConn *net.UDPConn, dataAddr net.Addr, srtt *int) {
	fmt.Printf("SENDING : " + strconv.Itoa(seqNum) + ":\n")
	// fmt.Printf(string(buffer))

	for {
		lastTime := time.Now()
		buffer.timeSent = lastTime
		(*packets)[seqNum] = buffer
		go sendPacket(buffer.content, seqNum, dataConn, dataAddr)
		time.Sleep(time.Duration(int(float32(*srtt)*3)) * time.Microsecond)
		//si le paquet a déjà été acquitté (n'est plus dans le tableau) ou qu'une autre go routine le retransmet déjà (fast retransmit)
		if _, ok := (*packets)[seqNum]; !ok || (*packets)[seqNum].timeSent != lastTime {
			break
		}
		fmt.Printf("RESENDING : " + strconv.Itoa(seqNum) + "\n")
	}
}
