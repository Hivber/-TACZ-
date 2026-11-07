package creative

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

func TestCreativeInventoryIncludesKnownVanillaItemsWithoutDedicatedImplementations(t *testing.T) {
	registerCreativeItems()

	want := map[string]struct{}{
		"minecraft:shield":          {},
		"minecraft:fishing_rod":     {},
		"minecraft:saddle":          {},
		"minecraft:trident":         {},
		"minecraft:brush":           {},
		"minecraft:wind_charge":     {},
		"minecraft:bundle":          {},
		"minecraft:wooden_spear":    {},
		"minecraft:breeze_rod":      {},
		"minecraft:trial_key":       {},
		"minecraft:ominous_trial_key": {},
		"minecraft:ominous_bottle":  {},
	}
	found := make(map[string]struct{}, len(want))
	for _, entry := range Items() {
		it := entry.Stack.Item()
		if it == nil {
			continue
		}
		name, _ := it.EncodeItem()
		if _, ok := want[name]; ok {
			found[name] = struct{}{}
		}
	}
	for name := range want {
		if _, ok := found[name]; !ok {
			t.Errorf("vanilla creative item %q is still missing", name)
		}
	}
	for name := range found {
		it, ok := world.ItemByName(name, 0)
		if !ok {
			t.Errorf("creative item %q cannot be resolved through ItemByName", name)
			continue
		}
		if _, ok := it.(interface{ EncodeItem() (string, int16) }); !ok {
			t.Errorf("creative item %q returned unexpected type %T", name, it)
		}
	}
}
