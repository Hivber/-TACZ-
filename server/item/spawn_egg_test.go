package item

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

func TestSupportedSpawnEggItemsAreRegistered(t *testing.T) {
	mobs := []string{"pig", "cow", "sheep", "chicken", "zombie", "skeleton"}
	for _, mob := range mobs {
		name := "minecraft:" + mob + "_spawn_egg"
		got, ok := world.ItemByName(name, 0)
		if !ok {
			t.Errorf("spawn egg %q is not registered", name)
			continue
		}
		if _, ok := got.(SpawnEgg); !ok {
			t.Errorf("spawn egg %q resolved to %T, want item.SpawnEgg", name, got)
		}
	}
}

func TestSpawnEggProperties(t *testing.T) {
	egg := SpawnEgg{Mob: "minecraft:pig"}
	if egg.MaxCount() != 64 {
		t.Fatalf("spawn egg max count = %d, want 64", egg.MaxCount())
	}
	if name, meta := egg.EncodeItem(); name != "minecraft:pig_spawn_egg" || meta != 0 {
		t.Fatalf("spawn egg encodes as %q:%d", name, meta)
	}
}
