//go:build dragonfly_rust && cgo

package world

/*
#cgo CFLAGS: -I${SRCDIR}/../../native/redstone-rs/include
#cgo linux LDFLAGS: -L${SRCDIR}/../../native/redstone-rs/target/release -ldragonfly_redstone -ldl -lm
#cgo android LDFLAGS: -L${SRCDIR}/../../native/redstone-rs/target/aarch64-linux-android/release -ldragonfly_redstone -ldl -lm
#include "dragonfly_redstone.h"
*/
import "C"

import "runtime"

// propagateRedstoneGraph uses one batched FFI call. All world access and state mutation remain in Go.
func propagateRedstoneGraph(relayers []bool, edges []redstoneEdge, powers []int) []int {
	// FFI setup and slice conversion dominate small networks. Keep those on the Go path.
	if len(relayers) < 256 || len(edges) < 256 || len(relayers) != len(powers) {
		return propagateRedstoneGraphGo(relayers, edges, powers)
	}
	if uint64(len(relayers)) > uint64(^uint32(0)) {
		return propagateRedstoneGraphGo(relayers, edges, powers)
	}
	flags := make([]C.uint8_t, len(relayers))
	for i, relayer := range relayers {
		if relayer {
			flags[i] = 1
		}
	}
	nativeEdges := make([]C.DragonflyRedstoneEdge, len(edges))
	const maxInt32 = int(^uint32(0) >> 1)
	const minInt32 = -maxInt32 - 1
	for i, edge := range edges {
		if edge.from < 0 || edge.from >= len(relayers) || edge.to < 0 || edge.to >= len(relayers) ||
			edge.from > maxInt32 || edge.to > maxInt32 || edge.weight < minInt32 || edge.weight > maxInt32 {
			return propagateRedstoneGraphGo(relayers, edges, powers)
		}
		nativeEdges[i] = C.DragonflyRedstoneEdge{
			from: C.uint32_t(edge.from),
			to: C.uint32_t(edge.to),
			weight: C.int32_t(edge.weight),
		}
	}
	nativePowers := make([]C.int32_t, len(powers))
	for i, power := range powers {
		nativePowers[i] = C.int32_t(ClampRedstonePower(power))
	}
	var edgePointer *C.DragonflyRedstoneEdge
	if len(nativeEdges) != 0 {
		edgePointer = &nativeEdges[0]
	}
	status := C.dragonfly_redstone_propagate(
		&flags[0], C.size_t(len(flags)),
		edgePointer, C.size_t(len(nativeEdges)),
		&nativePowers[0],
	)
	runtime.KeepAlive(flags)
	runtime.KeepAlive(nativeEdges)
	runtime.KeepAlive(nativePowers)
	if status != 0 {
		return propagateRedstoneGraphGo(relayers, edges, powers)
	}
	for i, power := range nativePowers {
		powers[i] = int(power)
	}
	return powers
}

