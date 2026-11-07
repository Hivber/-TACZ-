package block

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func TestTripwireHookStatesAndRedstoneOutput(t *testing.T) {
	states := allTripwireHooks()
	if len(states) != len(cube.Directions())*4 {
		t.Fatalf("tripwire hook states = %d, want %d", len(states), len(cube.Directions())*4)
	}

	seen := make(map[[2]uint64]struct{}, len(states))
	for _, state := range states {
		hook := state.(TripwireHook)
		name, properties := hook.EncodeBlock()
		if name != "minecraft:tripwire_hook" {
			t.Fatalf("tripwire hook block name = %q", name)
		}
		if got, want := properties["direction"], tripwireHookBedrockDirection(hook.Facing); got != want {
			t.Fatalf("tripwire hook direction = %v, want %v", got, want)
		}
		if properties["attached_bit"] != hook.Attached || properties["powered_bit"] != hook.Powered {
			t.Fatalf("tripwire hook properties do not match state: %+v", hook)
		}
		base, value := hook.Hash()
		key := [2]uint64{base, value}
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate tripwire hook hash for %+v", hook)
		}
		seen[key] = struct{}{}
	}

	for _, facing := range cube.Directions() {
		hook := TripwireHook{Facing: facing, Attached: true, Powered: true}
		if got := hook.RedstonePower(cube.Pos{}, nil, cube.FaceNorth); got != 15 {
			t.Errorf("weak output for facing %v = %d, want 15", facing, got)
		}
		for _, face := range cube.Faces() {
			want := 0
			if face == facing.Opposite().Face() {
				want = 15
			}
			if got := hook.RedstoneStrongPower(cube.Pos{}, nil, face); got != want {
				t.Errorf("strong output for facing %v on face %v = %d, want %d", facing, face, got, want)
			}
		}
	}
}

func TestTripwireLineEntityActivationAndRelease(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	leftHookPos, rightHookPos := cube.Pos{0, 64, 0}, cube.Pos{4, 64, 0}
	wirePos := cube.Pos{2, 64, 0}
	if err := w.Do(func(tx *world.Tx) {
		left := TripwireHook{Facing: cube.East}
		right := TripwireHook{Facing: cube.West}
		tx.SetBlock(leftHookPos, left, nil)
		tx.SetBlock(rightHookPos, right, nil)
		for x := 1; x <= 3; x++ {
			tx.SetBlock(cube.Pos{x, 64, 0}, String{}, nil)
		}
		updateTripwireHookLine(leftHookPos, left, tx)

		if !tx.Block(leftHookPos).(TripwireHook).Attached || !tx.Block(rightHookPos).(TripwireHook).Attached {
			t.Fatal("opposed hooks did not attach across a complete tripwire line")
		}
		for x := 1; x <= 3; x++ {
			wire := tx.Block(cube.Pos{x, 64, 0}).(String)
			if !wire.Attached {
				t.Fatalf("tripwire segment at x=%d was not marked attached", x)
			}
			if got := wire.Connections.Uint8(); got != 10 {
				t.Fatalf("tripwire segment at x=%d connections = %04b, want east/west connections", x, got)
			}
		}

		handle := (world.EntitySpawnOpts{
			Position: mgl64.Vec3{2.5, 64.45, 0.5},
		}).New(buttonTestArrowType{}, buttonTestArrowConfig{})
		arrow := tx.AddEntity(handle)
		wire := tx.Block(wirePos).(String)
		wire.EntityInside(wirePos, tx, arrow)

		if !tx.Block(wirePos).(String).Powered {
			t.Fatal("tripwire was not powered by an intersecting entity")
		}
		if !tx.Block(leftHookPos).(TripwireHook).Powered || !tx.Block(rightHookPos).(TripwireHook).Powered {
			t.Fatal("powered tripwire did not power both endpoint hooks")
		}

		tx.Block(wirePos).(String).ScheduledTick(wirePos, tx, nil)
		if !tx.Block(wirePos).(String).Powered {
			t.Fatal("tripwire released while an entity remained inside the wire plane")
		}

		tx.RemoveEntity(arrow)
		tx.Block(wirePos).(String).ScheduledTick(wirePos, tx, nil)
		if tx.Block(wirePos).(String).Powered {
			t.Fatal("tripwire remained powered after the entity left")
		}
		if tx.Block(leftHookPos).(TripwireHook).Powered || tx.Block(rightHookPos).(TripwireHook).Powered {
			t.Fatal("tripwire endpoint hooks remained powered after release")
		}
	}).Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDisarmedOrBrokenTripwireDetachesHooks(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	leftHookPos, rightHookPos := cube.Pos{0, 64, 0}, cube.Pos{4, 64, 0}
	disarmedPos := cube.Pos{2, 64, 0}
	if err := w.Do(func(tx *world.Tx) {
		left := TripwireHook{Facing: cube.East}
		right := TripwireHook{Facing: cube.West}
		tx.SetBlock(leftHookPos, left, nil)
		tx.SetBlock(rightHookPos, right, nil)
		for x := 1; x <= 3; x++ {
			tx.SetBlock(cube.Pos{x, 64, 0}, String{}, nil)
		}
		updateTripwireHookLine(leftHookPos, left, tx)
		if !tx.Block(leftHookPos).(TripwireHook).Attached {
			t.Fatal("tripwire did not initially attach")
		}

		wire := tx.Block(disarmedPos).(String)
		wire.Disarmed = true
		tx.SetBlock(disarmedPos, wire, nil)
		updateTripwireHookLine(leftHookPos, left, tx)
		if tx.Block(leftHookPos).(TripwireHook).Attached || tx.Block(rightHookPos).(TripwireHook).Attached {
			t.Fatal("disarmed segment did not detach the tripwire hooks")
		}

		wire.Disarmed = false
		tx.SetBlock(disarmedPos, wire, nil)
		updateTripwireHookLine(leftHookPos, left, tx)
		if !tx.Block(leftHookPos).(TripwireHook).Attached {
			t.Fatal("tripwire did not reattach after the disarmed state was cleared")
		}

		tx.SetBlock(disarmedPos, Air{}, nil)
		updateTripwireLinesForWire(disarmedPos, tx)
		if tx.Block(leftHookPos).(TripwireHook).Attached || tx.Block(rightHookPos).(TripwireHook).Attached {
			t.Fatal("missing wire segment did not detach the tripwire hooks")
		}
	}).Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}
