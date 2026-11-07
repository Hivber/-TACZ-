package block

import (
	"context"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func TestButtonStatesAndHashes(t *testing.T) {
	states := allButtons()
	want := buttonKindCount * len(cube.Faces()) * 2
	if len(states) != want {
		t.Fatalf("button states = %d, want %d", len(states), want)
	}

	seen := make(map[[2]uint64]struct{}, len(states))
	for _, state := range states {
		button := state.(Button)
		name, properties := button.EncodeBlock()
		if name != buttonBlockNames[button.Kind] {
			t.Fatalf("button block name = %q, want %q", name, buttonBlockNames[button.Kind])
		}
		if properties["button_pressed_bit"] != button.Powered {
			t.Fatalf("%s button_pressed_bit = %v, want %v", name, properties["button_pressed_bit"], button.Powered)
		}
		if properties["facing_direction"] != int32(button.Facing) {
			t.Fatalf("%s facing_direction = %v, want %d", name, properties["facing_direction"], button.Facing)
		}
		base, encoded := button.Hash()
		key := [2]uint64{base, encoded}
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate button block hash for %+v", button)
		}
		seen[key] = struct{}{}
	}
	if len(buttonBlockNames) != buttonKindCount {
		t.Fatalf("button names = %d, want %d", len(buttonBlockNames), buttonKindCount)
	}
}

func TestButtonRedstoneOutputFaces(t *testing.T) {
	for _, facing := range cube.Faces() {
		button := Button{Kind: ButtonStone, Facing: facing, Powered: true}
		for _, face := range cube.Faces() {
			if got := button.RedstonePower(cube.Pos{}, nil, face); got != 15 {
				t.Errorf("weak output for button facing %v on face %v = %d, want 15", facing, face, got)
			}
			wantStrong := 0
			if face == facing.Opposite() {
				wantStrong = 15
			}
			if got := button.RedstoneStrongPower(cube.Pos{}, nil, face); got != wantStrong {
				t.Errorf("strong output for button facing %v on face %v = %d, want %d", facing, face, got, wantStrong)
			}
		}
	}
	if got := (Button{Kind: ButtonStone}).RedstonePower(cube.Pos{}, nil, cube.FaceUp); got != 0 {
		t.Fatalf("unpressed button output = %d, want 0", got)
	}
}

func TestButtonPulseDurations(t *testing.T) {
	tests := []struct {
		name string
		kind int
		want time.Duration
	}{
		{name: "stone", kind: ButtonStone, want: time.Second},
		{name: "polished blackstone", kind: ButtonPolishedBlackstone, want: time.Second},
		{name: "oak", kind: ButtonOak, want: 3 * time.Second / 2},
		{name: "crimson", kind: ButtonCrimson, want: 3 * time.Second / 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := (Button{Kind: test.kind}).pulseDuration(); got != test.want {
				t.Fatalf("button pulse duration = %s, want %s", got, test.want)
			}
		})
	}
}

func TestButtonActivationAndScheduledRelease(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	pos := cube.Pos{0, 0, 0}
	button := Button{Kind: ButtonStone, Facing: cube.FaceUp}
	if err := w.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, button, nil)
		if !button.Activate(pos, cube.FaceUp, tx, nil, nil) {
			t.Fatal("button activation returned false")
		}
		current, ok := tx.Block(pos).(Button)
		if !ok || !current.Powered {
			t.Fatalf("button after activation = %#v, want powered button", tx.Block(pos))
		}
		if button.Activate(pos, cube.FaceUp, tx, nil, nil) {
			t.Fatal("already pressed button was activated a second time")
		}
		current.ScheduledTick(pos, tx, nil)
		current, ok = tx.Block(pos).(Button)
		if !ok || current.Powered {
			t.Fatalf("button after scheduled release = %#v, want unpowered button", tx.Block(pos))
		}
	}).Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestButtonModelsHaveThinNonSolidBoxes(t *testing.T) {
	for _, facing := range cube.Faces() {
		for _, pressed := range []bool{false, true} {
			m := Button{Facing: facing, Powered: pressed}.Model()
			boxes := m.BBox(cube.Pos{}, nil)
			if len(boxes) != 1 {
				t.Fatalf("button model for facing %v has %d boxes, want 1", facing, len(boxes))
			}
			for _, face := range cube.Faces() {
				if m.FaceSolid(cube.Pos{}, face, nil) {
					t.Fatalf("button model reports solid face %v", face)
				}
			}
		}
	}
}
