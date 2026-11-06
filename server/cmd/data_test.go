package cmd

import "testing"

func TestLookupNBTPath(t *testing.T) {
	root := map[string]any{
		"Items": []any{
			map[string]any{"id": "minecraft:diamond", "Count": int8(3)},
		},
		"CustomName": "Storage",
	}
	value, err := lookupNBTPath(root, "Items[0].id")
	if err != nil {
		t.Fatal(err)
	}
	if value != "minecraft:diamond" {
		t.Fatalf("got %v, want minecraft:diamond", value)
	}
	value, err = lookupNBTPath(root, "Items[0].Count")
	if err != nil {
		t.Fatal(err)
	}
	if value != int8(3) {
		t.Fatalf("got %v, want int8(3)", value)
	}
	value, err = lookupNBTPath(root, "CustomName")
	if err != nil {
		t.Fatal(err)
	}
	if value != "Storage" {
		t.Fatalf("got %v, want Storage", value)
	}
}

func TestLookupNBTPathRejectsInvalidPaths(t *testing.T) {
	root := map[string]any{"Items": []any{map[string]any{"id": "minecraft:stone"}}}
	for _, path := range []string{"Missing", "Items[3].id", "Items[-1]", "Items[x]", "Items[0", "Items[0].id.more"} {
		if _, err := lookupNBTPath(root, path); err == nil {
			t.Errorf("lookupNBTPath(%q) should fail", path)
		}
	}
}

func TestScaleNBTValue(t *testing.T) {
	value, err := scaleNBTValue(int16(12), 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if value != float64(6) {
		t.Fatalf("got %v, want 6", value)
	}
	if _, err := scaleNBTValue("not numeric", 2); err == nil {
		t.Fatal("expected non-numeric value to be rejected")
	}
}
