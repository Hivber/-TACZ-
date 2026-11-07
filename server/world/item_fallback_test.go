package world

import "testing"

func TestItemByNameReturnsFallbackForKnownVanillaItems(t *testing.T) {
	names := []string{
		"minecraft:shield",
		"minecraft:fishing_rod",
		"minecraft:saddle",
		"minecraft:filled_map",
		"minecraft:trident",
		"minecraft:brush",
		"minecraft:wind_charge",
		"minecraft:bundle",
		"minecraft:wooden_spear",
		"minecraft:breeze_rod",
		"minecraft:trial_key",
		"minecraft:ominous_trial_key",
		"minecraft:ominous_bottle",
		"minecraft:glow_berries",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			got, ok := ItemByName(name, 0)
			if !ok {
				t.Fatalf("ItemByName(%q) did not resolve a known vanilla item", name)
			}
			if _, fallback := got.(vanillaItemFallback); !fallback {
				t.Fatalf("ItemByName(%q) returned %T, want vanillaItemFallback", name, got)
			}
			gotName, gotMeta := got.EncodeItem()
			if gotName != name || gotMeta != 0 {
				t.Fatalf("fallback encoded as %q:%d, want %q:0", gotName, gotMeta, name)
			}
			if _, _, ok := ItemRuntimeID(got); !ok {
				t.Fatalf("fallback item %q has no runtime ID", name)
			}
		})
	}
}

func TestItemByNameStillRejectsUnknownIdentifiers(t *testing.T) {
	if got, ok := ItemByName("minecraft:not_a_real_vanilla_item", 0); ok {
		t.Fatalf("ItemByName returned %T for an unknown item identifier", got)
	}
}
