package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestHopperRedstonePowerUpdate(t *testing.T) {
	tests := []struct {
		name        string
		initial     bool
		power       int
		wantPowered bool
		wantChanged bool
	}{
		{name: "unpowered remains unlocked", initial: false, power: 0, wantPowered: false, wantChanged: false},
		{name: "weak signal locks hopper", initial: false, power: 1, wantPowered: true, wantChanged: true},
		{name: "full signal locks hopper", initial: false, power: 15, wantPowered: true, wantChanged: true},
		{name: "powered remains locked", initial: true, power: 15, wantPowered: true, wantChanged: false},
		{name: "signal removal unlocks hopper", initial: true, power: 0, wantPowered: false, wantChanged: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := Hopper{Powered: test.initial}
			afterBlock, changed := before.RedstonePowerUpdate(cube.Pos{}, nil, test.power)
			after, ok := afterBlock.(Hopper)
			if !ok {
				t.Fatalf("RedstonePowerUpdate returned %T, want Hopper", afterBlock)
			}
			if after.Powered != test.wantPowered {
				t.Errorf("Powered = %v, want %v", after.Powered, test.wantPowered)
			}
			if changed != test.wantChanged {
				t.Errorf("changed = %v, want %v", changed, test.wantChanged)
			}
		})
	}
}
