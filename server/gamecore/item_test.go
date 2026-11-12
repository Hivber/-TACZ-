package gamecore

import (
	"encoding/json"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestEncodeItemStackPreservesDragonflyNBT(t *testing.T) {
	stack := item.NewStack(item.Apple{}, 3).
		WithCustomName("Field ration").
		WithLore("Prepared for testing", "Do not discard")
	got, err := EncodeItemStack(stack)
	if err != nil {
		t.Fatalf("EncodeItemStack() error = %v", err)
	}
	if got.Identifier != "minecraft:apple" || got.Count != 3 || got.Metadata != 0 {
		t.Fatalf("unexpected item identity: %+v", got)
	}
	payload := make([]byte, len(got.UserData))
	for i, value := range got.UserData {
		if value > 255 {
			t.Fatalf("user_data[%d] is not a byte: %d", i, value)
		}
		payload[i] = byte(value)
	}
	var envelope struct {
		Format string         `json:"format"`
		NBT    map[string]any `json:"nbt"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode user data: %v", err)
	}
	if envelope.Format != "dragonfly-item-nbt-json-v1" {
		t.Fatalf("unexpected item payload format %q", envelope.Format)
	}
	tag, ok := envelope.NBT["tag"].(map[string]any)
	if !ok {
		t.Fatalf("disk NBT tag was not preserved: %#v", envelope.NBT)
	}
	display, ok := tag["display"].(map[string]any)
	if !ok || display["Name"] != "Field ration" {
		t.Fatalf("custom name was not preserved: %#v", tag["display"])
	}
	lore, ok := display["Lore"].([]any)
	if !ok || len(lore) != 2 || lore[0] != "Prepared for testing" {
		t.Fatalf("lore was not preserved: %#v", display["Lore"])
	}
}

func TestEncodeItemStackEmptyAndOversized(t *testing.T) {
	if got, err := EncodeItemStack(item.Stack{}); err != nil || got != nil {
		t.Fatalf("empty stack = (%v, %v), want (nil, nil)", got, err)
	}
	stack := item.NewStack(item.Apple{}, 65)
	if _, err := EncodeItemStack(stack); err == nil {
		t.Fatal("expected oversized stack to be rejected")
	}
}
