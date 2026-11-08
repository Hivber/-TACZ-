package world

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
)

const redstoneDirtyIndexThreshold = 64

type redstoneDirtyIndexNode struct {
	pos   cube.Pos
	dirty redstoneDirty
	axis  uint8
	left  int
	right int
}

type redstoneDirtyIndex struct {
	nodes []redstoneDirtyIndexNode
	root  int
}

type redstoneDirtyEntry struct {
	pos   cube.Pos
	dirty redstoneDirty
}

func newRedstoneDirtyIndex(dirty map[cube.Pos]redstoneDirty) *redstoneDirtyIndex {
	if len(dirty) < redstoneDirtyIndexThreshold {
		return nil
	}
	entries := make([]redstoneDirtyEntry, 0, len(dirty))
	for pos, context := range dirty {
		entries = append(entries, redstoneDirtyEntry{pos: pos, dirty: context})
	}
	index := &redstoneDirtyIndex{nodes: make([]redstoneDirtyIndexNode, 0, len(entries)), root: -1}
	index.root = index.build(entries, 0)
	return index
}

func (index *redstoneDirtyIndex) build(entries []redstoneDirtyEntry, axis uint8) int {
	if len(entries) == 0 {
		return -1
	}
	slices.SortFunc(entries, func(a, b redstoneDirtyEntry) int {
		if a.pos[axis] != b.pos[axis] {
			return a.pos[axis] - b.pos[axis]
		}
		return compareBlockPos(a.pos, b.pos)
	})
	middle := len(entries) / 2
	entry := entries[middle]
	nodeIndex := len(index.nodes)
	index.nodes = append(index.nodes, redstoneDirtyIndexNode{
		pos: entry.pos, dirty: entry.dirty, axis: axis, left: -1, right: -1,
	})
	nextAxis := (axis + 1) % 3
	left := index.build(entries[:middle], nextAxis)
	right := index.build(entries[middle+1:], nextAxis)
	index.nodes[nodeIndex].left = left
	index.nodes[nodeIndex].right = right
	return nodeIndex
}

// nearest returns the same Manhattan-nearest context and lexicographic tie-break as redstoneDirtyContext.
func (index *redstoneDirtyIndex) nearest(pos cube.Pos) (redstoneDirty, int) {
	if index == nil || index.root < 0 {
		return redstoneDirty{changed: pos, hasChanged: true, source: pos, hasSource: true, cause: RedstoneUpdateCauseCompilerRebuild}, 0
	}
	bestDistance := int(^uint(0) >> 1)
	var bestPos cube.Pos
	var best redstoneDirty
	visited := 0
	var search func(int)
	search = func(nodeIndex int) {
		if nodeIndex < 0 {
			return
		}
		node := index.nodes[nodeIndex]
		visited++
		distance := redstoneManhattanDistance(pos, node.pos)
		if distance < bestDistance || (distance == bestDistance && compareBlockPos(node.pos, bestPos) < 0) {
			bestDistance, bestPos, best = distance, node.pos, node.dirty
		}
		delta := pos[node.axis] - node.pos[node.axis]
		near, far := node.left, node.right
		if delta > 0 {
			near, far = node.right, node.left
		}
		search(near)
		if abs(delta) <= bestDistance {
			search(far)
		}
	}
	search(index.root)
	return best, visited
}
