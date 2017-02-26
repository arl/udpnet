package udpnet

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxSequence = 0xFFFFFFFF

func TestReliableConnectionJoin(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(1000) * time.Millisecond
	)

	client := NewReliableConn(protocolId, TimeOut, maxSequence)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolId, TimeOut, maxSequence)
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
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}

func TestReliableConnectionJoinTimeout(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(100) * time.Millisecond
	)

	client := NewReliableConn(protocolId, TimeOut, maxSequence)
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
		validateReliabilitySystem(t, client.reliabilitySystem)
	}

	assert.False(t, client.IsConnected(), "client should not be connected")
	assert.True(t, client.ConnectFailed(), "client.ConnectFailed() should return true")
}

func TestReliableConnectionJoinBusy(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(100) * time.Millisecond
	)

	// connect client to server

	client := NewReliableConn(protocolId, TimeOut, maxSequence)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolId, TimeOut, maxSequence)
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
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")

	// attempt another connection, verify connect fails (busy)
	busy := NewReliableConn(protocolId, TimeOut, maxSequence)
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
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
		busy.Update(DeltaTime)
		validateReliabilitySystem(t, busy.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
	assert.False(t, busy.IsConnected(), "busy should not be connected")
	assert.True(t, busy.ConnectFailed(), "busy.ConnectFailed() should return true")
}

func TestReliableConnectionRejoin(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(100) * time.Millisecond
	)

	// connect client to server

	client := NewReliableConn(protocolId, TimeOut, maxSequence)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolId, TimeOut, maxSequence)
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
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
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
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
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
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}

func TestReliableConnectionPayload(t *testing.T) {
	const (
		DeltaTime = time.Millisecond
		TimeOut   = time.Duration(100) * time.Millisecond
	)

	client := NewReliableConn(protocolId, TimeOut, maxSequence)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolId, TimeOut, maxSequence)
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
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}
