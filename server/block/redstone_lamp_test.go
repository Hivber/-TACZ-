package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func TestRedstoneLampStatesAndLight(t *testing.T) {
	states := allRedstoneLamps()
	if len(states) != 2 {
		t.Fatalf("redstone lamp states = %d, want 2", len(states))
	}
	seen := make(map[[2]uint64]struct{}, len(states))
	for _, state := range states {
		lamp := state.(RedstoneLamp)
		name, properties := lamp.EncodeBlock()
		if len(properties) != 0 {
			t.Fatalf("redstone lamp block properties = %v, want none", properties)
		}
		if lamp.Lit {
			if name != "minecraft:lit_redstone_lamp" || lamp.LightEmissionLevel() != 15 {
				t.Fatalf("lit lamp encodes as %q and emits %d light", name, lamp.LightEmissionLevel())
			}
		} else if name != "minecraft:redstone_lamp" || lamp.LightEmissionLevel() != 0 {
			t.Fatalf("unlit lamp encodes as %q and emits %d light", name, lamp.LightEmissionLevel())
		}
		base, encodedState := lamp.Hash()
		key := [2]uint64{base, encodedState}
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate redstone lamp hash for state %#v", lamp)
		}
		seen[key] = struct{}{}
	}
	if name, meta := (RedstoneLamp{Lit: true}).EncodeItem(); name != "minecraft:redstone_lamp" || meta != 0 {
		t.Fatalf("lit lamp item = %q:%d, want minecraft:redstone_lamp:0", name, meta)
	}
}

func TestRedstoneLampDelayedTurnOffAndRefresh(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	loader := world.NewLoader(1, w, world.NopViewer{})
	runWorld(w, func(tx *world.Tx) {
		loader.Move(tx, mgl64.Vec3{1, 64, 1})
		loader.Load(tx, 1)
	})
	defer func() {
		runWorld(w, func(tx *world.Tx) {
			loader.Close(tx)
		})
	}()

	lampPos := cube.Pos{1, 64, 1}
	sourcePos := lampPos.Side(cube.FaceWest)
	lit := func() bool {
		var got bool
		runWorld(w, func(tx *world.Tx) {
			got = tx.Block(lampPos).(RedstoneLamp).Lit
		})
		return got
	}

	runWorld(w, func(tx *world.Tx) {
		tx.SetBlock(lampPos, RedstoneLamp{}, nil)
		tx.SetBlock(sourcePos, RedstoneBlock{}, nil)
	})
	w.AdvanceTick()
	if !lit() {
		t.Fatal("redstone lamp did not turn on when powered")
	}

	runWorld(w, func(tx *world.Tx) {
		tx.SetBlock(sourcePos, nil, nil)
	})
	w.AdvanceTick()
	if !lit() {
		t.Fatal("redstone lamp turned off immediately instead of waiting four game ticks")
	}

	// Restore power before the delayed update expires. The stale off timer must
	// not turn the lamp off while it is powered again.
	runWorld(w, func(tx *world.Tx) {
		tx.SetBlock(sourcePos, RedstoneBlock{}, nil)
	})
	w.AdvanceTick()
	for range 4 {
		w.AdvanceTick()
	}
	if !lit() {
		t.Fatal("an old delayed update turned the lamp off after power was restored")
	}

	runWorld(w, func(tx *world.Tx) {
		tx.SetBlock(sourcePos, nil, nil)
	})
	w.AdvanceTick()
	if !lit() {
		t.Fatal("redstone lamp turned off before its second delayed turn-off completed")
	}
	for range 3 {
		w.AdvanceTick()
		if !lit() {
			t.Fatal("redstone lamp turned off before four game ticks elapsed")
		}
	}
	w.AdvanceTick()
	if lit() {
		t.Fatal("redstone lamp remained lit after four unpowered game ticks")
	}
}
