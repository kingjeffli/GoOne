package svrinstmgr

import "testing"

func benchNodes() []uint32 {
	return []uint32{10, 20, 30, 40, 50, 60, 70, 80}
}

// BenchmarkConsistentHash_Get 热路径：已构建 ring，仅查找。
func BenchmarkConsistentHash_Get(b *testing.B) {
	ch := newConsistentHashSorted(benchNodes(), defaultConsistentHashVirtualNodes)
	var k uint64 = 123456789
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ch.get(k)
		k++
	}
}

// BenchmarkConsistentHash_NewAndGetEachTime 旧行为参考：每次查询都 new + reset + get。
func BenchmarkConsistentHash_NewAndGetEachTime(b *testing.B) {
	nodes := benchNodes()
	var k uint64 = 123456789
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newConsistentHash(nodes, defaultConsistentHashVirtualNodes).get(k)
		k++
	}
}

// BenchmarkConsistentHash_BuildRing_Sorted ring 重建成本（注册表刷新时摊销）。
func BenchmarkConsistentHash_BuildRing_Sorted(b *testing.B) {
	nodes := benchNodes()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newConsistentHashSorted(nodes, defaultConsistentHashVirtualNodes)
	}
}

// BenchmarkServerInstanceMgr_ConsistentHashLookup_CachedRing 带缓存 ring 的管理器查询。
func BenchmarkServerInstanceMgr_ConsistentHashLookup_CachedRing(b *testing.B) {
	nodes := benchNodes()
	mgr := &ServerInstanceMgr{
		mapSvrTypeToIns: map[uint32][]uint32{1: nodes},
		consistentHashRing: map[uint32]*consistentHash{
			1: newConsistentHashSorted(nodes, defaultConsistentHashVirtualNodes),
		},
	}
	var k uint64 = 123456789
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.getSvrInsByConsistentHash(1, k)
		k++
	}
}

// BenchmarkServerInstanceMgr_ConsistentHashLookup_NoRingFallback 无预建 ring 时的回退路径。
func BenchmarkServerInstanceMgr_ConsistentHashLookup_NoRingFallback(b *testing.B) {
	nodes := benchNodes()
	mgr := &ServerInstanceMgr{
		mapSvrTypeToIns: map[uint32][]uint32{1: nodes},
	}
	var k uint64 = 123456789
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.getSvrInsByConsistentHash(1, k)
		k++
	}
}
