package udpnet

import (
	"fmt"
	"time"
)

// ReliableConn represents a connection between two distant parties, with
// reliability handled by SEQ/ACK
type ReliableConn struct {
	Conn

	// reliability system: manages sequence numbers and acks, tracks network
	// stats etc.
	reliabilitySystem *ReliabilitySystem

	// TODO: this is for unit test only
	packetLossMask uint // mask sequence number, if non-zero, drop packet
}

type clearDataCB struct{ c *ReliableConn }

func (cb *clearDataCB) OnStart()      {}
func (cb *clearDataCB) OnStop()       { cb.c.clearData() }
func (cb *clearDataCB) OnConnect()    {}
func (cb *clearDataCB) OnDisconnect() { cb.c.clearData() }

// NewReliableConn returns a new reliable connection
func NewReliableConn(protocolID uint, timeout time.Duration, maxSequence uint) *ReliableConn {
	c := &ReliableConn{
		Conn: Conn{
			protocolID: protocolID,
			timeout:    timeout,
			mode:       None,
			running:    false,
		},
		reliabilitySystem: NewReliabilitySystem(maxSequence),
	}

	c.clearData()
	// provide a callback that calls clearData on stop/disconnect
	c.Conn.cb = &clearDataCB{c}
	return c
}

func (c *ReliableConn) clearData() {
	c.reliabilitySystem.Reset()
}

func (c *ReliableConn) SendPacket(data []byte) bool {
	// TODO
	//#ifdef NET_UNIT_TEST
	if (c.reliabilitySystem.LocalSequence() & c.packetLossMask) != 0 {
		c.reliabilitySystem.PacketSent(len(data))
		return true
	}
	//#endif
	const header = 12
	packet := make([]byte, header+len(data))
	seq := c.reliabilitySystem.LocalSequence()
	ack := c.reliabilitySystem.RemoteSequence()
	ackBits := c.reliabilitySystem.GenerateAckBits()
	c.WriteHeader(packet, seq, ack, ackBits)
	copy(packet[header:], data)
	if err := c.Conn.SendPacket(packet); err != nil {
		fmt.Printf("couldn't send packet, %v\n", err)
		return false
	}
	c.reliabilitySystem.PacketSent(len(data))
	return true
}

func (c *ReliableConn) ReceivePacket(data []byte) int {
	const header = 12
	if len(data) <= header {
		return 0
	}
	packet := make([]byte, header+len(data))
	receivedBytes := c.Conn.ReceivePacket(packet)
	if receivedBytes == 0 {
		return 0
	}
	if receivedBytes <= header {
		return 0
	}
	var packetSequence, packetAck, packetAckBits uint
	packetSequence, packetAck, packetAckBits = c.ReadHeader(packet)
	c.reliabilitySystem.PacketReceived(packetSequence, receivedBytes-header)
	c.reliabilitySystem.ProcessAck(packetAck, packetAckBits)
	copy(data, packet[header:receivedBytes])
	return receivedBytes - header
}
func (c *ReliableConn) Update(deltaTime time.Duration) {
	c.Conn.Update(deltaTime)
	c.reliabilitySystem.Update(deltaTime)
}

func (c *ReliableConn) HeaderSize() int {
	return c.Conn.HeaderSize() + c.reliabilitySystem.HeaderSize()
}

func (c *ReliableConn) ReliabilitySystem() *ReliabilitySystem {
	return c.reliabilitySystem
}

// unit test controls

// TODO: this should only be enabled during unit tests
//#ifdef NET_UNIT_TEST
func (c *ReliableConn) SetPacketLossMask(mask uint) {
	c.packetLossMask = mask
}

//#endif

func (c *ReliableConn) WriteInteger(data []byte, i uint) {
	data[0] = byte(i >> 24)
	data[1] = byte((i >> 16) & 0xFF)
	data[2] = byte((i >> 8) & 0xFF)
	data[3] = byte(i & 0xFF)
}

func (c *ReliableConn) WriteHeader(header []byte, sequence, ack, ackBits uint) {
	c.WriteInteger(header, sequence)
	c.WriteInteger(header[4:], ack)
	c.WriteInteger(header[8:], ackBits)
}

func (c *ReliableConn) ReadInteger(data []byte) uint {
	return ((uint(data[0]) << 24) | (uint(data[1]) << 16) |
		(uint(data[2]) << 8) | (uint(data[3])))
}

func (c *ReliableConn) ReadHeader(header []byte) (sequence, ack, ackBits uint) {
	sequence = c.ReadInteger(header)
	ack = c.ReadInteger(header[4:])
	ackBits = c.ReadInteger(header[8:])
	return
}
