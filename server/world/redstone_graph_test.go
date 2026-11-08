package world

import "testing"

func TestPropagateRedstoneGraph(t *testing.T) {
	// 0 and 1 are relayers. Node 2 is a sink, while node 3 is a
	// non-relayer endpoint and must not propagate power any further.
	relayers := []bool{true, true, false, false, true}
	edges := []redstoneEdge{
		{from: 0, to: 1, weight: 1},
		{from: 1, to: 2, weight: 1},
		{from: 1, to: 4, weight: 2},
		{from: 2, to: 3, weight: 0},
		{from: 4, to: 0, weight: 1},
	}
	got := propagateRedstoneGraph(relayers, edges, []int{15, 0, 0, 0, 0})
	want := []int{15, 14, 13, 0, 12}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("power[%d] = %d, want %d (all=%v)", i, got[i], want[i], got)
		}
	}
}


func TestPropagateRedstoneGraphMatchesGoReference(t *testing.T) {
	for _, nodeCount := range []int{256, 512, 1024} {
		relayers := make([]bool, nodeCount)
		powers := make([]int, nodeCount)
		edges := make([]redstoneEdge, 0, nodeCount*2)
		for i := range nodeCount {
			relayers[i] = i%3 != 0
			if i%127 == 0 {
				powers[i] = 15
			}
			if i+1 < nodeCount {
				edges = append(edges, redstoneEdge{from: i, to: i + 1, weight: 1})
			}
			if i+2 < nodeCount {
				edges = append(edges, redstoneEdge{from: i, to: i + 2, weight: 2})
			}
		}
		gotInput := append([]int(nil), powers...)
		wantInput := append([]int(nil), powers...)
		got := propagateRedstoneGraph(relayers, edges, gotInput)
		want := propagateRedstoneGraphGo(relayers, edges, wantInput)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("node count %d, power[%d] = %d, want %d", nodeCount, i, got[i], want[i])
			}
		}
	}
}

func BenchmarkPropagateRedstoneGraph(b *testing.B) {
	const nodeCount = 4096
	relayers := make([]bool, nodeCount)
	powers := make([]int, nodeCount)
	edges := make([]redstoneEdge, 0, nodeCount*2)
	for i := range relayers {
		relayers[i] = true
		if i%512 == 0 {
			powers[i] = 15
		}
		if i+1 < nodeCount {
			edges = append(edges, redstoneEdge{from: i, to: i + 1, weight: 1})
		}
		if i+2 < nodeCount {
			edges = append(edges, redstoneEdge{from: i, to: i + 2, weight: 1})
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := propagateRedstoneGraph(relayers, edges, append([]int(nil), powers...))
		if len(result) != nodeCount {
			b.Fatal("unexpected result size")
		}
	}
}
