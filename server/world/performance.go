package world

import (
	"slices"
	"time"
)

const performanceTickSamples = 100

// PerformanceSnapshot contains a point-in-time view of World tick and workload
// metrics. A snapshot must be obtained from an active world transaction.
type PerformanceSnapshot struct {
	TickCount             uint64
	TicksPerSecond        float64
	LastTick              time.Duration
	AverageTick           time.Duration
	P95Tick               time.Duration
	MaxTick               time.Duration
	LoadedChunks          int
	LoadingChunks         int
	QueuedChunkLoads      int
	ChunkLoadQueueCapacity int
	ChunkLoadWorkers      int
	LoadedEntities        int
	ActiveLoaders         int
	SimulationDistance   int
	RandomTickSpeed       int
}

// tickPerformance is updated by the world's owner and deliberately avoids
// locks and allocations on the normal tick path.
type tickPerformance struct {
	total       uint64
	last        time.Duration
	average     time.Duration
	max         time.Duration
	samples     [performanceTickSamples]time.Duration
	sampleCount int
	sampleIndex int

	windowStart time.Time
	windowTicks uint64
	tps         float64
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
	}
	m.windowTicks++
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

func (w *World) recordTickPerformance(start time.Time) {
	now := time.Now()
	w.performance.record(now, now.Sub(start))
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

	return PerformanceSnapshot{
		TickCount:              m.total,
		TicksPerSecond:         tps,
		LastTick:               m.last,
		AverageTick:            m.average,
		P95Tick:                m.percentile95(),
		MaxTick:                m.max,
		LoadedChunks:           len(w.chunks),
		LoadingChunks:          len(w.chunkRequests),
		QueuedChunkLoads:       queueDepth,
		ChunkLoadQueueCapacity: queueCapacity,
		ChunkLoadWorkers:       w.conf.ChunkLoadWorkers,
		LoadedEntities:         len(w.entities),
		ActiveLoaders:          activeLoaders,
		SimulationDistance:     w.tickRange(),
		RandomTickSpeed:        w.conf.RandomTickSpeed,
	}
}
