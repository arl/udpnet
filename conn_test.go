package udp

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyCallback struct{}

func (dc dummyCallback) OnStart() {
	fmt.Println("start")
}
func (dc dummyCallback) OnStop() {
	fmt.Println("stop")
}
func (dc dummyCallback) OnConnect() {
	fmt.Println("connect")
}
func (dc dummyCallback) OnDisconnect() {
	fmt.Println("disconnect")
}

func TestConnectionJoin(t *testing.T) {
	const (
		ServerPort = 30000
		ClientPort = 30001
		ProtocolId = 0x11112222
		DeltaTime  = 0.001
		TimeOut    = 1.5
	)

	client := NewConn(dummyCallback{}, ProtocolId, TimeOut)
	require.True(t, client.Start(ClientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewConn(dummyCallback{}, ProtocolId, TimeOut)
	require.True(t, server.Start(ServerPort), "couldn't start server connection")
	defer server.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", ServerPort))
	client.Connect(cAddr)
	server.Listen()

	for {
		if client.IsConnected() && server.IsConnected() {
			break
		}

		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}

		clientPacket := []byte("client to server")
		client.SendPacket(clientPacket)

		serverPacket := []byte("server to client")
		server.SendPacket(serverPacket)

		for {
			var packet [256]byte
			bytesRead := client.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
		}

		for {
			var packet [256]byte
			bytesRead := server.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
		}

		client.Update(DeltaTime)
		server.Update(DeltaTime)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}

func TestConnectionJoinTimeout(t *testing.T) {
	const (
		ServerPort = 30000
		ClientPort = 30001
		ProtocolId = 0x11112222
		DeltaTime  = 0.001
		TimeOut    = 0.1
	)

	client := NewConn(dummyCallback{}, ProtocolId, TimeOut)
	require.True(t, client.Start(ClientPort), "couldn't start client connection")
	defer client.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", ServerPort))
	client.Connect(cAddr)

	for {
		if !client.IsConnecting() {
			break
		}

		clientPacket := []byte("client to server")
		client.SendPacket(clientPacket)

		for {
			var packet [256]byte
			bytesRead := client.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
		}

		client.Update(DeltaTime)
	}

	assert.False(t, client.IsConnected(), "client should not be connected")
	assert.True(t, client.ConnectFailed(), "client.ConnectFailed() should return true")
}
