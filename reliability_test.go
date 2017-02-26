package udpnet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func validateReliabilitySystem(t *testing.T, rs *ReliabilitySystem) {
	verifySorted(t, rs.sentQueue, rs.maxSequence)
	verifySorted(t, rs.receivedQueue, rs.maxSequence)
	verifySorted(t, rs.pendingAckQueue, rs.maxSequence)
	verifySorted(t, rs.ackedQueue, rs.maxSequence)
}

func TestReliabilitySystem(t *testing.T) {
	const MaximumSequence = 255

	t.Logf("check bit index for sequence\n")
	assert.EqualValues(t, bitIndexForSequence(99, 100, MaximumSequence), 0)
	assert.EqualValues(t, bitIndexForSequence(90, 100, MaximumSequence), 9)
	assert.EqualValues(t, bitIndexForSequence(0, 1, MaximumSequence), 0)
	assert.EqualValues(t, bitIndexForSequence(255, 0, MaximumSequence), 0)
	assert.EqualValues(t, bitIndexForSequence(255, 1, MaximumSequence), 1)
	assert.EqualValues(t, bitIndexForSequence(254, 1, MaximumSequence), 2)
	assert.EqualValues(t, bitIndexForSequence(254, 2, MaximumSequence), 3)

	t.Logf("check generate ack bits\n")
	var packetQueue PacketQueue
	for i := 0; i < 32; i++ {
		var data PacketData
		data.sequence = uint(i)
		packetQueue.InsertSorted(data, MaximumSequence)
		verifySorted(t, packetQueue, MaximumSequence)
	}
	assert.EqualValues(t, generateAckBits(32, &packetQueue, MaximumSequence), 0xFFFFFFFF)
	assert.EqualValues(t, generateAckBits(31, &packetQueue, MaximumSequence), 0x7FFFFFFF)
	assert.EqualValues(t, generateAckBits(33, &packetQueue, MaximumSequence), 0xFFFFFFFE)
	assert.EqualValues(t, generateAckBits(16, &packetQueue, MaximumSequence), 0x0000FFFF)
	assert.EqualValues(t, generateAckBits(48, &packetQueue, MaximumSequence), 0xFFFF0000)

	t.Logf("check generate ack bits with wrap\n")
	packetQueue = PacketQueue{}
	for i := 255 - 31; i <= 255; i++ {
		var data PacketData
		data.sequence = uint(i)
		packetQueue.InsertSorted(data, MaximumSequence)
		verifySorted(t, packetQueue, MaximumSequence)
	}
	assert.Len(t, packetQueue, 32)
	assert.EqualValues(t, generateAckBits(0, &packetQueue, MaximumSequence), 0xFFFFFFFF)
	assert.EqualValues(t, generateAckBits(255, &packetQueue, MaximumSequence), 0x7FFFFFFF)
	assert.EqualValues(t, generateAckBits(1, &packetQueue, MaximumSequence), 0xFFFFFFFE)
	assert.EqualValues(t, generateAckBits(240, &packetQueue, MaximumSequence), 0x0000FFFF)
	assert.EqualValues(t, generateAckBits(16, &packetQueue, MaximumSequence), 0xFFFF0000)

	t.Logf("check process ack (1)\n")
	{
		var pendingAckQueue PacketQueue
		for i := 0; i < 33; i++ {
			var data PacketData
			data.sequence = uint(i)
			data.time = 0.0
			pendingAckQueue.InsertSorted(data, MaximumSequence)
			verifySorted(t, pendingAckQueue, MaximumSequence)
		}
		var (
			ackedQueue    PacketQueue
			acks          []uint
			rtt           time.Duration
			acked_packets uint
		)
		processAck(32, 0xFFFFFFFF, &pendingAckQueue, &ackedQueue, &acks, &acked_packets, &rtt, MaximumSequence)
		assert.Len(t, acks, 33)
		assert.EqualValues(t, acked_packets, 33)
		assert.Len(t, ackedQueue, 33)
		assert.Len(t, pendingAckQueue, 0)
		verifySorted(t, ackedQueue, MaximumSequence)
		for i, ack := range acks {
			assert.EqualValues(t, ack, i)
		}
		for i, pkt := range ackedQueue {
			assert.EqualValues(t, pkt.sequence, i)
		}
	}

	t.Logf("check process ack (2)\n")
	{
		var pendingAckQueue PacketQueue
		for i := 0; i < 33; i++ {
			var data PacketData
			data.sequence = uint(i)
			data.time = 0.0
			pendingAckQueue.InsertSorted(data, MaximumSequence)
			verifySorted(t, pendingAckQueue, MaximumSequence)
		}
		var (
			ackedQueue    PacketQueue
			acks          []uint
			rtt           time.Duration
			acked_packets uint
		)
		processAck(32, 0x0000FFFF, &pendingAckQueue, &ackedQueue, &acks, &acked_packets, &rtt, MaximumSequence)
		assert.Len(t, acks, 17)
		assert.EqualValues(t, acked_packets, 17)
		assert.Len(t, ackedQueue, 17)
		assert.Len(t, pendingAckQueue, 33-17)
		verifySorted(t, ackedQueue, MaximumSequence)
		for i, pkt := range pendingAckQueue {
			assert.EqualValues(t, pkt.sequence, i)
		}
		for i, pkt := range ackedQueue {
			assert.EqualValues(t, pkt.sequence, i+16)
		}
		for i, ack := range acks {
			assert.EqualValues(t, ack, i+16)
		}
	}

	t.Logf("check process ack (3)\n")
	{
		var pendingAckQueue PacketQueue
		for i := 0; i < 32; i++ {
			var data PacketData
			data.sequence = uint(i)
			data.time = 0.0
			pendingAckQueue.InsertSorted(data, MaximumSequence)
			verifySorted(t, pendingAckQueue, MaximumSequence)
		}
		var (
			ackedQueue    PacketQueue
			acks          []uint
			rtt           time.Duration
			acked_packets uint
		)
		processAck(48, 0xFFFF0000, &pendingAckQueue, &ackedQueue, &acks, &acked_packets, &rtt, MaximumSequence)
		assert.Len(t, acks, 16)
		assert.EqualValues(t, acked_packets, 16)
		assert.Len(t, ackedQueue, 16)
		assert.Len(t, pendingAckQueue, 16)
		verifySorted(t, ackedQueue, MaximumSequence)
		for i, pkt := range pendingAckQueue {
			assert.EqualValues(t, pkt.sequence, i)
		}
		for i, pkt := range ackedQueue {
			assert.EqualValues(t, pkt.sequence, i+16)
		}
		for i, ack := range acks {
			assert.EqualValues(t, ack, i+16)
		}

	}

	t.Logf("check process ack wrap around (1)\n")
	{
		var pendingAckQueue PacketQueue
		for i := 255 - 31; i <= 256; i++ {
			var data PacketData
			data.sequence = uint(i & 0xFF)
			data.time = 0
			pendingAckQueue.InsertSorted(data, MaximumSequence)
			verifySorted(t, pendingAckQueue, MaximumSequence)
		}
		assert.Len(t, pendingAckQueue, 33)

		var (
			ackedQueue    PacketQueue
			acks          []uint
			rtt           time.Duration
			acked_packets uint
		)
		processAck(0, 0xFFFFFFFF, &pendingAckQueue, &ackedQueue, &acks, &acked_packets, &rtt, MaximumSequence)
		assert.Len(t, acks, 33)
		assert.EqualValues(t, acked_packets, 33)
		assert.Len(t, ackedQueue, 33)
		assert.Len(t, pendingAckQueue, 0)
		verifySorted(t, ackedQueue, MaximumSequence)
		for i, ack := range acks {
			assert.EqualValues(t, ack, (i+255-31)&0xFF)
		}
		for i, pkt := range ackedQueue {
			assert.EqualValues(t, pkt.sequence, (i+255-31)&0xFF)
		}
	}

	t.Logf("check process ack wrap around (2)\n")
	{
		var pendingAckQueue PacketQueue
		for i := 255 - 31; i <= 256; i++ {
			var data PacketData
			data.sequence = uint(i & 0xFF)
			data.time = 0
			pendingAckQueue.InsertSorted(data, MaximumSequence)
			verifySorted(t, pendingAckQueue, MaximumSequence)
		}
		assert.Len(t, pendingAckQueue, 33)

		var (
			ackedQueue    PacketQueue
			acks          []uint
			rtt           time.Duration
			acked_packets uint
		)
		processAck(0, 0x0000FFFF, &pendingAckQueue, &ackedQueue, &acks, &acked_packets, &rtt, MaximumSequence)
		assert.Len(t, acks, 17)
		assert.EqualValues(t, acked_packets, 17)
		assert.Len(t, ackedQueue, 17)
		assert.Len(t, pendingAckQueue, 33-17)
		verifySorted(t, ackedQueue, MaximumSequence)
		for i, ack := range acks {
			assert.EqualValues(t, ack, (i+255-15)&0xFF)
		}
		for i, pkt := range pendingAckQueue {
			assert.EqualValues(t, pkt.sequence, i+255-31)
		}
		for i, pkt := range ackedQueue {
			assert.EqualValues(t, pkt.sequence, ((i + 255 - 15) & 0xFF))
		}
	}

	t.Logf("check process ack wrap around (3)\n")
	{
		var pendingAckQueue PacketQueue
		for i := 255 - 31; i <= 255; i++ {
			var data PacketData
			data.sequence = uint(i & 0xFF)
			data.time = 0
			pendingAckQueue.InsertSorted(data, MaximumSequence)
			verifySorted(t, pendingAckQueue, MaximumSequence)
		}
		assert.Len(t, pendingAckQueue, 32)

		var (
			ackedQueue    PacketQueue
			acks          []uint
			rtt           time.Duration
			acked_packets uint
		)
		processAck(16, 0xFFFF0000, &pendingAckQueue, &ackedQueue, &acks, &acked_packets, &rtt, MaximumSequence)
		assert.Len(t, acks, 16)
		assert.EqualValues(t, acked_packets, 16)
		assert.Len(t, ackedQueue, 16)
		assert.Len(t, pendingAckQueue, 16)
		verifySorted(t, ackedQueue, MaximumSequence)

		for i, ack := range acks {
			assert.EqualValues(t, ack, (i+255-15)&0xFF)
		}
		for i, pkt := range pendingAckQueue {
			assert.EqualValues(t, pkt.sequence, i+255-31)
		}
		for i, pkt := range ackedQueue {
			assert.EqualValues(t, pkt.sequence, ((i + 255 - 15) & 0xFF))
		}
	}
}
