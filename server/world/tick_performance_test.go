package world

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestSimulationChunkPositions(t *testing.T) {
	loader := &Loader{pos: ChunkPos{-3, 7}}
	got := simulationChunkPositions([]*Loader{loader}, 1)
	want := map[ChunkPos]struct{}{
		{-3, 7}: {},
		{-4, 7}: {},
		{-2, 7}: {},
		{-3, 6}: {},
		{-3, 8}: {},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d chunks, want %d", len(got), len(want))
	}
	for pos := range want {
		if _, ok := got[pos]; !ok {
			t.Errorf("simulation area is missing chunk %v", pos)
		}
	}
}

func TestSimulationChunkPositionsDeduplicateOverlappingLoaders(t *testing.T) {
	first := &Loader{pos: ChunkPos{0, 0}}
	second := &Loader{pos: ChunkPos{1, 0}}
	got := simulationChunkPositions([]*Loader{first, second}, 1)
	if len(got) != 8 {
		t.Fatalf("got %d unique simulation chunks, want 8", len(got))
	}
}

func TestSimulationChunkPositionsZeroRadius(t *testing.T) {
	loader := &Loader{pos: ChunkPos{0, 0}}
	if got := simulationChunkPositions([]*Loader{loader}, 0); len(got) != 0 {
		t.Fatalf("zero simulation radius returned %d chunks, want 0", len(got))
	}
}

// TestPerformNeighbourUpdatesDoesNotLoadUnloadedChunks verifies that queued
// neighbour notifications never synchronously load or generate their target
// chunks while the world is ticking.
func TestPerformNeighbourUpdatesDoesNotLoadUnloadedChunks(t *testing.T) {
	w := Config{Synchronous: true}.New()
	defer w.Close()

	pos := cube.Pos{1600, 64, 1600}
	chunkPos := chunkPosFromBlockPos(pos)
	var loaded bool
	<-w.exec(func(tx *Tx) {
		w.neighbourUpdates = append(w.neighbourUpdates, neighbourUpdate{
			pos: pos, neighbour: pos,
		})
		ticker{}.performNeighbourUpdates(tx)
		_, loaded = w.chunks[chunkPos]
	})
	if loaded {
		t.Fatalf("processing a neighbour update loaded chunk %v", chunkPos)
	}
}
