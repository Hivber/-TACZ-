package cmd

import (
	"reflect"
	"testing"
)

func TestParseCommandSNBT(t *testing.T) {
	value, err := parseCommandSNBT(`{Lock:"vault",Items:[{Slot:0b,Count:2b,id:"minecraft:stone"}],Damage:3s,Enabled:true}`)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"Lock":    "vault",
		"Items":   []any{map[string]any{"Slot": int8(0), "Count": int8(2), "id": "minecraft:stone"}},
		"Damage":  int16(3),
		"Enabled": true,
	}
	if !reflect.DeepEqual(value, want) {
		t.Fatalf("parsed SNBT = %#v, want %#v", value, want)
	}
}

func TestCommandNBTPathSetAndRemove(t *testing.T) {
	root := map[string]any{"Items": []any{map[string]any{"id": "minecraft:stone"}}}
	if err := setCommandNBTPath(root, "Items[0].Count", int8(4)); err != nil {
		t.Fatal(err)
	}
	if err := removeCommandNBTPath(root, "Items[0].id"); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"Items": []any{map[string]any{"Count": int8(4)}}}
	if !reflect.DeepEqual(root, want) {
		t.Fatalf("NBT after modify/remove = %#v, want %#v", root, want)
	}
}

func TestRemoveCommandNBTListElement(t *testing.T) {
	root := map[string]any{"Items": []any{"a", "b", "c"}}
	if err := removeCommandNBTPath(root, "Items[1]"); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"Items": []any{"a", "c"}}
	if !reflect.DeepEqual(root, want) {
		t.Fatalf("NBT after list removal = %#v, want %#v", root, want)
	}
}

func TestMergeCommandNBT(t *testing.T) {
	destination := map[string]any{"display": map[string]any{"Name": "old"}, "Damage": int32(1)}
	mergeCommandNBT(destination, map[string]any{"display": map[string]any{"Lore": []any{"line"}}, "Damage": int32(2)})
	want := map[string]any{"display": map[string]any{"Name": "old", "Lore": []any{"line"}}, "Damage": int32(2)}
	if !reflect.DeepEqual(destination, want) {
		t.Fatalf("merged NBT = %#v, want %#v", destination, want)
	}
}
