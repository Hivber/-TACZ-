package creative

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

func TestCreativeInventoryIncludesKnownVanillaItemsWithoutDedicatedImplementations(t *testing.T) {
	registerCreativeItems()

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
	for _, entry := range Items() {
		if it := entry.Stack.Item(); it != nil {
			name, _ := it.EncodeItem()
			found[name] = struct{}{}
		}
	}
	for _, name := range want {
		if _, ok := found[name]; !ok {
			t.Errorf("vanilla creative item %q is still missing", name)
		}
		it, ok := world.ItemByName(name, 0)
		if !ok {
			t.Errorf("vanilla item %q cannot be resolved through ItemByName", name)
			continue
		}
		if got, _ := it.EncodeItem(); got != name {
			t.Errorf("ItemByName(%q) encoded as %q", name, got)
		}
	}
}
