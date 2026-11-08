package entity

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
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

func TestNaturalSpawnProfilesUseBiomeTagsAndLight(t *testing.T) {
	tests := []struct {
		name  string
		kind  mobKind
		biome world.Biome
		light uint8
		want  bool
	}{
		{name: "pig plains daytime", kind: PigType.kind, biome: biome.Plains{}, light: 15, want: true},
		{name: "pig desert excluded", kind: PigType.kind, biome: biome.Desert{}, light: 15, want: false},
		{name: "zombie desert night", kind: ZombieType.kind, biome: biome.Desert{}, light: 0, want: true},
		{name: "zombie bright surface excluded", kind: ZombieType.kind, biome: biome.Plains{}, light: 15, want: false},
		{name: "skeleton darkness threshold", kind: SkeletonType.kind, biome: biome.Forest{}, light: 7, want: true},
		{name: "skeleton too bright", kind: SkeletonType.kind, biome: biome.Forest{}, light: 8, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := mobKindCanSpawn(test.kind, test.biome, test.light); got != test.want {
				t.Fatalf("mobKindCanSpawn() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNaturalSpawnProfilesHaveValidGroupAndWeight(t *testing.T) {
	for id, typ := range mobTypes {
		kind := typ.kind
		if kind.spawnCategory == mobSpawnNone {
			continue
		}
		if kind.spawnWeight <= 0 {
			t.Errorf("%s has non-positive spawn weight %d", id, kind.spawnWeight)
		}
		if kind.groupMin < 1 || kind.groupMax < kind.groupMin {
			t.Errorf("%s has invalid group bounds %d..%d", id, kind.groupMin, kind.groupMax)
		}
		if kind.minLight > kind.maxLight || kind.maxLight > 15 {
			t.Errorf("%s has invalid light range %d..%d", id, kind.minLight, kind.maxLight)
		}
	}
}

func TestNaturalMobIsMarkedForDespawnPersistence(t *testing.T) {
	if !newMobBehaviour(PigType.kind, true).naturalSpawn {
		t.Fatal("natural mob behaviour must be marked as naturally spawned")
	}
	if newMobBehaviour(PigType.kind, false).naturalSpawn {
		t.Fatal("spawn-egg mob behaviour must not be marked as naturally spawned")
	}
}

func TestDeadMobStateSurvivesNBT(t *testing.T) {
	data := &world.EntityData{}
	ZombieType.DecodeNBT(map[string]any{
		"Health":       float32(0),
		"NaturalSpawn": true,
		"Dead":          true,
		"DeathTicks":    int32(9),
	}, data)

	state, ok := data.Data.(*mobBehaviour)
	if !ok {
		t.Fatal("decoded mob data does not contain mob behaviour")
	}
	if !state.dead {
		t.Fatal("dead state was not restored")
	}
	if state.health.Health() != 0 {
		t.Fatalf("restored health = %v, want 0", state.health.Health())
	}
	if state.deathTicks != 9 {
		t.Fatalf("restored death ticks = %d, want 9", state.deathTicks)
	}

	encoded := ZombieType.EncodeNBT(data)
	if encoded["Dead"] != true || encoded["DeathTicks"] != int32(9) {
		t.Fatalf("encoded death state = %#v", encoded)
	}
}
