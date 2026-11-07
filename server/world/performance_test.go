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
