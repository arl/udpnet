package main

import (
	"fmt"
	"net"
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
	DeltaTime  = 0.25
	SendRate   = 0.25
	TimeOut    = 10.0
)

func main() {

	connection := udp.NewConn(DummyCallback{}, ProtocolId, TimeOut)

	if !connection.Start(ClientPort) {
		fmt.Printf("could not start connection on port %d\n", ClientPort)
		return
	}
	defer connection.Stop()

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", ServerPort))
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

		connection.Update(DeltaTime)

		time.Sleep(time.Duration(float64(time.Second) * DeltaTime))
	}
}
