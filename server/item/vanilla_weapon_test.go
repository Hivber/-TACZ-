package item

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

func TestShieldGameplayProperties(t *testing.T) {
	shield := Shield{}
	if shield.MaxCount() != 1 {
		t.Fatalf("shield max count = %d, want 1", shield.MaxCount())
	}
	if !shield.OffHand() {
		t.Fatal("shield must be usable in the off-hand")
	}
	if got := shield.DurabilityInfo().MaxDurability; got != 336 {
		t.Fatalf("shield durability = %d, want 336", got)
	}
	if name, meta := shield.EncodeItem(); name != "minecraft:shield" || meta != 0 {
		t.Fatalf("shield encodes as %q:%d", name, meta)
	}
}

func TestTridentMeleeProperties(t *testing.T) {
	trident := Trident{}
	if trident.MaxCount() != 1 {
		t.Fatalf("trident max count = %d, want 1", trident.MaxCount())
	}
	if got := trident.AttackDamage(); got != 9 {
		t.Fatalf("trident melee damage = %v, want 9", got)
	}
	info := trident.DurabilityInfo()
	if info.MaxDurability != 251 || info.AttackDurability != 1 {
		t.Fatalf("unexpected trident durability info: %+v", info)
	}
	if name, meta := trident.EncodeItem(); name != "minecraft:trident" || meta != 0 {
		t.Fatalf("trident encodes as %q:%d", name, meta)
	}
}

func TestItemByNameUsesDedicatedShieldAndTridentImplementations(t *testing.T) {
	for name, check := range map[string]func(world.Item) bool{
		"minecraft:shield":  func(i world.Item) bool { _, ok := i.(Shield); return ok },
		"minecraft:trident": func(i world.Item) bool { _, ok := i.(Trident); return ok },
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := world.ItemByName(name, 0)
			if !ok {
				t.Fatalf("ItemByName(%q) did not resolve", name)
			}
			if !check(got) {
				t.Fatalf("ItemByName(%q) returned %T, want dedicated implementation", name, got)
			}
		})
	}
}
