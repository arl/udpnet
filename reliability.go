package udpnet

import (
	"fmt"
	"time"
)

// reliability system to support reliable connection
//  + manages sent, received, pending ack and acked packet queues
//  + separated out from reliable connection so they can be unit-tested
type ReliabilitySystem struct {
	maxSequence    uint // maximum sequence value before wrap around (used to test sequence wrap at low # values)
	localSequence  uint // local sequence number for most recently sent packet
	remoteSequence uint // remote sequence number for most recently received packet

	sentPackets  uint // total number of packets sent
	recvPackets  uint // total number of packets received
	lostPackets  uint // total number of packets lost
	ackedPackets uint // total number of packets acked

	sentBandwidth  float64       // approximate sent bandwidth over the last second
	ackedBandwidth float64       // approximate acked bandwidth over the last second
	rtt            time.Duration // estimated round trip time
	rttMax         time.Duration // maximum expected round trip time (hard coded to one second for the moment)

	acks []uint // acked packets from last set of packet receives. cleared each update!

	sentQueue       PacketQueue // sent packets used to calculate sent bandwidth (kept until rttMax)
	pendingAckQueue PacketQueue // sent packets which have not been acked yet (kept until rttMax * 2 )
	receivedQueue   PacketQueue // received packets for determining acks to send (kept up to most recent recv sequence - 32)
	ackedQueue      PacketQueue // acked packets (kept until rttMax * 2)
}

//
func NewReliabilitySystem(maxSequence uint) *ReliabilitySystem {
	rs := &ReliabilitySystem{
		rttMax:      1 * time.Second,
		maxSequence: maxSequence,
	}
	rs.Reset()
	return rs
}

func (rs *ReliabilitySystem) Reset() {
	rs.localSequence = 0
	rs.remoteSequence = 0
	rs.sentQueue = PacketQueue{}
	rs.receivedQueue = PacketQueue{}
	rs.pendingAckQueue = PacketQueue{}
	rs.ackedQueue = PacketQueue{}
	rs.sentPackets = 0
	rs.recvPackets = 0
	rs.lostPackets = 0
	rs.ackedPackets = 0
	rs.sentBandwidth = 0.0
	rs.ackedBandwidth = 0.0
	rs.rtt = 0 * time.Second
	rs.rttMax = 1 * time.Second
}

func (rs *ReliabilitySystem) PacketSent(size int) {
	if rs.sentQueue.Exists(rs.localSequence) {
		fmt.Printf("local sequence %d exists\n", rs.localSequence)
		for i := 0; i < len(rs.sentQueue); i++ {
			fmt.Printf(" + %d\n", rs.sentQueue[i].sequence)
		}
	}
	// TODO: remove after debugging/testing
	if rs.sentQueue.Exists(rs.localSequence) {
		panic("assert( !sentQueue.exists( localSequence ) )")
	}
	if rs.pendingAckQueue.Exists(rs.localSequence) {
		panic("assert( !pendingAckQueue.exists( localSequence ) )")
	}

	var data PacketData
	data.sequence = rs.localSequence
	data.time = 0
	data.size = size
	rs.sentQueue = append(rs.sentQueue, data)
	rs.pendingAckQueue = append(rs.pendingAckQueue, data)
	rs.sentPackets++
	rs.localSequence++
	if rs.localSequence > rs.maxSequence {
		rs.localSequence = 0
	}
}

func (rs *ReliabilitySystem) PacketReceived(sequence uint, size int) {
	rs.recvPackets++
	if rs.receivedQueue.Exists(sequence) {
		return
	}
	var data PacketData
	data.sequence = sequence
	data.time = 0
	data.size = size
	rs.receivedQueue = append(rs.receivedQueue, data)
	if sequenceMoreRecent(sequence, rs.remoteSequence, rs.maxSequence) {
		rs.remoteSequence = sequence
	}
}

func (rs *ReliabilitySystem) GenerateAckBits() uint {
	return generateAckBits(rs.remoteSequence, &rs.receivedQueue, rs.maxSequence)
}

func (rs *ReliabilitySystem) ProcessAck(ack, ackBits uint) {
	processAck(ack, ackBits, &rs.pendingAckQueue, &rs.ackedQueue, &rs.acks, &rs.ackedPackets, &rs.rtt, rs.maxSequence)
}

func (rs *ReliabilitySystem) Update(deltaTime time.Duration) {
	rs.acks = []uint{}
	rs.advanceQueueTime(deltaTime)
	rs.updateQueues()
	rs.updateStats()
}

// data accessors

func (rs *ReliabilitySystem) LocalSequence() uint {
	return rs.localSequence
}

func (rs *ReliabilitySystem) RemoteSequence() uint {
	return rs.remoteSequence
}

func (rs *ReliabilitySystem) MaxSequence() uint {
	return rs.maxSequence
}

func (rs *ReliabilitySystem) Acks() []uint {
	return rs.acks
}

func (rs *ReliabilitySystem) SentPackets() uint {
	return rs.sentPackets
}

func (rs *ReliabilitySystem) ReceivedPackets() uint {
	return rs.recvPackets
}

func (rs *ReliabilitySystem) LostPackets() uint {
	return rs.lostPackets
}

func (rs *ReliabilitySystem) AckedPackets() uint {
	return rs.ackedPackets
}

func (rs *ReliabilitySystem) SentBandwidth() float64 {
	return rs.sentBandwidth
}

func (rs *ReliabilitySystem) AckedBandwidth() float64 {
	return rs.ackedBandwidth
}

func (rs *ReliabilitySystem) RoundTripTime() time.Duration {
	return rs.rtt
}

func (rs *ReliabilitySystem) HeaderSize() int {
	return 12
}

func (rs *ReliabilitySystem) advanceQueueTime(deltaTime time.Duration) {
	for i := 0; i < len(rs.sentQueue); i++ {
		rs.sentQueue[i].time += deltaTime
	}
	for i := 0; i < len(rs.receivedQueue); i++ {
		rs.receivedQueue[i].time += deltaTime
	}
	for i := 0; i < len(rs.pendingAckQueue); i++ {
		rs.pendingAckQueue[i].time += deltaTime
	}
	for i := 0; i < len(rs.ackedQueue); i++ {
		rs.ackedQueue[i].time += deltaTime
	}
}

func (rs *ReliabilitySystem) updateQueues() {
	const epsilon = 1 * time.Millisecond

	for len(rs.sentQueue) > 0 && rs.sentQueue[0].time > rs.rttMax+epsilon {
		// pop front
		rs.sentQueue = rs.sentQueue[1:]
	}

	if len(rs.receivedQueue) > 0 {
		latestSequence := rs.receivedQueue[len(rs.receivedQueue)-1].sequence

		var minSequence uint
		if latestSequence >= 34 {
			minSequence = (latestSequence - 34)
		} else {
			minSequence = rs.maxSequence - (34 - latestSequence)
		}

		for len(rs.receivedQueue) > 0 && !sequenceMoreRecent(rs.receivedQueue[0].sequence, minSequence, rs.maxSequence) {
			// pop front
			rs.receivedQueue = rs.receivedQueue[1:]
		}
	}

	for len(rs.ackedQueue) > 0 && rs.ackedQueue[0].time > rs.rttMax*2-epsilon {
		// pop front
		rs.ackedQueue = rs.ackedQueue[1:]
	}

	for len(rs.pendingAckQueue) > 0 && rs.pendingAckQueue[0].time > rs.rttMax+epsilon {
		// pop front
		rs.pendingAckQueue = rs.pendingAckQueue[1:]
		rs.lostPackets++
	}
}

func (rs *ReliabilitySystem) updateStats() {
	var sentBytesPerSecond float64
	for i := 0; i < len(rs.sentQueue); i++ {
		sentBytesPerSecond += float64(rs.sentQueue[i].size)
	}
	var (
		ackedPacketsPerSecond float64
		ackedBytesPerSecond   float64
	)
	for i := 0; i < len(rs.ackedQueue); i++ {
		if rs.ackedQueue[i].time >= rs.rttMax {
			ackedPacketsPerSecond++
			ackedBytesPerSecond += float64(rs.ackedQueue[i].size)
		}
	}
	sentBytesPerSecond /= float64(rs.rttMax)
	ackedBytesPerSecond /= float64(rs.rttMax)
	rs.sentBandwidth = sentBytesPerSecond * (8.0 / 1000.0)
	rs.ackedBandwidth = ackedBytesPerSecond * (8.0 / 1000.0)
}
