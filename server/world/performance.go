package world

import (
	"slices"
	"time"
)

const performanceTickSamples = 100

// TickStage identifies a measured subsystem of the world tick.
type TickStage uint8

const (
	TickStageLightning TickStage = iota
	TickStageEntities
	TickStageScheduled
	TickStageRandomBlocks
	TickStageNeighbours
	TickStageRedstone
	TickStageOther
	tickStageCount
)

var tickStageNames = [tickStageCount]string{
	"lightning", "entities", "scheduled", "random_blocks", "neighbours", "redstone", "other",
}

// PerformanceSnapshot contains a point-in-time view of World tick and workload
// metrics. A snapshot must be obtained from an active world transaction.
type PerformanceSnapshot struct {
	TickCount                    uint64
	TicksPerSecond               float64
	LastTick                     time.Duration
	AverageTick                  time.Duration
	P95Tick                      time.Duration
	RecentMaxTick                time.Duration
	MaxTick                      time.Duration
	LoadedChunks                 int
	LoadingChunks                int
	QueuedChunkLoads             int
	ChunkLoadQueueCapacity       int
	ChunkLoadWorkers             int
	LoadedEntities               int
	ActiveLoaders                int
	SimulationDistance           int
	RandomTickSpeed              int
	Stages                       [tickStageCount]TickStageSnapshot
	QueueWait                    TickStageSnapshot
	NeighbourUpdatesProcessed    int
	NeighbourUpdatesSkippedUnloaded int
	NeighbourBlockCallbacks      int
	NeighbourLiquidCallbacks     int
	PendingNeighbourUpdates      int
	RedstoneDirtyPositions       int
	RedstoneGraphNodes           int
	RedstoneGraphEdges           int
	RedstoneContextScanPositions uint64
}

// TickStageSnapshot describes the recent cost of one stage of world ticking.
type TickStageSnapshot struct {
	Name    string
	Last    time.Duration
	Average time.Duration
	P95     time.Duration
	Count   int
}

// tickPerformance is updated by the world's owner and deliberately avoids
// locks and allocations on the normal tick path.
type tickPerformance struct {
	total                     uint64
	last                      time.Duration
	average                   time.Duration
	max                       time.Duration
	samples                   [performanceTickSamples]time.Duration
	sampleCount               int
	sampleIndex               int
	stages                    [tickStageCount]tickStagePerformance
	stageRecorded             [tickStageCount]bool
	queueWait                 tickStagePerformance
	neighbourUpdatesProcessed       int
	neighbourUpdatesSkippedUnloaded int
	neighbourBlockCallbacks         int
	neighbourLiquidCallbacks        int
	windowStart               time.Time
	windowTicks               uint64
	tps                       float64
}

type tickStagePerformance struct {
	last        time.Duration
	average     time.Duration
	max         time.Duration
	samples     [performanceTickSamples]time.Duration
	sampleCount int
	sampleIndex int
}

func (m *tickPerformance) beginTick() {
	m.neighbourUpdatesProcessed = 0
	m.neighbourUpdatesSkippedUnloaded = 0
	m.neighbourBlockCallbacks = 0
	m.neighbourLiquidCallbacks = 0
	clear(m.stageRecorded[:])
	for i := range m.stages {
		m.stages[i].last = 0
	}
}

func (m *tickPerformance) recordStage(stage TickStage, duration time.Duration) {
	m.stageRecorded[stage] = true
	m.stages[stage].record(duration)
}

// recordUnrecordedStages adds zero-duration samples for stages skipped by an early tick return.
func (m *tickPerformance) recordUnrecordedStages() {
	for stage := TickStageLightning; stage < TickStageOther; stage++ {
		if !m.stageRecorded[stage] {
			m.recordStage(stage, 0)
		}
	}
}

func (m *tickPerformance) recordQueueWait(duration time.Duration) {
	m.queueWait.record(duration)
}

func (m *tickStagePerformance) record(duration time.Duration) {
	m.last = duration
	if m.sampleCount == 0 {
		m.average = duration
	} else {
		m.average += (duration - m.average) / 20
	}
	if duration > m.max {
		m.max = duration
	}
	m.samples[m.sampleIndex] = duration
	m.sampleIndex = (m.sampleIndex + 1) % len(m.samples)
	if m.sampleCount < len(m.samples) {
		m.sampleCount++
	}
}

func (m *tickStagePerformance) percentile95() time.Duration {
	if m.sampleCount == 0 {
		return 0
	}
	samples := make([]time.Duration, m.sampleCount)
	copy(samples, m.samples[:m.sampleCount])
	slices.Sort(samples)
	rank := (len(samples)*95 + 99) / 100
	return samples[rank-1]
}

func (m *tickStagePerformance) recentMax() time.Duration {
	var maximum time.Duration
	for _, duration := range m.samples[:m.sampleCount] {
		if duration > maximum {
			maximum = duration
		}
	}
	return maximum
}

func (m *tickPerformance) record(now time.Time, duration time.Duration) {
	m.total++
	m.last = duration
	if m.total == 1 {
		m.average = duration
	} else {
		// A 20-sample exponential moving average smooths one-off spikes.
		m.average += (duration - m.average) / 20
	}
	if duration > m.max {
		m.max = duration
	}
	m.samples[m.sampleIndex] = duration
	m.sampleIndex = (m.sampleIndex + 1) % len(m.samples)
	if m.sampleCount < len(m.samples) {
		m.sampleCount++
	}
	if m.windowStart.IsZero() {
		m.windowStart = now
	} else {
		m.windowTicks++
	}
	if elapsed := now.Sub(m.windowStart); elapsed >= time.Second {
		m.tps = float64(m.windowTicks) / elapsed.Seconds()
		m.windowStart = now
		m.windowTicks = 0
	}
}

func (m *tickPerformance) percentile95() time.Duration {
	if m.sampleCount == 0 {
		return 0
	}
	samples := make([]time.Duration, m.sampleCount)
	copy(samples, m.samples[:m.sampleCount])
	slices.Sort(samples)
	rank := (len(samples)*95 + 99) / 100
	return samples[rank-1]
}

func (m *tickPerformance) recentMax() time.Duration {
	var maximum time.Duration
	for _, duration := range m.samples[:m.sampleCount] {
		if duration > maximum {
			maximum = duration
		}
	}
	return maximum
}

func (w *World) recordTickPerformance(start time.Time) {
	now := time.Now()
	duration := now.Sub(start)
	w.performance.recordUnrecordedStages()
	var stageDuration time.Duration
	for i := TickStageLightning; i < TickStageOther; i++ {
		stageDuration += w.performance.stages[i].last
	}
	w.performance.recordStage(TickStageOther, max(duration-stageDuration, 0))
	w.performance.record(now, duration)
}

// Performance returns tick and workload metrics for the World being operated
// on by tx. It must only be called from a valid world transaction.
func (tx *Tx) Performance() PerformanceSnapshot {
	w := tx.World()
	m := &w.performance
	tps := m.tps
	if m.windowTicks > 0 {
		if elapsed := time.Since(m.windowStart); elapsed > 0 {
			tps = float64(m.windowTicks) / elapsed.Seconds()
		}
	}
	w.viewerMu.Lock()
	activeLoaders := len(w.viewers)
	w.viewerMu.Unlock()
	queueDepth, queueCapacity := 0, 0
	if w.chunkWorkers != nil {
		queueDepth = len(w.chunkWorkers.queue)
		queueCapacity = cap(w.chunkWorkers.queue)
	}
	var stages [tickStageCount]TickStageSnapshot
	for i := range stages {
		stage := &m.stages[i]
		stages[i] = TickStageSnapshot{
			Name:    tickStageNames[i],
			Last:    stage.last,
			Average: stage.average,
			P95:     stage.percentile95(),
			Count:   stage.sampleCount,
		}
	}
	return PerformanceSnapshot{
		TickCount: m.total, TicksPerSecond: tps, LastTick: m.last,
		AverageTick: m.average, P95Tick: m.percentile95(),
		RecentMaxTick: m.recentMax(), MaxTick: m.max,
		LoadedChunks: len(w.chunks), LoadingChunks: len(w.chunkRequests),
		QueuedChunkLoads: queueDepth, ChunkLoadQueueCapacity: queueCapacity,
		ChunkLoadWorkers: w.conf.ChunkLoadWorkers, LoadedEntities: len(w.entities),
		ActiveLoaders: activeLoaders, SimulationDistance: w.tickRange(),
		RandomTickSpeed: w.conf.RandomTickSpeed, Stages: stages,
		QueueWait: TickStageSnapshot{
			Name:    "world_queue_wait",
			Last:    m.queueWait.last,
			Average: m.queueWait.average,
			P95:     m.queueWait.percentile95(),
			Count:   m.queueWait.sampleCount,
		},
		NeighbourUpdatesProcessed:       m.neighbourUpdatesProcessed,
		NeighbourUpdatesSkippedUnloaded: m.neighbourUpdatesSkippedUnloaded,
		NeighbourBlockCallbacks:         m.neighbourBlockCallbacks,
		NeighbourLiquidCallbacks:        m.neighbourLiquidCallbacks,
		PendingNeighbourUpdates:      len(w.neighbourUpdates),
		RedstoneDirtyPositions:       w.redstone.lastDirtyPositions,
		RedstoneGraphNodes:           w.redstone.lastGraphNodes,
		RedstoneGraphEdges:           w.redstone.lastGraphEdges,
		RedstoneContextScanPositions: w.redstone.lastContextScanPositions,
	}
}
