package svrinstmgr

import (
	"encoding/binary"
	"hash/crc32"
	"sort"
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
const defaultConsistentHashVirtualNodes = 50

type consistentHash struct {
	vnodes int
	ring   []ringPoint
}

type ringPoint struct {
	hash uint32
	node uint32
}

func newConsistentHash(nodes []uint32, vnodes int) *consistentHash {
	ch := &consistentHash{vnodes: normalizeConsistentHashVirtualNodes(vnodes)}
	ch.reset(nodes)
	return ch
}

func newConsistentHashSorted(nodes []uint32, vnodes int) *consistentHash {
	ch := &consistentHash{vnodes: normalizeConsistentHashVirtualNodes(vnodes)}
	ch.resetSorted(nodes)
	return ch
}

func normalizeConsistentHashVirtualNodes(vnodes int) int {
	if vnodes <= 0 {
		return defaultConsistentHashVirtualNodes
	}
	return vnodes
}

func (h *consistentHash) reset(nodes []uint32) {
	if len(nodes) == 0 {
		h.ring = h.ring[:0]
		return
	}
	ns := append([]uint32(nil), nodes...)
	sort.Slice(ns, func(i, j int) bool { return ns[i] < ns[j] })
	ns = Uint32SliceDeduplicateSorted(ns)
	h.resetSorted(ns)
}

func (h *consistentHash) resetSorted(nodes []uint32) {
	h.ring = h.ring[:0]
	if len(nodes) == 0 {
		return
	}

	h.ring = make([]ringPoint, 0, len(nodes)*h.vnodes)
	var buf [8]byte
	for _, n := range nodes {
		binary.LittleEndian.PutUint32(buf[:4], n)
		for i := 0; i < h.vnodes; i++ {
			binary.LittleEndian.PutUint32(buf[4:], uint32(i))
			hash := crc32.ChecksumIEEE(buf[:])
			h.ring = append(h.ring, ringPoint{hash: hash, node: n})
		}
	}
	sort.Slice(h.ring, func(i, j int) bool { return h.ring[i].hash < h.ring[j].hash })
}

func (h *consistentHash) get(key uint64) uint32 {
	if h == nil || len(h.ring) == 0 {
		return 0
	}
	// Hash key (little endian) to match typical stable hashing.
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], key)
	hash := crc32.ChecksumIEEE(buf[:])

	idx := h.search(hash)
	if idx == len(h.ring) {
		idx = 0
	}
	return h.ring[idx].node
}

func (h *consistentHash) search(hash uint32) int {
	lo, hi := 0, len(h.ring)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if h.ring[mid].hash < hash {
			lo = mid + 1
			continue
		}
		hi = mid
	}
	return lo
}

var _ = newConsistentHash
