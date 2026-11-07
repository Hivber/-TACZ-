package world

import (
	"testing"
	"time"
)

func TestTickPerformanceRollingSamples(t *testing.T) {
	var metrics tickPerformance
	start := time.Now()
	for i := 1; i <= performanceTickSamples+20; i++ {
		metrics.record(start.Add(time.Duration(i)*50*time.Millisecond), time.Duration(i)*time.Millisecond)
	}

	if metrics.total != performanceTickSamples+20 {
		t.Fatalf("tick count = %d, want %d", metrics.total, performanceTickSamples+20)
	}
	if metrics.last != time.Duration(performanceTickSamples+20)*time.Millisecond {
		t.Fatalf("last tick = %s, want %s", metrics.last, time.Duration(performanceTickSamples+20)*time.Millisecond)
	}
	if metrics.max != metrics.last {
		t.Fatalf("max tick = %s, want %s", metrics.max, metrics.last)
	}
	if got, want := metrics.percentile95(), 115*time.Millisecond; got != want {
		t.Fatalf("P95 tick = %s, want %s", got, want)
	}
	if metrics.tps < 19 || metrics.tps > 21 {
		t.Fatalf("measured TPS = %.2f, want approximately 20", metrics.tps)
	}
}

func TestTickPerformanceEmptyPercentile(t *testing.T) {
	var metrics tickPerformance
	if got := metrics.percentile95(); got != 0 {
		t.Fatalf("empty P95 = %s, want 0", got)
	}
}

func TestTickStageRollingMetrics(t *testing.T) {
	var stage tickStagePerformance
	stage.record(500 * time.Millisecond)
	for i := 1; i <= performanceTickSamples; i++ {
		stage.record(time.Duration(i) * time.Millisecond)
	}
	if stage.sampleCount != performanceTickSamples {
		t.Fatalf("sample count = %d, want %d", stage.sampleCount, performanceTickSamples)
	}
	if got, want := stage.percentile95(), 95*time.Millisecond; got != want {
		t.Fatalf("P95 = %s, want %s", got, want)
	}
	if got, want := stage.recentMax(), 100*time.Millisecond; got != want {
		t.Fatalf("recent maximum = %s, want %s", got, want)
	}
	if got, want := stage.max, 500*time.Millisecond; got != want {
		t.Fatalf("lifetime maximum = %s, want %s", got, want)
	}
}

func TestTickPerformanceRecordsSkippedStagesAsZero(t *testing.T) {
	var metrics tickPerformance
	metrics.beginTick()
	metrics.recordStage(TickStageEntities, 5*time.Millisecond)
	metrics.recordUnrecordedStages()

	if got := metrics.stages[TickStageEntities].last; got != 5*time.Millisecond {
		t.Fatalf("entity stage = %s, want 5ms", got)
	}
	if got := metrics.stages[TickStageRedstone].last; got != 0 {
		t.Fatalf("skipped redstone stage = %s, want 0", got)
	}
	if got := metrics.stages[TickStageRedstone].sampleCount; got != 1 {
		t.Fatalf("skipped redstone sample count = %d, want 1", got)
	}

	metrics.beginTick()
	metrics.recordUnrecordedStages()
	if got := metrics.stages[TickStageEntities].sampleCount; got != 2 {
		t.Fatalf("entity stage sample count after skipped tick = %d, want 2", got)
	}
	if got := metrics.stages[TickStageEntities].average; got != 4750*time.Microsecond {
		t.Fatalf("entity stage EMA after skipped tick = %s, want 475µs", got)
	}
}
