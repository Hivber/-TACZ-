package entity

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

func TestDefaultRegistryContainsInitialMobTypes(t *testing.T) {
	want := []string{
		"minecraft:pig",
		"minecraft:cow",
		"minecraft:sheep",
		"minecraft:chicken",
		"minecraft:zombie",
		"minecraft:skeleton",
	}
	for _, name := range want {
		if _, ok := DefaultRegistry.Lookup(name); !ok {
			t.Errorf("default entity registry does not contain %q", name)
		}
	}
}

func TestNewMobRejectsUnknownType(t *testing.T) {
	if handle := NewMob(world.EntitySpawnOpts{}, "minecraft:unknown_mob"); handle != nil {
		t.Fatal("NewMob must reject an unknown mob identifier")
	}
}
