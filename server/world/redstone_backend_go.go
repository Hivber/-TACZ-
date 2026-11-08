//go:build !dragonfly_rust || !cgo

package world

func propagateRedstoneGraph(relayers []bool, edges []redstoneEdge, powers []int) []int {
	return propagateRedstoneGraphGo(relayers, edges, powers)
}
