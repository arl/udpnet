package udp

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyCallback struct{}

func (dc dummyCallback) OnStart()      {}
func (dc dummyCallback) OnStop()       {}
func (dc dummyCallback) OnConnect()    {}
func (dc dummyCallback) OnDisconnect() {}

const (
	serverPort = 30000
	clientPort = 30001
	protocolId = 0x11112222
)

var (
	clientPacket = []byte("client to server")
	serverPacket = []byte("server to client")
	busyPacket   = []byte("i'm so busy!")
)

func TestConnectionJoin(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(1500) * time.Millisecond
	)

	client := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, server.Start(serverPort), "couldn't start server connection")
	defer server.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	client.Connect(cAddr)
	server.Listen()

	for {
		if client.IsConnected() && server.IsConnected() {
			break
		}

		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}

		client.SendPacket(clientPacket)
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
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(100) * time.Millisecond
	)

	client := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	client.Connect(cAddr)

	for {
		if !client.IsConnecting() {
			break
		}

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

func TestConnectionJoinBusy(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(100) * time.Millisecond
	)

	// connect client to server

	client := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, server.Start(serverPort), "couldn't start server connection")
	defer server.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	client.Connect(cAddr)
	server.Listen()

	for {
		if client.IsConnected() && server.IsConnected() {
			break
		}

		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}

		client.SendPacket(clientPacket)
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

	// attempt another connection, verify connect fails (busy)
	busy := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, busy.Start(clientPort+1), "couldn't start busy connection")
	defer busy.Stop()

	bAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	busy.Connect(bAddr)

	for {
		if !busy.IsConnecting() || busy.IsConnected() {
			break
		}

		client.SendPacket(clientPacket)
		server.SendPacket(serverPacket)
		busy.SendPacket(busyPacket)

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

		for {
			var packet [256]byte
			bytesRead := busy.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
		}

		client.Update(DeltaTime)
		server.Update(DeltaTime)
		busy.Update(DeltaTime)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
	assert.False(t, busy.IsConnected(), "busy should not be connected")
	assert.True(t, busy.ConnectFailed(), "busy.ConnectFailed() should return true")
}

func TestConnectionRejoin(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(100) * time.Millisecond
	)

	// connect client to server

	client := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, server.Start(serverPort), "couldn't start server connection")
	defer server.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	client.Connect(cAddr)
	server.Listen()

	for {
		if client.IsConnected() && server.IsConnected() {
			break
		}

		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}

		client.SendPacket(clientPacket)
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

	// let connection timeout

	for client.IsConnected() || server.IsConnected() {
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

	assert.False(t, client.IsConnected(), "client should not be connected")
	assert.False(t, server.IsConnected(), "server should not be connected")

	// reconnect client

	client.Connect(cAddr)

	for {
		if client.IsConnected() && server.IsConnected() {
			break
		}

		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}

		client.SendPacket(clientPacket)
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

func TestConnectionPayload(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(100) * time.Millisecond
	)

	client := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewConn(dummyCallback{}, protocolId, TimeOut)
	require.True(t, server.Start(serverPort), "couldn't start server connection")
	defer server.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	client.Connect(cAddr)
	server.Listen()

	for {
		if client.IsConnected() && server.IsConnected() {
			break
		}

		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}

		client.SendPacket(clientPacket)
		server.SendPacket(serverPacket)

		for {
			var packet [256]byte
			bytesRead := client.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
			assert.Equal(t, "server to client", string(bytes.Trim(packet[:], "\x00")))
		}

		for {
			var packet [256]byte
			bytesRead := server.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
			assert.Equal(t, "client to server", string(bytes.Trim(packet[:], "\x00")))
		}

		client.Update(DeltaTime)
		server.Update(DeltaTime)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}
