package main

import (
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/pelletier/go-toml"
)

func TestFeatureFlagsTOMLRoundTrip(t *testing.T) {
	config := server.DefaultConfig()
	config.Features.JavaCompatibilityCommands = true
	config.Features.PerformanceDiagnostics = true
	config.Features.Pprof = true
	config.Features.PprofAddress = "localhost:6061"
	config.Features.GameCoreSocket = "/tmp/dragonfly-game-core-test.sock"

	data, err := toml.Marshal(config)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	var decoded server.UserConfig
	if err := toml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if decoded.Features != config.Features {
		t.Fatalf("feature flags round trip = %+v, want %+v", decoded.Features, config.Features)
	}
}

func TestLegacyConfigKeepsOptionalFeaturesDisabled(t *testing.T) {
	config := server.DefaultConfig()
	if err := toml.Unmarshal([]byte("[Server]\nName = \"Legacy config\"\n"), &config); err != nil {
		t.Fatalf("unmarshal legacy config: %v", err)
	}
	if config.Features.JavaCompatibilityCommands || config.Features.PerformanceDiagnostics || config.Features.Pprof || config.Features.GameCoreSocket != "" {
		t.Fatalf("legacy config unexpectedly enables optional features: %+v", config.Features)
	}
}
