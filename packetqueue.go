package udpnet

import "time"

// packet queue to store information about sent and received packets sorted in
// sequence order + we define ordering using the "sequenceMoreRecent" function,
// this works provided there is a large gap when sequence wrap occurs

type PacketData struct {
	sequence uint          // packet sequence number
	time     time.Duration // time offset since packet was sent or received (depending on context)
	size     int           // packet size in bytes
}

// TODO: check if its better to have a slice of pointers to PacketData
type PacketQueue []PacketData

func (pq *PacketQueue) Exists(sequence uint) bool {
	for i := 0; i < len(*pq); i++ {
		if (*pq)[i].sequence == sequence {
			return true
		}
	}
	return false
}

func (pq *PacketQueue) InsertSorted(p PacketData, maxSequence uint) {
	if len(*pq) == 0 {
		*pq = append(*pq, p)
	} else {
		if !sequenceMoreRecent(p.sequence, (*pq)[0].sequence, maxSequence) {
			*pq = append(PacketQueue{p}, *pq...)
		} else if sequenceMoreRecent(p.sequence, (*pq)[len(*pq)-1].sequence, maxSequence) {
			*pq = append(*pq, p)
		} else {
			for i := 0; i < len(*pq); i++ {
				if sequenceMoreRecent((*pq)[i].sequence, p.sequence, maxSequence) {
					(*pq) = append((*pq)[:i], append(PacketQueue{p}, (*pq)[i:]...)...)
					i++
				}
			}
		}
	}
}

//
// utility functions
//

func sequenceMoreRecent(s1, s2, maxSequence uint) bool {
	return (s1 > s2) && (s1-s2 <= maxSequence/2) || (s2 > s1) && (s2-s1 > maxSequence/2)
}

func bitIndexForSequence(sequence, ack, maxSequence uint) uint {
	// TODO: remove those asserts once done
	if sequence == ack {
		panic("assert(sequence != ack)")
	}
	if sequenceMoreRecent(sequence, ack, maxSequence) {
		panic("assert(!sequenceMoreRecent(sequence, ack, maxSequence))")
	}
	if sequence > ack {
		if ack >= 33 {
			panic("assert(ack < 33)")
		}
		if maxSequence < sequence {
			panic("assert(maxSequence >= sequence)")
		}
		return ack + (maxSequence - sequence)
	}
	if ack < 1 {
		panic("assert(ack >= 1)")
	}
	if sequence > ack-1 {
		panic("assert(sequence <= ack-1)")
	}
	return ack - 1 - sequence
}

func generateAckBits(ack uint, receivedQueue *PacketQueue, maxSequence uint) uint {
	var ackBits uint
	for itor := 0; itor < len(*receivedQueue); itor++ {
		iseq := (*receivedQueue)[itor].sequence

		if iseq == ack || sequenceMoreRecent(iseq, ack, maxSequence) {
			break
		}
		bitIndex := bitIndexForSequence(iseq, ack, maxSequence)
		if bitIndex <= 31 {
			ackBits |= 1 << bitIndex
		}
	}
	return ackBits
}

func processAck(ack, ackBits uint,
	pendingAckQueue, ackedQueue *PacketQueue,
	acks *[]uint, ackedPackets *uint,
	rtt *time.Duration, maxSequence uint) {
	if len(*pendingAckQueue) == 0 {
		return
	}

	i := 0
	for i < len(*pendingAckQueue) {
		var acked bool
		itor := &(*pendingAckQueue)[i]

		if itor.sequence == ack {
			acked = true
		} else if !sequenceMoreRecent(itor.sequence, ack, maxSequence) {
			bitIndex := bitIndexForSequence(itor.sequence, ack, maxSequence)
			if bitIndex <= 31 {
				acked = ((ackBits >> bitIndex) & 1) != 0
			}
		}

		if acked {
			(*rtt) += (itor.time - *rtt) / 10

			ackedQueue.InsertSorted(*itor, maxSequence)
			*acks = append(*acks, itor.sequence)
			*ackedPackets++
			//itor = pending_ack_queue.erase( itor );
			*pendingAckQueue = append((*pendingAckQueue)[:i], (*pendingAckQueue)[i+1:]...)
		} else {
			i++
		}
	}
}
