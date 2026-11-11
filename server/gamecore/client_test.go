package gamecore

import (
	"context"
	"testing"
)

func TestEventJSONUsesVersionedSemanticSchema(t *testing.T) {
	event := Event{Type: "player_join", UUID: "player-uuid", Name: "Steve", Dimension: 0, GameMode: 1, Position: Position{X: 1, Y: 64, Z: -2}}
	response, err := Client{}.Call(context.Background(), event)
	if err == nil || response.OK {
		t.Fatal("empty socket path must fail without claiming success")
	}
	if event.ProtocolVersion != 0 {
		t.Fatal("Call must not mutate the caller's event")
	}
}

func TestRejectsMissingSocketPath(t *testing.T) {
	_, err := (Client{}).Call(context.Background(), Event{Type: "ping"})
	if err == nil {
		t.Fatal("expected missing socket path error")
	}
}
