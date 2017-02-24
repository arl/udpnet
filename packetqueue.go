package udpnet

// packet queue to store information about sent and received packets sorted in
// sequence order + we define ordering using the "sequenceMoreRecent" function,
// this works provided there is a large gap when sequence wrap occurs

type PacketData struct {
	sequence uint    // packet sequence number
	time     float64 // time offset since packet was sent or received (depending on context)
	size     int     // packet size in bytes
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

func bitIndexForSequence(sequence, ack, max_sequence uint) uint {
	// TODO: remove those asserts once done
	if sequence == ack {
		panic("assert(sequence != ack)")
	}
	if sequenceMoreRecent(sequence, ack, max_sequence) {
		panic("assert(!sequence_more_recent(sequence, ack, max_sequence))")
	}
	if sequence > ack {
		if ack >= 33 {
			panic("assert(ack < 33)")
		}
		if max_sequence < sequence {
			panic("assert(max_sequence >= sequence)")
		}
		return ack + (max_sequence - sequence)
	} else {
		if ack < 1 {
			panic("assert(ack >= 1)")
		}
		if sequence > ack-1 {
			panic("passert(sequence <= ack-1)")
		}
		return ack - 1 - sequence
	}
}

func generateAckBits(ack uint, received_queue *PacketQueue, max_sequence uint) uint {
	var ack_bits uint
	for itor := 0; itor < len(*received_queue); itor++ {
		iseq := (*received_queue)[itor].sequence

		if iseq == ack || sequenceMoreRecent(iseq, ack, max_sequence) {
			break
		}
		bit_index := bitIndexForSequence(iseq, ack, max_sequence)
		if bit_index <= 31 {
			ack_bits |= 1 << bit_index
		}
	}
	return ack_bits
}

func processAck(ack, ack_bits uint,
	pending_ack_queue, acked_queue *PacketQueue,
	acks *[]uint, acked_packets *uint,
	rtt *float64, max_sequence uint) {
	if len(*pending_ack_queue) == 0 {
		return
	}

	//PacketQueue::iterator itor = pending_ack_queue.begin();
	for i := 0; i < len(*pending_ack_queue); i++ {
		var acked bool
		itor := &(*pending_ack_queue)[i]

		if itor.sequence == ack {
			acked = true
		} else if !sequenceMoreRecent(itor.sequence, ack, max_sequence) {
			bit_index := bitIndexForSequence(itor.sequence, ack, max_sequence)
			if bit_index <= 31 {
				acked = ((ack_bits >> bit_index) & 1) != 0
			}
		}

		if acked {
			(*rtt) += (itor.time - *rtt) * 0.1

			acked_queue.InsertSorted(*itor, max_sequence)
			*acks = append(*acks, itor.sequence)
			*acked_packets++
			//itor = pending_ack_queue.erase( itor );
			*pending_ack_queue = append((*pending_ack_queue)[:i], (*pending_ack_queue)[i+1:]...)
		}
	}
}
