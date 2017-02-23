package main

import (
	"fmt"
	"time"

	"github.com/aurelien-rainone/udpnet"
)

type dummyCallback struct{}

func (dc dummyCallback) OnStart()      { fmt.Println("start") }
func (dc dummyCallback) OnStop()       { fmt.Println("stop") }
func (dc dummyCallback) OnConnect()    { fmt.Println("connect") }
func (dc dummyCallback) OnDisconnect() { fmt.Println("disconnect") }

const (
	serverPort = 30000
	clientPort = 30001
	protocolId = 0x99887766
	deltaTime  = time.Duration(250) * time.Millisecond
	sendRate   = time.Duration(250) * time.Millisecond
	timeout    = time.Duration(10) * time.Second
)

func main() {
	connection := udpnet.NewConn(dummyCallback{}, protocolId, timeout)

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
