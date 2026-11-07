package cmd

import (
	"runtime"
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

func init() {
	Register(New("perf", "Shows live server performance diagnostics.", nil, perfCommand{}))
}

type perfCommand struct{}

func (perfCommand) Allow(src Source) bool { return commandOperator(src) }

func (perfCommand) Run(_ Source, out *Output, tx *world.Tx) {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	const mib = 1024 * 1024
	out.Printf(
		"Process: CPUs=%d, GOMAXPROCS=%d, goroutines=%d, heap_alloc=%.1f MiB, heap_inuse=%.1f MiB, GC_cycles=%d, GC_pause_total=%.1f ms, GC_CPU=%.2f%%.",
		runtime.NumCPU(),
		runtime.GOMAXPROCS(0),
		runtime.NumGoroutine(),
		float64(memory.HeapAlloc)/mib,
		float64(memory.HeapInuse)/mib,
		memory.NumGC,
		float64(memory.PauseTotalNs)/float64(time.Millisecond),
		memory.GCCPUFraction*100,
	)

	if tx == nil {
		out.Printf("World metrics unavailable: run /perf from a player in the target world.")
		return
	}

	stats := tx.Performance()
	out.Printf(
		"World: TPS=%.1f, ticks=%d, tick_last/EMA/P95/recent_max/lifetime_max=%s/%s/%s/%s/%s.",
		stats.TicksPerSecond,
		stats.TickCount,
		stats.LastTick,
		stats.AverageTick,
		stats.P95Tick,
		stats.RecentMaxTick,
		stats.MaxTick,
	)
	out.Printf(
		"Workload: loaded_chunks=%d, loading_chunks=%d, chunk_queue=%d/%d, chunk_workers=%d, entities=%d, active_loaders=%d, simulation_distance=%d, random_tick_speed=%d.",
		stats.LoadedChunks,
		stats.LoadingChunks,
		stats.QueuedChunkLoads,
		stats.ChunkLoadQueueCapacity,
		stats.ChunkLoadWorkers,
		stats.LoadedEntities,
		stats.ActiveLoaders,
		stats.SimulationDistance,
		stats.RandomTickSpeed,
	)
	out.Printf(
		"Tick stage EMA/P95: entities=%s/%s, scheduled=%s/%s, random_blocks=%s/%s.",
		stats.Stages[world.TickStageEntities].Average, stats.Stages[world.TickStageEntities].P95,
		stats.Stages[world.TickStageScheduled].Average, stats.Stages[world.TickStageScheduled].P95,
		stats.Stages[world.TickStageRandomBlocks].Average, stats.Stages[world.TickStageRandomBlocks].P95,
	)
	out.Printf(
		"Tick stage EMA/P95: neighbours=%s/%s, redstone=%s/%s, lightning=%s/%s, other=%s/%s.",
		stats.Stages[world.TickStageNeighbours].Average, stats.Stages[world.TickStageNeighbours].P95,
		stats.Stages[world.TickStageRedstone].Average, stats.Stages[world.TickStageRedstone].P95,
		stats.Stages[world.TickStageLightning].Average, stats.Stages[world.TickStageLightning].P95,
		stats.Stages[world.TickStageOther].Average, stats.Stages[world.TickStageOther].P95,
	)
	out.Printf(
		"Queue/updates: world_queue_wait last/EMA/P95=%s/%s/%s, neighbour_processed=%d, neighbour_pending=%d.",
		stats.QueueWait.Last, stats.QueueWait.Average, stats.QueueWait.P95,
		stats.NeighbourUpdatesProcessed, stats.PendingNeighbourUpdates,
	)
	out.Printf(
		"Redstone: dirty_positions=%d, graph_nodes=%d, graph_edges=%d, context_scan_positions=%d.",
		stats.RedstoneDirtyPositions, stats.RedstoneGraphNodes, stats.RedstoneGraphEdges,
		stats.RedstoneContextScanPositions,
	)
}
