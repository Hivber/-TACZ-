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
		"Process: goroutines=%d, heap_alloc=%.1f MiB, heap_inuse=%.1f MiB, GC_cycles=%d, GC_pause_total=%.1f ms, GC_CPU=%.2f%%.",
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
		"World: TPS=%.1f, ticks=%d, tick_last/avg/P95/max=%s/%s/%s/%s.",
		stats.TicksPerSecond,
		stats.TickCount,
		stats.LastTick,
		stats.AverageTick,
		stats.P95Tick,
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
}
