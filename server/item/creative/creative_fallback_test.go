package creative

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestCreativeCatalogueResolvesKnownVanillaItemsWithoutDedicatedImplementations(t *testing.T) {
	var catalogue struct {
		Groups []creativeGroupEntry `nbt:"groups"`
		Items  []creativeItemEntry  `nbt:"items"`
	}
	if err := nbt.Unmarshal(creativeItemData, &catalogue); err != nil {
		t.Fatalf("decode embedded creative catalogue: %v", err)
	}

	want := []string{
		"minecraft:fishing_rod",
		"minecraft:saddle",
		"minecraft:brush",
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

func TestCreativeCatalogueUsesDedicatedShieldAndTridentImplementations(t *testing.T) {
	var catalogue struct {
		Groups []creativeGroupEntry `nbt:"groups"`
		Items  []creativeItemEntry  `nbt:"items"`
	}
	if err := nbt.Unmarshal(creativeItemData, &catalogue); err != nil {
		t.Fatalf("decode embedded creative catalogue: %v", err)
	}

	for name, check := range map[string]func(item.Stack) bool{
		"minecraft:shield":      func(s item.Stack) bool { _, ok := s.Item().(item.Shield); return ok },
		"minecraft:trident":     func(s item.Stack) bool { _, ok := s.Item().(item.Trident); return ok },
		"minecraft:wind_charge": func(s item.Stack) bool { _, ok := s.Item().(item.WindCharge); return ok },
		"minecraft:pig_spawn_egg":     func(s item.Stack) bool { _, ok := s.Item().(item.SpawnEgg); return ok },
		"minecraft:cow_spawn_egg":     func(s item.Stack) bool { _, ok := s.Item().(item.SpawnEgg); return ok },
		"minecraft:sheep_spawn_egg":   func(s item.Stack) bool { _, ok := s.Item().(item.SpawnEgg); return ok },
		"minecraft:chicken_spawn_egg": func(s item.Stack) bool { _, ok := s.Item().(item.SpawnEgg); return ok },
		"minecraft:zombie_spawn_egg":  func(s item.Stack) bool { _, ok := s.Item().(item.SpawnEgg); return ok },
		"minecraft:skeleton_spawn_egg": func(s item.Stack) bool { _, ok := s.Item().(item.SpawnEgg); return ok },
	} {
		t.Run(name, func(t *testing.T) {
			for _, entry := range catalogue.Items {
				if entry.Name != name || len(entry.BlockProperties) != 0 {
					continue
				}
				stack, ok := itemStackFromEntry(entry)
				if !ok {
					t.Fatalf("creative item %q could not be resolved", name)
				}
				if !check(stack) {
					t.Fatalf("creative item %q resolved to %T, want dedicated gameplay implementation", name, stack.Item())
				}
				return
			}
			t.Fatalf("creative catalogue does not contain %q", name)
		})
	}
}
