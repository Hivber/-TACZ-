package block

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
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

func TestTripwireHookModelHasWallMountedCollision(t *testing.T) {
	for _, facing := range cube.Directions() {
		model := TripwireHook{Facing: facing}.Model()
		boxes := model.BBox(cube.Pos{}, nil)
		if len(boxes) != 1 {
			t.Fatalf("tripwire hook facing %v has %d collision boxes, want 1", facing, len(boxes))
		}
		box := boxes[0]
		min, max := box.Min(), box.Max()
		switch facing {
		case cube.North:
			if min[2] < 14.0/16 || max[2] != 1 {
				t.Fatalf("north-facing hook collision = %v to %v, want against south wall", min, max)
			}
		case cube.South:
			if min[2] != 0 || max[2] > 2.0/16 {
				t.Fatalf("south-facing hook collision = %v to %v, want against north wall", min, max)
			}
		case cube.West:
			if min[0] < 14.0/16 || max[0] != 1 {
				t.Fatalf("west-facing hook collision = %v to %v, want against east wall", min, max)
			}
		case cube.East:
			if min[0] != 0 || max[0] > 2.0/16 {
				t.Fatalf("east-facing hook collision = %v to %v, want against west wall", min, max)
			}
		}
		for _, face := range cube.Faces() {
			if model.FaceSolid(cube.Pos{}, face, nil) {
				t.Fatalf("tripwire hook facing %v reports solid face %v", facing, face)
			}
		}
	}
}

type tripwireShearsTestUser struct {
	item.User
	held item.Stack
}

func (u tripwireShearsTestUser) HeldItems() (item.Stack, item.Stack) {
	return u.held, item.Stack{}
}

func TestTripwireBreakPulseAndShearsSuppression(t *testing.T) {
	for _, test := range []struct {
		name      string
		tool      item.Tool
		held      item.Stack
		wantPulse bool
		wantDrops int
	}{
		{name: "ordinary break", tool: item.ToolNone{}, held: item.NewStack(item.Stick{}, 1), wantPulse: true, wantDrops: 1},
		{name: "shears break", tool: item.Shears{}, held: item.NewStack(item.Shears{}, 1), wantPulse: false, wantDrops: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := world.Config{Synchronous: true}.New()
			defer w.Close()

			leftHookPos, rightHookPos := cube.Pos{0, 64, 0}, cube.Pos{4, 64, 0}
			wirePos := cube.Pos{2, 64, 0}
			var drops []item.Stack
			if err := w.Do(func(tx *world.Tx) {
				left := TripwireHook{Facing: cube.East}
				right := TripwireHook{Facing: cube.West}
				tx.SetBlock(leftHookPos, left, nil)
				tx.SetBlock(rightHookPos, right, nil)
				for x := 1; x <= 3; x++ {
					tx.SetBlock(cube.Pos{x, 64, 0}, String{}, nil)
				}
				updateTripwireHookLine(leftHookPos, left, tx)
				wire := tx.Block(wirePos).(String)
				info := wire.BreakInfo()
				drops = info.Drops(test.tool, nil)

				tx.SetBlock(wirePos, Air{}, nil)
				user := tripwireShearsTestUser{held: test.held}
				info.BreakHandler(wirePos, tx, user)

				leftAfter := tx.Block(leftHookPos).(TripwireHook)
				rightAfter := tx.Block(rightHookPos).(TripwireHook)
				if leftAfter.Attached || rightAfter.Attached {
					t.Fatal("broken tripwire left hooks attached")
				}
				if leftAfter.Powered != test.wantPulse || rightAfter.Powered != test.wantPulse {
					t.Fatalf("endpoint power = (%v, %v), want (%v, %v)", leftAfter.Powered, rightAfter.Powered, test.wantPulse, test.wantPulse)
				}
				if test.wantPulse {
					leftAfter.ScheduledTick(leftHookPos, tx, nil)
					rightAfter.ScheduledTick(rightHookPos, tx, nil)
					if tx.Block(leftHookPos).(TripwireHook).Powered || tx.Block(rightHookPos).(TripwireHook).Powered {
						t.Fatal("tripwire break pulse did not end")
					}
				}
			}).Wait(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(drops) != test.wantDrops {
				t.Fatalf("tripwire drops = %d, want %d", len(drops), test.wantDrops)
			}
		})
	}
}
