package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestRedstoneRepeaterStates(t *testing.T) {
	states := allRedstoneRepeaters()
	if len(states) != 32 {
		t.Fatalf("repeater states = %d, want 32", len(states))
	}
	seen := make(map[[2]uint64]struct{}, len(states))
	for _, state := range states {
		repeater := state.(RedstoneRepeater)
		name, properties := repeater.EncodeBlock()
		if name != "minecraft:unpowered_repeater" && name != "minecraft:powered_repeater" {
			t.Fatalf("unexpected repeater block name %q", name)
		}
		if properties["repeater_delay"] != int32(repeater.Delay-1) {
			t.Fatalf("repeater delay property = %v, want %d", properties["repeater_delay"], repeater.Delay-1)
		}
		if properties["minecraft:cardinal_direction"] != repeater.Direction.Opposite().String() {
			t.Fatalf("repeater cardinal direction = %v, want input-facing %s", properties["minecraft:cardinal_direction"], repeater.Direction.Opposite())
		}
		if _, hasPoweredBit := properties["powered_bit"]; hasPoweredBit {
			t.Fatal("repeater powered state must be represented by the powered/unpowered block identifier")
		}
		base, encodedState := repeater.Hash()
		key := [2]uint64{base, encodedState}
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate repeater block hash for %+v", repeater)
		}
		seen[key] = struct{}{}
	}
}

func TestObserverStatesAndOutputFace(t *testing.T) {
	states := allObservers()
	if len(states) != 12 {
		t.Fatalf("observer states = %d, want 12", len(states))
	}
	for _, face := range cube.Faces() {
		observer := Observer{Facing: face, Powered: true}
		if got := observer.RedstonePower(cube.Pos{}, nil, face.Opposite()); got != 15 {
			t.Fatalf("observer facing %v output = %d, want 15 on opposite face", face, got)
		}
		if got := observer.RedstonePower(cube.Pos{}, nil, face); got != 0 {
			t.Fatalf("observer facing %v output = %d on monitored face, want 0", face, got)
		}
	}
}

func TestRepeaterOnlyPowersItsOutputDirection(t *testing.T) {
	for _, direction := range cube.Directions() {
		repeater := RedstoneRepeater{Direction: direction, Delay: 1, Powered: true}
		if got := repeater.RedstonePower(cube.Pos{}, nil, direction.Face()); got != 15 {
			t.Fatalf("repeater facing %v output = %d, want 15", direction, got)
		}
		if got := repeater.RedstonePower(cube.Pos{}, nil, direction.Opposite().Face()); got != 0 {
			t.Fatalf("repeater facing %v outputs backwards: %d", direction, got)
		}
	}
}
