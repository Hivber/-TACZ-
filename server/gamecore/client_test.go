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
	event := Event{Type: "inventory_insert", InventoryID: "player:player-uuid:carried", Stack: stack}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["type"] != "inventory_insert" || decoded["inventory_id"] != "player:player-uuid:carried" {
		t.Fatalf("unexpected inventory event JSON: %s", payload)
	}
	var response Response
	raw := []byte(`{"protocol_version":2,"ok":true,"message":"batch_applied","online_players":1,"processed_events":1,"inventory_results":[{"inventory_id":"player:player-uuid:carried","status":"inserted","slots":[{"identifier":"minecraft:cobblestone","count":32,"metadata":0}],"remainder":null}]}`)
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.InventoryResults) != 1 || response.InventoryResults[0].Status != "inserted" {
		t.Fatalf("unexpected inventory response: %+v", response.InventoryResults)
	}
}

func TestInventorySyncSnapshotJSONPreservesEmptySlots(t *testing.T) {
	event := Event{
		Type:        "inventory_sync",
		InventoryID: "player:player-uuid:carried",
		Slots: []*ItemStack{
			{Identifier: "minecraft:diamond", Count: 3},
			nil,
		},
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	slots, ok := decoded["slots"].([]any)
	if !ok || len(slots) != 2 || slots[1] != nil {
		t.Fatalf("inventory snapshot must preserve empty slot positions: %s", payload)
	}
}

func TestInventoryTransferEventJSON(t *testing.T) {
	in := Event{Type: "inventory_transfer", SourceInventoryID: "player:alice:carried", DestinationInventoryID: "world-container:0:1:64:2", SourceSlot: 4, DestinationSlot: 8, Count: 3}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Event
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != in.Type || out.SourceInventoryID != in.SourceInventoryID || out.DestinationInventoryID != in.DestinationInventoryID || out.SourceSlot != 4 || out.DestinationSlot != 8 || out.Count != 3 {
		t.Fatalf("inventory transfer event round-trip mismatch: %#v", out)
	}
}

func TestInventoryTransferResponseJSON(t *testing.T) {
	in := InventoryResult{InventoryID: "player:alice:carried", Status: "transferred", RelatedInventoryID: "world-container:0:1:64:2", Slots: []*ItemStack{nil}, RelatedSlots: []*ItemStack{{Identifier: "minecraft:diamond", Count: 2}}}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out InventoryResult
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.RelatedInventoryID != in.RelatedInventoryID || len(out.RelatedSlots) != 1 || out.RelatedSlots[0].Count != 2 {
		t.Fatalf("related inventory snapshot lost: %#v", out)
	}
}

func TestInventorySyncPairEventJSON(t *testing.T) {
	in := Event{
		Type:               "inventory_sync_pair",
		InventoryID:        "player:alice:carried",
		Slots:              []*ItemStack{{Identifier: "minecraft:diamond", Count: 2}, nil},
		RelatedInventoryID: "world-container:0:1:64:2",
		RelatedSlots:       []*ItemStack{nil, {Identifier: "minecraft:diamond", Count: 2}},
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Event
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != in.Type || out.InventoryID != in.InventoryID ||
		out.RelatedInventoryID != in.RelatedInventoryID || len(out.Slots) != 2 ||
		len(out.RelatedSlots) != 2 || out.RelatedSlots[1].Count != 2 {
		t.Fatalf("atomic inventory snapshot pair round-trip mismatch: %#v", out)
	}
}
