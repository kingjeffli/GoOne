package svrinstmgr

import "testing"

func TestConsistentHash_EmptyRing(t *testing.T) {
	ch := newConsistentHash(nil, 10)
	if got := ch.get(123); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestConsistentHash_Stability_AddNode(t *testing.T) {
	n1 := []uint32{10, 20, 30}
	n2 := []uint32{10, 20, 30, 40}

	ch1 := newConsistentHash(n1, 50)
	ch2 := newConsistentHash(n2, 50)

	same := 0
	total := 2000
	for i := 0; i < total; i++ {
		k := uint64(i) * 1315423911
		if ch1.get(k) == ch2.get(k) {
			same++
		}
	}
	// 3->4 nodes theoretical unchanged ~ 75%. Use 60% as a loose lower bound.
	if same < int(float64(total)*0.6) {
		t.Fatalf("stability too low: same=%d total=%d", same, total)
	}
}

func TestServerInstanceMgr_ModuloHashAndConsistentHashAreDifferentRules(t *testing.T) {
	mgr := &ServerInstanceMgr{
		routeRules: map[uint32]uint32{1: SvrRouterRule_Hash_UID},
		mapSvrTypeToIns: map[uint32][]uint32{
			1: {10, 20, 30},
		},
	}

	// Hash_UID (modulo): id%3
	if got, _ := mgr.GetSvrInsBySvrType(1, 0, 1, 0); got != 20 {
		t.Fatalf("Hash_UID should be modulo routing: expected 20, got %v", got)
	}

	mgr.routeRules[1] = SvrRouterRule_ConsistentHash_UID
	got, _ := mgr.GetSvrInsBySvrType(1, 0, 1, 0)
	if got != 10 && got != 20 && got != 30 {
		t.Fatalf("ConsistentHash_UID returned unexpected node: %v", got)
	}
}

func TestConsistentHash_SortedCtorMatchesDefaultCtor(t *testing.T) {
	nodes := []uint32{30, 10, 20, 20}
	defaultCtor := newConsistentHash(nodes, defaultConsistentHashVirtualNodes)
	sortedCtor := newConsistentHashSorted([]uint32{10, 20, 30}, defaultConsistentHashVirtualNodes)

	for _, key := range []uint64{1, 7, 42, 99, 123456789} {
		if got, want := sortedCtor.get(key), defaultCtor.get(key); got != want {
			t.Fatalf("key %d: sorted ctor got %v, want %v", key, got, want)
		}
	}
}
