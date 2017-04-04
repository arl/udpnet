package main

import (
	"fmt"
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
	connection := udpnet.NewReliableConn(protocolID, timeout, MaxSequence)

	if !connection.Start(serverPort) {
		fmt.Printf("could not start connection on port %d\n", serverPort)
		return
	}
	defer connection.Stop()

	connection.Listen()

	for {
		if connection.IsConnected() {
			packet := []byte("server to client")
			connection.SendPacket(packet)
		}

		for {
			var packet [256]byte

			bytesRead := connection.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
			fmt.Printf("received packet from client\n")
		}

		connection.Update(deltaTime)
		time.Sleep(deltaTime)
	}
}
