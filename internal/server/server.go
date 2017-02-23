package main

import (
	"fmt"
	"time"

	"github.com/aurelien-rainone/udp"
)

type DummyCallback struct{}

func (dc DummyCallback) OnStart() {
	fmt.Println("start")
}
func (dc DummyCallback) OnStop() {
	fmt.Println("stop")
}
func (dc DummyCallback) OnConnect() {
	fmt.Println("connect")
}
func (dc DummyCallback) OnDisconnect() {
	fmt.Println("disconnect")
}

const (
	ServerPort = 30000
	ClientPort = 30001
	ProtocolId = 0x99887766
	DeltaTime  = time.Duration(250) * time.Millisecond
	SendRate   = time.Duration(250) * time.Millisecond
	TimeOut    = time.Duration(10) * time.Second
)

func main() {

	connection := udp.NewConn(DummyCallback{}, ProtocolId, TimeOut)

	if !connection.Start(ServerPort) {
		fmt.Printf("could not start connection on port %d\n", ServerPort)
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

		connection.Update(DeltaTime)

		time.Sleep(DeltaTime)
	}
}
