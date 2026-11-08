package session

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestClientDiagnosticsHandlerRequiresFeatureFlag(t *testing.T) {
	disabled := &Session{}
	disabled.registerHandlers()
	if disabled.handlers[packet.IDServerBoundDiagnostics] != nil {
		t.Fatal("client diagnostics handler must be disabled by default")
	}

	enabled := &Session{conf: Config{PerformanceDiagnostics: true}}
	enabled.registerHandlers()
	if _, ok := enabled.handlers[packet.IDServerBoundDiagnostics].(*ServerBoundDiagnosticsHandler); !ok {
		t.Fatal("client diagnostics handler was not registered when enabled")
	}
}
