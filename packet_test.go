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

	client := NewReliableConn(protocolID, TimeOut, maxSequence)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolID, TimeOut, maxSequence)
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

		acks = client.ReliabilitySystem().Acks()
		for _, ack := range acks {
			if ack < PacketCount {
				assert.False(t, clientAckedPackets[ack])
				clientAckedPackets[ack] = true
			}
		}

		acks = server.ReliabilitySystem().Acks()
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

	client := NewReliableConn(protocolID, TimeOut, maxSequence)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolID, TimeOut, maxSequence)
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
			acks = client.ReliabilitySystem().Acks()
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
		acks = server.ReliabilitySystem().Acks()
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

	client := NewReliableConn(protocolID, TimeOut, maxSequence)
	client.SetPacketLossMask(1)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolID, TimeOut, maxSequence)
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
			acks = client.ReliabilitySystem().Acks()
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
		acks = server.ReliabilitySystem().Acks()
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

func TestSequenceWrapAround(t *testing.T) {
	const (
		DeltaTime     = 50 * time.Millisecond
		TimeOut       = 1 * time.Second
		PacketCount   = 256
		maxSequence31 = 31 // [0,31]
	)

	client := NewReliableConn(protocolID, TimeOut, maxSequence31)
	require.True(t, client.Start(clientPort), "couldn't start client connection")
	defer client.Stop()

	server := NewReliableConn(protocolID, TimeOut, maxSequence31)
	require.True(t, server.Start(serverPort), "couldn't start server connection")
	defer server.Stop()

	cAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	client.Connect(cAddr)
	server.Listen()

	var (
		clientAckCount  [maxSequence31 + 1]uint
		serverAckCount  [maxSequence31 + 1]uint
		allPacketsAcked bool
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

		server.SendPacket(ackPacket[:])
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
		acks = client.ReliabilitySystem().Acks()
		for _, ack := range acks {
			if ack < PacketCount {
				assert.True(t, ack <= maxSequence31)
				clientAckCount[ack]++
			}
		}

		acks = server.ReliabilitySystem().Acks()
		for _, ack := range acks {
			if ack < PacketCount {
				assert.True(t, ack <= maxSequence31)
				serverAckCount[ack]++
			}
		}

		var totalClientAcks, totalServerAcks uint
		for i := 0; i < maxSequence31; i++ {
			totalClientAcks += clientAckCount[i]
			totalServerAcks += serverAckCount[i]
		}
		allPacketsAcked = totalClientAcks >= PacketCount && totalServerAcks >= PacketCount

		client.Update(DeltaTime)
		validateReliabilitySystem(t, client.reliabilitySystem)
		server.Update(DeltaTime)
		validateReliabilitySystem(t, server.reliabilitySystem)
	}

	assert.True(t, client.IsConnected(), "client should be connected")
	assert.True(t, server.IsConnected(), "server should be connected")
}
