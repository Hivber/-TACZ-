package block

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func TestDaylightDetectorStates(t *testing.T) {
	states := allDaylightDetectors()
	if len(states) != 32 {
		t.Fatalf("daylight detector states = %d, want 32", len(states))
	}
	seen := make(map[[2]uint64]struct{}, len(states))
	for _, state := range states {
		detector := state.(DaylightDetector)
		name, properties := detector.EncodeBlock()
		wantName := "minecraft:daylight_detector"
		if detector.Inverted {
			wantName = "minecraft:daylight_detector_inverted"
		}
		if name != wantName {
			t.Fatalf("encoded block name = %q, want %q", name, wantName)
		}
		if got := properties["redstone_signal"]; got != int32(detector.Power) {
			t.Fatalf("encoded redstone_signal = %v, want %d", got, detector.Power)
		}
		if got, _ := detector.EncodeItem(); got != "minecraft:daylight_detector" {
			t.Fatalf("encoded item = %q, want minecraft:daylight_detector", got)
		}
		base, value := detector.Hash()
		key := [2]uint64{base, value}
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate daylight detector hash for %+v", detector)
		}
		seen[key] = struct{}{}
	}
}

func TestDaylightDetectorSignal(t *testing.T) {
	tests := []struct {
		name       string
		sky        uint8
		worldTime  int64
		raining    bool
		thundering bool
		inverted   bool
		dimension  world.Dimension
		want       int
	}{
		{name: "clear noon", sky: 15, worldTime: 6000, dimension: world.Overworld, want: 15},
		{name: "rainy noon", sky: 15, worldTime: 6000, raining: true, dimension: world.Overworld, want: 12},
		{name: "thunderstorm noon", sky: 15, worldTime: 6000, raining: true, thundering: true, dimension: world.Overworld, want: 10},
		{name: "clear midnight", sky: 15, worldTime: 18000, dimension: world.Overworld, want: 0},
		{name: "inverted clear noon", sky: 15, worldTime: 6000, inverted: true, dimension: world.Overworld, want: 0},
		{name: "inverted clear midnight", sky: 15, worldTime: 18000, inverted: true, dimension: world.Overworld, want: 11},
		{name: "inverted fully shaded", sky: 0, worldTime: 6000, inverted: true, dimension: world.Overworld, want: 15},
		{name: "daylight fully shaded", sky: 0, worldTime: 6000, dimension: world.Overworld, want: 0},
		{name: "inverted Nether", sky: 15, worldTime: 6000, inverted: true, dimension: world.Nether, want: 0},
		{name: "inverted End", sky: 15, worldTime: 6000, inverted: true, dimension: world.End, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := daylightDetectorSignalForDimension(test.sky, test.worldTime, test.raining, test.thundering, test.inverted, test.dimension)
			if got != test.want {
				t.Fatalf("signal = %d, want %d", got, test.want)
			}
		})
	}
}

func TestDaylightDetectorPowerFaces(t *testing.T) {
	detector := DaylightDetector{Power: 9}
	for _, face := range cube.Faces() {
		if got := detector.RedstonePower(cube.Pos{}, nil, face); got != 9 {
			t.Errorf("power on face %v = %d, want 9", face, got)
		}
	}
}

func TestDaylightDetectorModel(t *testing.T) {
	boxes := (DaylightDetector{}).Model().BBox(cube.Pos{}, nil)
	if len(boxes) != 1 {
		t.Fatalf("daylight detector collision boxes = %d, want 1", len(boxes))
	}
	if got := boxes[0].Max()[1]; got != 0.375 {
		t.Fatalf("daylight detector height = %v, want 0.375", got)
	}
}

func TestDaylightDetectorUpdateDelay(t *testing.T) {
	tests := []struct {
		tick int64
		want time.Duration
	}{
		{tick: 0, want: time.Second},
		{tick: 1, want: 19 * time.Second / 20},
		{tick: 19, want: time.Second / 20},
		{tick: 20, want: time.Second},
		{tick: 21, want: 19 * time.Second / 20},
	}
	for _, test := range tests {
		if got := daylightDetectorUpdateDelay(test.tick); got != test.want {
			t.Errorf("delay at tick %d = %v, want %v", test.tick, got, test.want)
		}
	}
}
