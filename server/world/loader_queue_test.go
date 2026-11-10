package world

import "testing"

func TestLoaderQueueDeduplicatesInConstantTime(t *testing.T) {
	l := &Loader{
		r:       4,
		w:       &World{},
		loaded:  make(map[ChunkPos]*Column),
		pending: make(map[ChunkPos]struct{}),
		queued:  make(map[ChunkPos]struct{}),
	}
	pos := ChunkPos{1, 1}
	l.queueLoad(pos)
	l.queueLoad(pos)
	if len(l.loadQueue) != 1 {
		t.Fatalf("queued %d copies of the same chunk, want 1", len(l.loadQueue))
	}
	if _, ok := l.queued[pos]; !ok {
		t.Fatal("queued position missing from deduplication index")
	}
	l.queueLoad(ChunkPos{8, 8})
	if len(l.loadQueue) != 1 {
		t.Fatal("out-of-radius chunk was queued")
	}
}
