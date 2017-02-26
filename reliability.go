package udpnet

import "fmt"

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

	sentBandwidth  float64 // approximate sent bandwidth over the last second
	ackedBandwidth float64 // approximate acked bandwidth over the last second
	rtt            float64 // estimated round trip time
	rttMax         float64 // maximum expected round trip time (hard coded to one second for the moment)

	acks []uint // acked packets from last set of packet receives. cleared each update!

	sentQueue       PacketQueue // sent packets used to calculate sent bandwidth (kept until rtt_maximum)
	pendingAckQueue PacketQueue // sent packets which have not been acked yet (kept until rtt_maximum * 2 )
	receivedQueue   PacketQueue // received packets for determining acks to send (kept up to most recent recv sequence - 32)
	ackedQueue      PacketQueue // acked packets (kept until rtt_maximum * 2)
}

//
func NewReliabilitySystem(maxSequence uint) *ReliabilitySystem {
	rs := &ReliabilitySystem{
		rttMax:      1.0,
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
	rs.rtt = 0.0
	rs.rttMax = 1.0
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
		panic("assert( !sentQueue.exists( local_sequence ) )")
	}
	if rs.pendingAckQueue.Exists(rs.localSequence) {

		panic("assert( !pendingAckQueue.exists( local_sequence ) )")
	}

	var data PacketData
	data.sequence = rs.localSequence
	data.time = 0.0
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
	data.time = 0.0
	data.size = size
	rs.receivedQueue = append(rs.receivedQueue, data)
	if sequenceMoreRecent(sequence, rs.remoteSequence, rs.maxSequence) {
		rs.remoteSequence = sequence
	}
}

func (rs *ReliabilitySystem) GenerateAckBits() uint {
	return generateAckBits(rs.remoteSequence, &rs.receivedQueue, rs.maxSequence)
}

func (rs *ReliabilitySystem) ProcessAck(ack, ack_bits uint) {
	processAck(ack, ack_bits, &rs.pendingAckQueue, &rs.ackedQueue, &rs.acks, &rs.ackedPackets, &rs.rtt, rs.maxSequence)
}

func (rs *ReliabilitySystem) Update(deltaTime float64) {
	rs.acks = []uint{}
	rs.advanceQueueTime(deltaTime)
	rs.updateQueues()
	rs.updateStats()
}

// data accessors

func (rs *ReliabilitySystem) GetLocalSequence() uint {
	return rs.localSequence
}

func (rs *ReliabilitySystem) GetRemoteSequence() uint {
	return rs.remoteSequence
}

func (rs *ReliabilitySystem) GetMaxSequence() uint {
	return rs.maxSequence
}

func (rs *ReliabilitySystem) GetAcks() []uint {
	return rs.acks
}

func (rs *ReliabilitySystem) GetSentPackets() uint {
	return rs.sentPackets
}

func (rs *ReliabilitySystem) GetReceivedPackets() uint {
	return rs.recvPackets
}

func (rs *ReliabilitySystem) GetLostPackets() uint {
	return rs.lostPackets
}

func (rs *ReliabilitySystem) GetAckedPackets() uint {
	return rs.ackedPackets
}

func (rs *ReliabilitySystem) GetSentBandwidth() float64 {
	return rs.sentBandwidth
}

func (rs *ReliabilitySystem) GetAckedBandwidth() float64 {
	return rs.ackedBandwidth
}

func (rs *ReliabilitySystem) GetRoundTripTime() float64 {
	return rs.rtt
}

func (rs *ReliabilitySystem) GetHeaderSize() int {
	return 12
}

func (rs *ReliabilitySystem) advanceQueueTime(deltaTime float64) {
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
	const epsilon = 0.001

	for len(rs.sentQueue) > 0 && rs.sentQueue[0].time > rs.rttMax+epsilon {
		// pop front
		rs.sentQueue = rs.sentQueue[1:]
	}

	if len(rs.receivedQueue) > 0 {
		latest_sequence := rs.receivedQueue[len(rs.receivedQueue)-1].sequence

		var minimum_sequence uint
		if latest_sequence >= 34 {
			minimum_sequence = (latest_sequence - 34)
		} else {
			minimum_sequence = rs.maxSequence - (34 - latest_sequence)
		}

		for len(rs.receivedQueue) > 0 && !sequenceMoreRecent(rs.receivedQueue[0].sequence, minimum_sequence, rs.maxSequence) {
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
	var sent_bytes_per_second float64
	for i := 0; i < len(rs.sentQueue); i++ {
		sent_bytes_per_second += float64(rs.sentQueue[i].size)
	}
	var (
		acked_packets_per_second float64
		acked_bytes_per_second   float64
	)
	for i := 0; i < len(rs.ackedQueue); i++ {
		if rs.ackedQueue[i].time >= rs.rttMax {
			acked_packets_per_second++
			acked_bytes_per_second += float64(rs.ackedQueue[i].size)
		}
	}
	sent_bytes_per_second /= float64(rs.rttMax)
	acked_bytes_per_second /= float64(rs.rttMax)
	rs.sentBandwidth = sent_bytes_per_second * (8.0 / 1000.0)
	rs.ackedBandwidth = acked_bytes_per_second * (8.0 / 1000.0)
}
