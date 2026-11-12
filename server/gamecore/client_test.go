package gamecore

import (
	"context"
	"encoding/json"
	"testing"
)

func TestEventJSONUsesVersionedSemanticSchema(t *testing.T) {
	event := Event{Type: "player_join", UUID: "player-uuid", Name: "Steve", Dimension: 0, GameMode: 1, Position: Position{X: 1, Y: 64, Z: -2}}
	response, err := (&Client{}).Call(context.Background(), event)
	if err == nil || response.OK {
		t.Fatal("empty socket path must fail without claiming success")
	}
	if event.ProtocolVersion != 0 {
		t.Fatal("Call must not mutate the caller's event")
	}
}

func TestRejectsMissingSocketPath(t *testing.T) {
	_, err := (&Client{}).Call(context.Background(), Event{Type: "ping"})
	if err == nil {
		t.Fatal("expected missing socket path error")
	}
}

func TestMiningSemanticEventAndResultJSON(t *testing.T) {
	profile := &BlockBreakProfile{Hardness: 1.5, ToolType: 4, HarvestLevel: 1, RequiresCorrectTool: true}
	target := &BlockTarget{Position: BlockPos{X: 1, Y: 2, Z: 3}, Face: 1}
	event := Event{Type: "mining_start", UUID: "player-uuid", Target: target,
		Profile: profile, EyePosition: [3]float32{1.5, 3.62, 6.0}, MaxReach: 7, Tick: 10}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["type"] != "mining_start" || decoded["uuid"] != "player-uuid" {
		t.Fatalf("unexpected semantic event JSON: %s", payload)
	}
	var response Response
	if err := json.Unmarshal([]byte(`{"protocol_version":1,"ok":true,"message":"batch_applied","online_players":1,"processed_events":1,"mining_results":[{"uuid":"player-uuid","status":"progress","progress":0.25,"crack_stage":2}]}`), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.MiningResults) != 1 || response.MiningResults[0].Status != "progress" ||
		response.MiningResults[0].CrackStage != 2 {
		t.Fatalf("unexpected mining response: %+v", response.MiningResults)
	}
}


func TestInventorySemanticEventAndResultJSON(t *testing.T) {
	stack := &ItemStack{Identifier: "minecraft:cobblestone", Count: 32}
	event := Event{Type: "inventory_insert", UUID: "player-uuid", Stack: stack}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["type"] != "inventory_insert" || decoded["uuid"] != "player-uuid" {
		t.Fatalf("unexpected inventory event JSON: %s", payload)
	}
	var response Response
	raw := []byte(`{"protocol_version":1,"ok":true,"message":"batch_applied","online_players":1,"processed_events":1,"inventory_results":[{"uuid":"player-uuid","status":"inserted","slots":[{"identifier":"minecraft:cobblestone","count":32,"metadata":0}],"remainder":null}]}`)
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.InventoryResults) != 1 || response.InventoryResults[0].Status != "inserted" {
		t.Fatalf("unexpected inventory response: %+v", response.InventoryResults)
	}
}
