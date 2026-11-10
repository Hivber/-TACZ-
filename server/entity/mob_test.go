package entity

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/go-gl/mathgl/mgl64"
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
		"Dead":         true,
		"DeathTicks":   int32(9),
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

// TestMobDespawnDistanceUsesThreeDimensions verifies vertical separation is
// included in the Euclidean player distance used by natural mob despawning.
func TestMobDespawnDistanceUsesThreeDimensions(t *testing.T) {
	if got := mobDistanceSquared(mgl64.Vec3{0, 0, 0}, mgl64.Vec3{0, 33, 0}); got != 33*33 {
		t.Fatalf("vertical distance squared = %v, want %v", got, 33*33)
	}
}

// TestMobImmediateDespawnDistanceMatchesSimulationDistance verifies the
// Bedrock immediate-despawn boundary for simulation distance 4 and 6+.
func TestMobImmediateDespawnDistanceMatchesSimulationDistance(t *testing.T) {
	tests := []struct {
		tickRange int
		want      float64
	}{
		{tickRange: 4, want: 44},
		{tickRange: 6, want: 128},
		{tickRange: 8, want: 128},
	}
	for _, test := range tests {
		if got := mobImmediateDespawnDistance(test.tickRange); got != test.want {
			t.Errorf("tick range %d immediate despawn distance = %v, want %v", test.tickRange, got, test.want)
		}
	}
}

// TestMobDespawnInactivityCounter verifies the 32-block reset and 30-second
// inactivity threshold used before random despawning begins.
func TestMobDespawnInactivityCounter(t *testing.T) {
	if got := nextMobDespawnTicks(32*32, 599); got != 0 {
		t.Fatalf("counter at 32 blocks = %d, want reset to 0", got)
	}
	if got := nextMobDespawnTicks(33*33, 598); got != 599 {
		t.Fatalf("counter before threshold = %d, want 599", got)
	}
	if got := nextMobDespawnTicks(33*33, 599); got != mobDespawnInactiveTicks {
		t.Fatalf("counter at threshold = %d, want %d", got, mobDespawnInactiveTicks)
	}
	if got := nextMobDespawnTicks(33*33, mobDespawnInactiveTicks); got != mobDespawnInactiveTicks {
		t.Fatalf("counter after threshold = %d, want saturation at %d", got, mobDespawnInactiveTicks)
	}
}

// TestMobDespawnInactivityCounterPersistsThroughNBT keeps the inactivity
// countdown stable across a chunk save and reload.
func TestMobDespawnInactivityCounterPersistsThroughNBT(t *testing.T) {
	data := &world.EntityData{}
	ZombieType.DecodeNBT(map[string]any{
		"DespawnTicks": int32(417),
	}, data)
	state := data.Data.(*mobBehaviour)
	if state.despawnTicks != 417 {
		t.Fatalf("decoded despawn ticks = %d, want 417", state.despawnTicks)
	}
	encoded := ZombieType.EncodeNBT(data)
	if encoded["DespawnTicks"] != int32(417) {
		t.Fatalf("encoded despawn ticks = %v, want 417", encoded["DespawnTicks"])
	}
}


// TestNaturalSpawnDistanceLimitsMatchBedrockSimulationDistance verifies the
// 24-44 and 24-128 spherical spawn shells and the larger-distance chunk limit.
func TestNaturalSpawnDistanceLimitsMatchBedrockSimulationDistance(t *testing.T) {
	tests := []struct {
		name                    string
		tickRange               int
		minimum, maximum, horizontal float64
	}{
		{name: "simulation distance 4", tickRange: 4, minimum: 24, maximum: 44, horizontal: 44},
		{name: "simulation distance 6", tickRange: 6, minimum: 24, maximum: 128, horizontal: 96},
		{name: "simulation distance 8", tickRange: 8, minimum: 24, maximum: 128, horizontal: 96},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			minimum, maximum, horizontal := naturalSpawnDistanceLimits(test.tickRange)
			if minimum != test.minimum || maximum != test.maximum || horizontal != test.horizontal {
				t.Fatalf("spawn limits = %.0f..%.0f (horizontal %.0f), want %.0f..%.0f (horizontal %.0f)",
					minimum, maximum, horizontal, test.minimum, test.maximum, test.horizontal)
			}
		})
	}
}

// TestNaturalSpawnDistanceUsesThreeDimensionalShell verifies that valid
// vertical distance can satisfy the spherical shell even when horizontal
// distance is small, while candidates outside the Euclidean bounds are rejected.
func TestNaturalSpawnDistanceUsesThreeDimensionalShell(t *testing.T) {
	player := mgl64.Vec3{8, 64, 8}
	tests := []struct {
		name string
		pos  cube.Pos
		minimum, maximum, horizontal float64
		want bool
	}{
		{name: "inside shell", pos: cube.Pos{8, 89, 8}, minimum: 24, maximum: 128, horizontal: 96, want: true},
		{name: "too close", pos: cube.Pos{8, 87, 8}, minimum: 24, maximum: 128, horizontal: 96, want: false},
		{name: "too far vertically", pos: cube.Pos{8, 193, 8}, minimum: 24, maximum: 128, horizontal: 96, want: false},
		{name: "simulation distance four upper edge", pos: cube.Pos{8, 108, 8}, minimum: 24, maximum: 44, horizontal: 44, want: true},
		{name: "outside simulation distance four", pos: cube.Pos{8, 109, 8}, minimum: 24, maximum: 44, horizontal: 44, want: false},
		{name: "chunk centre exceeds horizontal limit", pos: cube.Pos{112, 64, 8}, minimum: 24, maximum: 128, horizontal: 96, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := naturalSpawnDistanceAllowed(player, test.pos, test.minimum, test.maximum, test.horizontal)
			if got != test.want {
				t.Fatalf("naturalSpawnDistanceAllowed() = %v, want %v", got, test.want)
			}
		})
	}
}
