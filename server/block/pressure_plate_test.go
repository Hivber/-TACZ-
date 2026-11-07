package block

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type pressurePlateLivingTestEntity struct {
	world.Entity
}

func (pressurePlateLivingTestEntity) Hurt(float64, world.DamageSource) (float64, bool) {
	return 0, true
}

func TestPressurePlateStatesAndHashes(t *testing.T) {
	states := allPressurePlates()
	wantStates := 0
	for kind := 0; kind < pressurePlateKindCount; kind++ {
		wantStates += pressurePlateStateCount(kind)
	}
	if len(states) != wantStates {
		t.Fatalf("pressure plate states = %d, want %d", len(states), wantStates)
	}

	seen := make(map[[2]uint64]struct{}, len(states))
	for _, state := range states {
		plate := state.(PressurePlate)
		name, properties := plate.EncodeBlock()
		if properties["redstone_signal"] != int32(plate.Power) {
			t.Fatalf("%s redstone_signal = %v, want %d", name, properties["redstone_signal"], plate.Power)
		}
		if plate.Power >= pressurePlateStateCount(plate.Kind) {
			t.Fatalf("%s registered unsupported power state %d", name, plate.Power)
		}
		base, encoded := plate.Hash()
		key := [2]uint64{base, encoded}
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate pressure plate state hash: %+v", plate)
		}
		seen[key] = struct{}{}
	}
	if len(pressurePlateBlockNames) != pressurePlateKindCount {
		t.Fatalf("pressure plate names = %d, want %d", len(pressurePlateBlockNames), pressurePlateKindCount)
	}
}

func TestWeightedPressurePlatePower(t *testing.T) {
	tests := []struct {
		name     string
		kind     int
		entities int
		want     int
	}{
		{name: "empty light weighted", kind: PressurePlateLightWeighted, entities: 0, want: 0},
		{name: "one entity light weighted", kind: PressurePlateLightWeighted, entities: 1, want: 1},
		{name: "maximum light weighted", kind: PressurePlateLightWeighted, entities: 20, want: 15},
		{name: "below heavy threshold", kind: PressurePlateHeavyWeighted, entities: 9, want: 0},
		{name: "heavy threshold", kind: PressurePlateHeavyWeighted, entities: 10, want: 1},
		{name: "heavy partial", kind: PressurePlateHeavyWeighted, entities: 149, want: 14},
		{name: "maximum heavy weighted", kind: PressurePlateHeavyWeighted, entities: 150, want: 15},
		{name: "stone plate", kind: PressurePlateStone, entities: 1, want: 15},
		{name: "empty wood plate", kind: PressurePlateOak, entities: 0, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := pressurePlatePower(test.kind, test.entities); got != test.want {
				t.Fatalf("pressure plate power = %d, want %d", got, test.want)
			}
		})
	}
}

func TestPressurePlateEntityStepAndRelease(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	pos := cube.Pos{0, 0, 0}
	plate := PressurePlate{Kind: PressurePlateStone}
	if err := w.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, plate, nil)
		plate.EntityStepOn(pos, tx, pressurePlateLivingTestEntity{})
		if got := tx.Block(pos).(PressurePlate).Power; got != 1 {
			t.Fatalf("stone plate redstone_signal after a living entity stepped on it = %d, want 1", got)
		}
		current := tx.Block(pos).(PressurePlate)
		current.ScheduledTick(pos, tx, nil)
		if got := tx.Block(pos).(PressurePlate).Power; got != 0 {
			t.Fatalf("stone plate power after entity left = %d, want 0", got)
		}
	}).Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPressurePlatePowerOutputFaces(t *testing.T) {
	tests := []struct {
		name  string
		plate PressurePlate
		want  int
	}{
		{name: "weighted analogue", plate: PressurePlate{Kind: PressurePlateLightWeighted, Power: 11}, want: 11},
		{name: "ordinary digital", plate: PressurePlate{Kind: PressurePlateStone, Power: 1}, want: 15},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, face := range cube.Faces() {
				if got := test.plate.RedstonePower(cube.Pos{}, nil, face); got != test.want {
					t.Errorf("weak output on face %v = %d, want %d", face, got, test.want)
				}
				wantStrong := 0
				if face == cube.FaceDown {
					wantStrong = test.want
				}
				if got := test.plate.RedstoneStrongPower(cube.Pos{}, nil, face); got != wantStrong {
					t.Errorf("strong output on face %v = %d, want %d", face, got, wantStrong)
				}
			}
		})
	}
}
