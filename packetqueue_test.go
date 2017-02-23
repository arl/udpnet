package udpnet

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(270679)
}

func verifySorted(t *testing.T, pq *PacketQueue, maxSequence uint) {
	prev := len(*pq)

	for iter := 0; iter != len(*pq); iter++ {
		assert.True(t, (*pq)[iter].sequence <= maxSequence)
		if prev != len(*pq) {
			assert.True(t, sequenceMoreRecent((*pq)[iter].sequence, (*pq)[prev].sequence, maxSequence))
			prev = iter
		}
	}
}

func TestPacketQueue(t *testing.T) {
	const MaximumSequence = 255

	var packetQueue PacketQueue

	t.Logf("check insert back\n")
	for i := 0; i < 100; i++ {
		var data PacketData
		data.sequence = uint(i)
		packetQueue.InsertSorted(data, MaximumSequence)
		verifySorted(t, &packetQueue, MaximumSequence)
	}

	t.Logf("check insert front\n")
	packetQueue = PacketQueue{}
	for i := 100; i < 0; i++ {
		var data PacketData
		data.sequence = uint(i)
		packetQueue.InsertSorted(data, MaximumSequence)
		verifySorted(t, &packetQueue, MaximumSequence)
	}

	t.Logf("check insert random\n")
	packetQueue = PacketQueue{}
	for i := 100; i < 0; i++ {
		var data PacketData
		data.sequence = uint(rand.Int() & 0xFF)
		packetQueue.InsertSorted(data, MaximumSequence)
		verifySorted(t, &packetQueue, MaximumSequence)
	}

	t.Logf("check insert wrap around\n")
	packetQueue = PacketQueue{}
	for i := 200; i <= 255; i++ {
		var data PacketData
		data.sequence = uint(i)
		packetQueue.InsertSorted(data, MaximumSequence)
		verifySorted(t, &packetQueue, MaximumSequence)
	}
	for i := 0; i <= 50; i++ {
		var data PacketData
		data.sequence = uint(i)
		packetQueue.InsertSorted(data, MaximumSequence)
		verifySorted(t, &packetQueue, MaximumSequence)
	}
}
