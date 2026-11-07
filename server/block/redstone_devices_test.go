package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world/sound"
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
		if properties["minecraft:cardinal_direction"] != repeater.Direction.String() {
			t.Fatalf("repeater cardinal direction = %v, want output-facing %s", properties["minecraft:cardinal_direction"], repeater.Direction)
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

func TestComparatorInventorySignal(t *testing.T) {
	inv := inventory.New(9, nil)
	for slot := 0; slot < 5; slot++ {
		if err := inv.SetItem(slot, item.NewStack(item.Stick{}, 64)); err != nil {
			t.Fatal(err)
		}
	}
	if got := comparatorContainerSignal(inv); got != 8 {
		t.Fatalf("half-filled inventory comparator signal = %d, want 8", got)
	}
	if got := comparatorContainerSignal(inventory.New(9, nil)); got != 0 {
		t.Fatalf("empty inventory comparator signal = %d, want 0", got)
	}
	partial := inventory.New(9, nil)
	if err := partial.SetItem(0, item.NewStack(item.Stick{}, 32)); err != nil {
		t.Fatal(err)
	}
	if got := comparatorContainerSignal(partial); got != 1 {
		t.Fatalf("partially filled single slot comparator signal = %d, want 1", got)
	}
}

func TestJukeboxComparatorSignal(t *testing.T) {
	want := map[string]int{
		"13": 1, "cat": 2, "blocks": 3, "chirp": 4, "far": 5,
		"mall": 6, "mellohi": 7, "stal": 8, "strad": 9, "lava_chicken": 9,
		"ward": 10, "tears": 10, "11": 11, "creator_music_box": 11,
		"wait": 12, "creator": 12, "pigstep": 13, "precipice": 13,
		"otherside": 14, "relic": 14, "5": 15,
	}
	if got := (Jukebox{}).ComparatorOutput(cube.Pos{}, nil); got != 0 {
		t.Fatalf("empty jukebox comparator signal = %d, want 0", got)
	}
	for _, disc := range sound.MusicDiscs() {
		name := disc.String()
		expected, ok := want[name]
		if !ok {
			t.Fatalf("missing expected comparator strength for music disc %q", name)
		}
		jukebox := Jukebox{Item: item.NewStack(item.MusicDisc{DiscType: disc}, 1)}
		if got := jukebox.ComparatorOutput(cube.Pos{}, nil); got != expected {
			t.Errorf("jukebox comparator signal for %q = %d, want %d", name, got, expected)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("comparator test has unmapped music discs: %v", want)
	}
}

func TestLecternBedrockComparatorOutput(t *testing.T) {
	tests := []struct {
		name     string
		pages    int
		page     int
		written  bool
		want     int
	}{
		{name: "empty lectern", want: 0},
		{name: "one page", pages: 1, want: 15},
		{name: "two pages", pages: 2, page: 1, want: 15},
		{name: "three pages", pages: 3, page: 2, want: 15},
		{name: "four pages first spread", pages: 4, page: 0, want: 1},
		{name: "four pages second spread", pages: 4, page: 2, want: 15},
		{name: "eight pages first spread", pages: 8, page: 0, want: 1},
		{name: "eight pages second spread", pages: 8, page: 2, want: 5},
		{name: "eight pages third spread", pages: 8, page: 4, want: 10},
		{name: "eight pages final spread", pages: 8, page: 6, want: 15},
		{name: "nine pages final full spread", pages: 9, page: 6, want: 15},
		{name: "nine pages trailing odd page", pages: 9, page: 8, want: 15},
		{name: "written book", pages: 8, page: 4, written: true, want: 10},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lectern := Lectern{Page: test.page}
			if test.pages > 0 {
				pages := make([]string, test.pages)
				if test.written {
					lectern.Book = item.NewStack(item.WrittenBook{Pages: pages}, 1)
				} else {
					lectern.Book = item.NewStack(item.BookAndQuill{Pages: pages}, 1)
				}
			}
			if got := lectern.ComparatorOutput(cube.Pos{}, nil); got != test.want {
				t.Fatalf("lectern comparator output = %d, want %d", got, test.want)
			}
		})
	}
}

func TestComparatorModes(t *testing.T) {
	tests := []struct {
		input, side int
		subtract    bool
		want        int
	}{
		{input: 12, side: 5, want: 12},
		{input: 4, side: 5, want: 0},
		{input: 12, side: 5, subtract: true, want: 7},
		{input: 4, side: 5, subtract: true, want: 0},
		{input: 15, side: 15, want: 15},
	}
	for _, test := range tests {
		if got := comparatorOutput(test.input, test.side, test.subtract); got != test.want {
			t.Errorf("comparatorOutput(%d,%d,%v) = %d, want %d", test.input, test.side, test.subtract, got, test.want)
		}
	}
}

func TestComparatorStatesAndNBT(t *testing.T) {
	states := allRedstoneComparators()
	if len(states) != 16 {
		t.Fatalf("comparator states = %d, want 16", len(states))
	}
	seen := make(map[[2]uint64]struct{}, len(states))
	for _, state := range states {
		comparator := state.(RedstoneComparator)
		name, properties := comparator.EncodeBlock()
		if name != "minecraft:unpowered_comparator" && name != "minecraft:powered_comparator" {
			t.Fatalf("unexpected comparator block name %q", name)
		}
		if properties["minecraft:cardinal_direction"] != comparator.Direction.String() ||
			properties["output_subtract_bit"] != comparator.Subtract ||
			properties["output_lit_bit"] != comparator.Powered {
			t.Fatalf("comparator properties do not match state: %+v -> %v", comparator, properties)
		}
		base, encodedState := comparator.Hash()
		key := [2]uint64{base, encodedState}
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate comparator state hash: %+v", comparator)
		}
		seen[key] = struct{}{}
	}
	original := RedstoneComparator{Direction: cube.East, Subtract: true, Powered: true, output: 13}
	decoded := original.DecodeNBT(original.EncodeNBT()).(RedstoneComparator)
	if decoded.output != 13 || !decoded.Powered || !decoded.Subtract || decoded.Direction != cube.East {
		t.Fatalf("comparator NBT round trip mismatch: %+v", decoded)
	}
}
