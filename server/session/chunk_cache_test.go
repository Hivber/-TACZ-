package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world/chunk"
)

func TestEncodedChunkPartCacheSharesEntries(t *testing.T) {
	cache := newEncodedChunkPartCache(4)
	key := encodedChunkPartKey{chunk: new(chunk.Chunk), index: 2, revision: 7}
	encodes := 0
	encode := func() []byte {
		encodes++
		return []byte("encoded")
	}

	first, hit := cache.get(key, encode)
	if hit {
		t.Fatal("first lookup unexpectedly hit")
	}
	second, hit := cache.get(key, encode)
	if !hit {
		t.Fatal("second lookup did not hit")
	}
	if encodes != 1 {
		t.Fatalf("encode called %d times, want 1", encodes)
	}
	if string(first) != string(second) {
		t.Fatalf("cached payload differs: %q != %q", first, second)
	}
}

func TestEncodedChunkPartCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := newEncodedChunkPartCache(2)
	chunks := []*chunk.Chunk{new(chunk.Chunk), new(chunk.Chunk), new(chunk.Chunk)}
	keys := []encodedChunkPartKey{
		{chunk: chunks[0], index: 0, revision: 1},
		{chunk: chunks[1], index: 0, revision: 1},
		{chunk: chunks[2], index: 0, revision: 1},
	}
	encode := func() []byte { return []byte("payload") }

	cache.get(keys[0], encode)
	cache.get(keys[1], encode)
	cache.get(keys[0], encode) // Make key 0 the most recently used.
	cache.get(keys[2], encode)

	if _, hit := cache.get(keys[0], encode); !hit {
		t.Fatal("most recently used entry was evicted")
	}
	if _, hit := cache.get(keys[1], encode); hit {
		t.Fatal("least recently used entry was not evicted")
	}
	if cache.order.Len() != 2 || len(cache.entries) != 2 {
		t.Fatalf("cache size = %d entries / %d list nodes, want 2", len(cache.entries), cache.order.Len())
	}
}

func TestEncodedChunkPartCacheSeparatesRevisions(t *testing.T) {
	cache := newEncodedChunkPartCache(4)
	base := encodedChunkPartKey{chunk: new(chunk.Chunk), index: 1, revision: 1}
	updated := base
	updated.revision++

	cache.get(base, func() []byte { return []byte("old") })
	got, hit := cache.get(updated, func() []byte { return []byte("new") })
	if hit || string(got) != "new" {
		t.Fatalf("new revision returned hit=%v payload=%q", hit, got)
	}
}


func TestEncodedChunkPartCacheEnforcesByteBudget(t *testing.T) {
	cache := newEncodedChunkPartCacheWithLimits(8, 10)
	chunks := []*chunk.Chunk{new(chunk.Chunk), new(chunk.Chunk), new(chunk.Chunk)}
	keys := []encodedChunkPartKey{
		{chunk: chunks[0], index: 0, revision: 1},
		{chunk: chunks[1], index: 0, revision: 1},
		{chunk: chunks[2], index: 0, revision: 1},
	}
	cache.get(keys[0], func() []byte { return []byte("123456") })
	cache.get(keys[1], func() []byte { return []byte("abcd") })
	if cache.bytes != 10 {
		t.Fatalf("cached bytes = %d, want 10", cache.bytes)
	}
	cache.get(keys[2], func() []byte { return []byte("xyz") })
	if cache.bytes != 7 || cache.order.Len() != 2 {
		t.Fatalf("after byte-budget eviction: bytes=%d entries=%d, want 7 bytes and 2 entries", cache.bytes, cache.order.Len())
	}
	if _, hit := cache.get(keys[0], func() []byte { return []byte("123456") }); hit {
		t.Fatal("oldest entry should have been evicted to satisfy byte budget")
	}
	if _, hit := cache.get(keys[2], func() []byte { return []byte("xyz") }); !hit {
		t.Fatal("newest entry should remain cached")
	}
}

func TestEncodedChunkPartCacheDoesNotRetainOversizedPayload(t *testing.T) {
	cache := newEncodedChunkPartCacheWithLimits(8, 4)
	key := encodedChunkPartKey{chunk: new(chunk.Chunk), index: 0, revision: 1}
	payload := []byte("oversized")
	got, hit := cache.get(key, func() []byte { return payload })
	if hit || string(got) != string(payload) {
		t.Fatalf("oversized payload lookup returned hit=%v data=%q", hit, got)
	}
	if cache.order.Len() != 0 || cache.bytes != 0 {
		t.Fatalf("oversized payload was retained: entries=%d bytes=%d", cache.order.Len(), cache.bytes)
	}
}
