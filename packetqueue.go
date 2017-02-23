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

func sequenceMoreRecent(s1, s2, maxSequence uint) bool {
	return (s1 > s2) && (s1-s2 <= maxSequence/2) || (s2 > s1) && (s2-s1 > maxSequence/2)
}
