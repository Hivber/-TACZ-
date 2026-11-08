package session

import (
	"bytes"
	"testing"

	"github.com/df-mc/dragonfly/server/world/chunk"
)

func TestCachedChunkPartReusesRevisionAndBoundsMemory(t *testing.T) {
	c := &chunk.Chunk{}
	s := &Session{
		conf:                  Config{PerformanceDiagnostics: true},
		encodedChunkParts:     make(map[encodedChunkPartKey][]byte, encodedChunkPartCacheCapacity),
		encodedChunkPartOrder: make([]encodedChunkPartKey, 0, encodedChunkPartCacheCapacity),
	}
	calls := 0
	encode := func() []byte {
		calls++
		return []byte{byte(calls)}
	}

	first := s.cachedChunkPart(c, 0, 1, encode)
	second := s.cachedChunkPart(c, 0, 1, encode)
	if calls != 1 || !bytes.Equal(first, second) {
		t.Fatalf("same revision encoded %d times; first=%v second=%v", calls, first, second)
	}
	changed := s.cachedChunkPart(c, 0, 2, encode)
	if calls != 2 || bytes.Equal(first, changed) {
		t.Fatalf("new revision was not re-encoded: calls=%d first=%v changed=%v", calls, first, changed)
	}
	if s.encodedChunkPartCacheHits.Load() != 1 || s.encodedChunkPartCacheMisses.Load() != 2 {
		t.Fatalf("cache counters = hits %d, misses %d; want 1 and 2", s.encodedChunkPartCacheHits.Load(), s.encodedChunkPartCacheMisses.Load())
	}

	for i := 1; i <= encodedChunkPartCacheCapacity; i++ {
		s.cachedChunkPart(c, int16(i), 1, encode)
	}
	if got := len(s.encodedChunkParts); got != encodedChunkPartCacheCapacity {
		t.Fatalf("cache size = %d, want capacity %d", got, encodedChunkPartCacheCapacity)
	}
	if _, ok := s.encodedChunkParts[encodedChunkPartKey{chunk: c, index: 0, revision: 1}]; ok {
		t.Fatal("oldest cache entry was not evicted")
	}
}
