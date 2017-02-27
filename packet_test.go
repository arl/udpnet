package udpnet

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcks(t *testing.T) {
	const (
		DeltaTime   = time.Millisecond
		TimeOut     = time.Duration(100) * time.Millisecond
		PacketCount = 100
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

	var (
		clientAckedPackets [PacketCount]bool
		serverAckedPackets [PacketCount]bool
		allPacketsAcked    bool
	)

	for {
		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}
		if allPacketsAcked {
			break
		}

		var ackPacket [256]byte
		for i := range ackPacket {
			ackPacket[i] = byte(i)
		}

		client.SendPacket(ackPacket[:])
		server.SendPacket(ackPacket[:])

		for {
			var packet [256]byte
			bytesRead := client.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
			assert.EqualValues(t, bytesRead, 256)
			for i := range packet {
				assert.EqualValues(t, packet[i], i)
			}
		}

		for {
			var packet [256]byte
			bytesRead := server.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
			assert.EqualValues(t, bytesRead, 256)
			for i := range packet {
				assert.EqualValues(t, packet[i], i)
			}
		}

		var acks []uint

		acks = client.ReliabilitySystem().GetAcks()
		for _, ack := range acks {
			if ack < PacketCount {
				assert.False(t, clientAckedPackets[ack])
				clientAckedPackets[ack] = true
			}
		}

		acks = server.ReliabilitySystem().GetAcks()
		for _, ack := range acks {
			if ack < PacketCount {
				assert.False(t, serverAckedPackets[ack])
				serverAckedPackets[ack] = true
			}
		}

		var clientAckCount, serverAckCount uint
		for i := 0; i < PacketCount; i++ {
			if clientAckedPackets[i] {
				clientAckCount++
			}
			if serverAckedPackets[i] {
				serverAckCount++
			}
		}
		allPacketsAcked = clientAckCount == PacketCount && serverAckCount == PacketCount

		client.Update(DeltaTime)
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}

func TestAckBits(t *testing.T) {
	const (
		DeltaTime   = time.Millisecond
		TimeOut     = time.Duration(100) * time.Millisecond
		PacketCount = 100
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

	var (
		clientAckedPackets [PacketCount]bool
		serverAckedPackets [PacketCount]bool
		allPacketsAcked    bool
	)

	for {
		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}
		if allPacketsAcked {
			break
		}

		var ackPacket [256]byte
		for i := range ackPacket {
			ackPacket[i] = byte(i)
		}

		for i := 0; i < 10; i++ {
			client.SendPacket(ackPacket[:])

			for {
				var packet [256]byte
				bytesRead := client.ReceivePacket(packet[:])
				if bytesRead == 0 {
					break
				}
				assert.EqualValues(t, bytesRead, 256)
				for i := range packet {
					assert.EqualValues(t, packet[i], i)
				}
			}

			var acks []uint
			acks = client.ReliabilitySystem().GetAcks()
			for _, ack := range acks {
				if ack < PacketCount {
					assert.False(t, clientAckedPackets[ack])
					clientAckedPackets[ack] = true
				}
			}

			client.Update(DeltaTime / 10)
			validateReliabilitySystem(t, client.reliabilitySystem)
		}

		server.SendPacket(ackPacket[:])

		for {
			var packet [256]byte
			bytesRead := server.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
			assert.EqualValues(t, bytesRead, 256)
			for i := range packet {
				assert.EqualValues(t, packet[i], i)
			}
		}

		var acks []uint
		acks = server.ReliabilitySystem().GetAcks()
		for _, ack := range acks {
			if ack < PacketCount {
				assert.False(t, serverAckedPackets[ack])
				serverAckedPackets[ack] = true
			}
		}

		var clientAckCount, serverAckCount uint
		for i := 0; i < PacketCount; i++ {
			if clientAckedPackets[i] {
				clientAckCount++
			}
			if serverAckedPackets[i] {
				serverAckCount++
			}
		}
		allPacketsAcked = clientAckCount == PacketCount && serverAckCount == PacketCount

		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}

func TestPacketLoss(t *testing.T) {
	const (
		DeltaTime   = time.Millisecond
		TimeOut     = time.Duration(100) * time.Millisecond
		PacketCount = 100
	)

	client := NewReliableConn(protocolId, TimeOut, maxSequence)
	client.SetPacketLossMask(1)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolId, TimeOut, maxSequence)
	server.SetPacketLossMask(1)
	require.True(t, server.Start(serverPort), "couldn't start server connection")
	defer server.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	client.Connect(cAddr)
	server.Listen()

	var (
		clientAckedPackets [PacketCount]bool
		serverAckedPackets [PacketCount]bool
		allPacketsAcked    bool
	)

	for {
		if !client.IsConnecting() && client.ConnectFailed() {
			break
		}
		if allPacketsAcked {
			break
		}

		var ackPacket [256]byte
		for i := range ackPacket {
			ackPacket[i] = byte(i)
		}

		for i := 0; i < 10; i++ {
			client.SendPacket(ackPacket[:])

			for {
				var packet [256]byte
				bytesRead := client.ReceivePacket(packet[:])
				if bytesRead == 0 {
					break
				}
				assert.EqualValues(t, bytesRead, 256)
				for i := range packet {
					assert.EqualValues(t, packet[i], i)
				}
			}

			var acks []uint
			acks = client.ReliabilitySystem().GetAcks()
			for _, ack := range acks {
				if ack < PacketCount {
					assert.False(t, clientAckedPackets[ack])
					assert.EqualValues(t, ack&1, 0)
					clientAckedPackets[ack] = true
				}
			}

			client.Update(DeltaTime / 10)
			validateReliabilitySystem(t, client.reliabilitySystem)
		}

		server.SendPacket(ackPacket[:])

		for {
			var packet [256]byte
			bytesRead := server.ReceivePacket(packet[:])
			if bytesRead == 0 {
				break
			}
			assert.EqualValues(t, bytesRead, 256)
			for i := range packet {
				assert.EqualValues(t, packet[i], i)
			}
		}

		var acks []uint
		acks = server.ReliabilitySystem().GetAcks()
		for _, ack := range acks {
			if ack < PacketCount {
				assert.False(t, serverAckedPackets[ack])
				assert.EqualValues(t, ack&1, 0)
				serverAckedPackets[ack] = true
			}
		}

		var clientAckCount, serverAckCount uint
		for i := 0; i < PacketCount; i++ {
			if (i & 1) != 0 {
				assert.False(t, clientAckedPackets[i])
				assert.False(t, serverAckedPackets[i])
			}

			if clientAckedPackets[i] {
				clientAckCount++
			}
			if serverAckedPackets[i] {
				serverAckCount++
			}
		}
		allPacketsAcked = clientAckCount == PacketCount/2 && serverAckCount == PacketCount/2

		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}
