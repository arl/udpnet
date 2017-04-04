package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/aurelien-rainone/udpnet"
)

const (
	serverPort  = 30000
	protocolID  = 0x99887766
	deltaTime   = time.Duration(250) * time.Millisecond
	sendRate    = time.Duration(250) * time.Millisecond
	timeout     = time.Duration(10) * time.Second
	MaxSequence = 0xFFFFFFFF
)

func main() {

	var clientPort = 30001
	if len(os.Args) > 1 {
		if val, err := strconv.Atoi(os.Args[1]); err != nil {
			fmt.Println("invalid port number", os.Args[1], ", ", err)
			os.Exit(1)
		} else {
			clientPort = val
		}
	}
	fmt.Println("client port is", clientPort)
	connection := udpnet.NewReliableConn(protocolID, timeout, MaxSequence)

	if !connection.Start(clientPort) {
		fmt.Printf("could not start connection on port %d\n", clientPort)
		return
	}
	defer connection.Stop()

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	if err != nil {
		fmt.Printf("could not resolve server address, %v\n", err)
		return
	}
	connection.Connect(addr)
	var connected bool

	for {
		if !connected && connection.IsConnected() {
			fmt.Printf("client connected to server\n")
			connected = true
		}

		if !connected && connection.ConnectFailed() {
			fmt.Printf("connection failed\n")
			break
		}

		packet := []byte("client to server")
		connection.SendPacket(packet)
		for {
			var packet [256]byte

			bytesRead := connection.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
			fmt.Printf("received packet from server\n")
		}

		connection.Update(deltaTime)
		time.Sleep(deltaTime)
	}
}
