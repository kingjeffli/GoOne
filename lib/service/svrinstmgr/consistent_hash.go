package svrinstmgr

import (
	"encoding/binary"
	"hash/crc32"
	"sort"
	"strconv"
)

// consistentHash implements a tiny, dependency-free consistent hashing ring.
//
// Contract:
//   - nodes are uint32 (busID / ip-int)
//   - key is uint64 (uid/zone/routerId)
//   - returns 0 when ring is empty
//
// Notes:
//   - we use virtual nodes for better distribution
//   - we keep ring sorted by hash so lookup is O(logN)
//   - this is deterministic across runs
//
// This is intentionally self-contained to match project style (no extra deps).
// If needed later, we can extract it to lib/util/lru.
type consistentHash struct {
	vnodes int
	ring   []ringPoint
}

type ringPoint struct {
	hash uint32
	node uint32
}

func newConsistentHash(nodes []uint32, vnodes int) *consistentHash {
	if vnodes <= 0 {
		vnodes = 50
	}
	ch := &consistentHash{vnodes: vnodes}
	ch.reset(nodes)
	return ch
}

func (h *consistentHash) reset(nodes []uint32) {
	h.ring = h.ring[:0]
	if len(nodes) == 0 {
		return
	}
	// nodes should already be deduplicated and sorted by caller, but don't assume.
	ns := append([]uint32(nil), nodes...)
	sort.Slice(ns, func(i, j int) bool { return ns[i] < ns[j] })
	ns = Uint32SliceDeduplicateSorted(ns)

	h.ring = make([]ringPoint, 0, len(ns)*h.vnodes)
	for _, n := range ns {
		for i := 0; i < h.vnodes; i++ {
			h.hringAdd(n, i)
		}
	}
	sort.Slice(h.ring, func(i, j int) bool { return h.ring[i].hash < h.ring[j].hash })
}

func (h *consistentHash) hringAdd(node uint32, vnodeIdx int) {
	// Hash inputs as text keeps it stable and simple.
	// Use a delimiter to avoid ambiguity.
	s := strconv.FormatUint(uint64(node), 10) + "#" + strconv.Itoa(vnodeIdx)
	hash := crc32.ChecksumIEEE([]byte(s))
	h.ring = append(h.ring, ringPoint{hash: hash, node: node})
}

func (h *consistentHash) get(key uint64) uint32 {
	if h == nil || len(h.ring) == 0 {
		return 0
	}
	// Hash key (little endian) to match typical stable hashing.
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], key)
	hash := crc32.ChecksumIEEE(buf[:])

	idx := sort.Search(len(h.ring), func(i int) bool { return h.ring[i].hash >= hash })
	if idx == len(h.ring) {
		idx = 0
	}
	return h.ring[idx].node
}

var _ = newConsistentHash
