package session

import (
	"fmt"
	"time"
)

// PerformanceString returns a compact snapshot of per-session network and client diagnostics.
// Packet rates are measured since the previous call (or since session creation on the first call).
func (s *Session) PerformanceString() string {
	now := time.Now()
	s.performanceSampleMu.Lock()
	elapsed := now.Sub(s.performanceSampleAt).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}
	packets := s.outPackets.Load()
	moves := s.entityMovementPackets.Load()
	velocities := s.entityVelocityPackets.Load()
	spawns := s.entitySpawnPackets.Load()
	removes := s.entityRemovePackets.Load()
	chunks := s.chunkPackets.Load()
	blocks := s.blockUpdatePackets.Load()
	cacheResponses := s.cacheResponsePackets.Load()

	rate := func(current, previous uint64) float64 {
		return float64(current-previous) / elapsed
	}
	result := fmt.Sprintf(
		"Session network/s: packets=%.1f, entity_move=%.1f, entity_velocity=%.1f, entity_spawn=%.1f, entity_remove=%.1f, chunk_packets=%.1f, block_updates=%.1f, cache_responses=%.1f, cache_hits=%d, cache_misses=%d, subchunk_requests=%d, encode_cache_hits=%d, encode_cache_misses=%d.",
		rate(packets, s.performanceLastPackets),
		rate(moves, s.performanceLastMoves),
		rate(velocities, s.performanceLastVel),
		rate(spawns, s.performanceLastSpawns),
		rate(removes, s.performanceLastRemoves),
		rate(chunks, s.performanceLastChunks),
		rate(blocks, s.performanceLastBlocks),
		rate(cacheResponses, s.performanceLastCache),
		s.cacheHits.Load(), s.cacheMisses.Load(), s.subChunkRequests.Load(),
		s.encodedChunkPartCacheHits.Load(), s.encodedChunkPartCacheMisses.Load(),
	)
	s.performanceSampleAt = now
	s.performanceLastPackets = packets
	s.performanceLastMoves = moves
	s.performanceLastVel = velocities
	s.performanceLastSpawns = spawns
	s.performanceLastRemoves = removes
	s.performanceLastChunks = chunks
	s.performanceLastBlocks = blocks
	s.performanceLastCache = cacheResponses
	s.performanceSampleMu.Unlock()

	stats := s.chunkLoaderStats()
	result += fmt.Sprintf(
		" Session chunks: radius=%d, loaded=%d, pending=%d, queued=%d, cache_blobs=%d, cache_transactions=%d, packet_queue=%d/256.",
		stats.Radius, stats.Loaded, stats.Pending, stats.Queued,
		s.pendingBlobCount(), s.pendingChunkTransactionCount(), len(s.packets),
	)
	if d := s.clientDiagnostics.Load(); d != nil {
		result += fmt.Sprintf(
			" Client: FPS=%.1f, server_sim=%.2fms, client_sim=%.2fms, render=%.2fms, begin_frame=%.2fms, input=%.2fms.",
			d.AverageFramesPerSecond, d.AverageServerSimTickTime, d.AverageClientSimTickTime,
			d.AverageRenderTime, d.AverageBeginFrameTime, d.AverageInputTime,
		)
	} else {
		result += " Client diagnostics: waiting for a client report."
	}
	return result
}

func (s *Session) chunkLoaderStats() (stats struct{ Radius, Loaded, Pending, Queued int }) {
	if s.chunkLoader == nil {
		return
	}
	v := s.chunkLoader.Stats()
	stats.Radius, stats.Loaded, stats.Pending, stats.Queued = v.Radius, v.Loaded, v.Pending, v.Queued
	return
}

func (s *Session) pendingBlobCount() int {
	s.blobMu.Lock()
	defer s.blobMu.Unlock()
	return len(s.blobs)
}

func (s *Session) pendingChunkTransactionCount() int {
	s.blobMu.Lock()
	defer s.blobMu.Unlock()
	return len(s.openChunkTransactions)
}
