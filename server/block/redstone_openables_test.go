package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func TestRedstoneOpensAndClosesExistingMechanisms(t *testing.T) {
	tests := []struct {
		name   string
		block  world.Block
		update func(world.Block, int) (world.Block, bool)
		isOpen func(world.Block) bool
	}{
		{
			name:  "wood door",
			block: WoodDoor{},
			update: func(b world.Block, power int) (world.Block, bool) {
				return b.(WoodDoor).RedstonePowerUpdate(cube.Pos{}, nil, power)
			},
			isOpen: func(b world.Block) bool { return b.(WoodDoor).Open },
		},
		{
			name:  "copper door",
			block: CopperDoor{},
			update: func(b world.Block, power int) (world.Block, bool) {
				return b.(CopperDoor).RedstonePowerUpdate(cube.Pos{}, nil, power)
			},
			isOpen: func(b world.Block) bool { return b.(CopperDoor).Open },
		},
		{
			name:  "wood trapdoor",
			block: WoodTrapdoor{},
			update: func(b world.Block, power int) (world.Block, bool) {
				return b.(WoodTrapdoor).RedstonePowerUpdate(cube.Pos{}, nil, power)
			},
			isOpen: func(b world.Block) bool { return b.(WoodTrapdoor).Open },
		},
		{
			name:  "copper trapdoor",
			block: CopperTrapdoor{},
			update: func(b world.Block, power int) (world.Block, bool) {
				return b.(CopperTrapdoor).RedstonePowerUpdate(cube.Pos{}, nil, power)
			},
			isOpen: func(b world.Block) bool { return b.(CopperTrapdoor).Open },
		},
		{
			name:  "wood fence gate",
			block: WoodFenceGate{},
			update: func(b world.Block, power int) (world.Block, bool) {
				return b.(WoodFenceGate).RedstonePowerUpdate(cube.Pos{}, nil, power)
			},
			isOpen: func(b world.Block) bool { return b.(WoodFenceGate).Open },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opened, changed := test.update(test.block, 1)
			if !changed || !test.isOpen(opened) {
				t.Fatalf("powering did not open the block: changed=%v block=%#v", changed, opened)
			}
			stillOpen, changed := test.update(opened, 15)
			if changed || !test.isOpen(stillOpen) {
				t.Fatalf("steady power changed the open state: changed=%v block=%#v", changed, stillOpen)
			}
			closed, changed := test.update(stillOpen, 0)
			if !changed || test.isOpen(closed) {
				t.Fatalf("removing power did not close the block: changed=%v block=%#v", changed, closed)
			}
		})
	}
}
