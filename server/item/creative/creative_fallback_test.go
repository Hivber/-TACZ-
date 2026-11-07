package creative

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestCreativeCatalogueResolvesKnownVanillaItemsWithoutDedicatedImplementations(t *testing.T) {
	var catalogue struct {
		Items []creativeItemEntry `nbt:"items"`
	}
	if err := nbt.Unmarshal(creativeItemData, &catalogue); err != nil {
		t.Fatalf("decode embedded creative catalogue: %v", err)
	}

	want := []string{
		"minecraft:shield",
		"minecraft:fishing_rod",
		"minecraft:saddle",
		"minecraft:trident",
		"minecraft:brush",
		"minecraft:wind_charge",
		"minecraft:bundle",
		"minecraft:wooden_spear",
		"minecraft:breeze_rod",
		"minecraft:trial_key",
		"minecraft:ominous_trial_key",
		"minecraft:ominous_bottle",
	}
	found := make(map[string]struct{}, len(want))
	for _, name := range want {
		var entry *creativeItemEntry
		for i := range catalogue.Items {
			if catalogue.Items[i].Name == name && len(catalogue.Items[i].BlockProperties) == 0 {
				entry = &catalogue.Items[i]
				break
			}
		}
		if entry == nil {
			t.Errorf("vanilla creative catalogue does not contain item %q", name)
			continue
		}
		stack, ok := itemStackFromEntry(*entry)
		if !ok {
			t.Errorf("vanilla creative entry %q could not be resolved", name)
			continue
		}
		if got, _ := stack.Item().EncodeItem(); got != name {
			t.Errorf("creative item %q resolved as %q", name, got)
		}
		found[name] = struct{}{}
	}
	if len(found) != len(want) {
		t.Fatalf("resolved %d of %d expected vanilla items", len(found), len(want))
	}
}
