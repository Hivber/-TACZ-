package server

import "testing"

func TestNormaliseChunkRadius(t *testing.T) {
	tests := []struct {
		configured int
		want       int
	}{
		{configured: 0, want: defaultChunkRadius},
		{configured: 8, want: 8},
		{configured: 10, want: 10},
		{configured: 32, want: maximumChunkRadius},
	}
	for _, test := range tests {
		if got := normaliseChunkRadius(test.configured, nil); got != test.want {
			t.Errorf("normaliseChunkRadius(%d) = %d, want %d", test.configured, got, test.want)
		}
	}
}

func TestNormaliseChunkLoadWorkers(t *testing.T) {
	tests := []struct {
		configured int
		want       int
	}{
		{configured: 0, want: 2},
		{configured: 1, want: 1},
		{configured: 4, want: 4},
		{configured: 32, want: maximumChunkLoadWorkers},
	}
	for _, test := range tests {
		if got := normaliseChunkLoadWorkers(test.configured); got != test.want {
			t.Errorf("normaliseChunkLoadWorkers(%d) = %d, want %d", test.configured, got, test.want)
		}
	}
}

func TestDefaultConfigOptionalFeaturesDisabled(t *testing.T) {
	conf := DefaultConfig()
	if conf.Features.JavaCompatibilityCommands {
		t.Fatal("Java compatibility commands must be disabled by default")
	}
	if conf.Features.PerformanceDiagnostics {
		t.Fatal("performance diagnostics command must be disabled by default")
	}
	if conf.Features.Pprof {
		t.Fatal("pprof must be disabled by default")
	}
	if conf.Features.PprofAddress != "127.0.0.1:6060" {
		t.Fatalf("default pprof address = %q, want 127.0.0.1:6060", conf.Features.PprofAddress)
	}
}
