package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func TestRedstoneSignalSynchronisesBothDoorHalves(t *testing.T) {
	tests := []struct {
		name   string
		bottom world.Block
		top    world.Block
		isOpen func(world.Block) bool
	}{
		{
			name:   "wood door",
			bottom: WoodDoor{Facing: cube.North},
			top:    WoodDoor{Facing: cube.North, Top: true},
			isOpen: func(b world.Block) bool { return b.(WoodDoor).Open },
		},
		{
			name:   "copper door",
			bottom: CopperDoor{Facing: cube.North},
			top:    CopperDoor{Facing: cube.North, Top: true},
			isOpen: func(b world.Block) bool { return b.(CopperDoor).Open },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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

			bottomPos := cube.Pos{1, 64, 1}
			topPos := bottomPos.Side(cube.FaceUp)
			sourcePos := bottomPos.Side(cube.FaceWest)
			runWorld(w, func(tx *world.Tx) {
				tx.SetBlock(bottomPos.Side(cube.FaceDown), Stone{}, nil)
				tx.SetBlock(bottomPos, test.bottom, nil)
				tx.SetBlock(topPos, test.top, nil)
				tx.SetBlock(sourcePos, RedstoneBlock{}, nil)
			})
			w.AdvanceTick()

			var bottom, top world.Block
			runWorld(w, func(tx *world.Tx) {
				bottom, top = tx.Block(bottomPos), tx.Block(topPos)
			})
			if !test.isOpen(bottom) || !test.isOpen(top) {
				t.Fatalf("powered door halves are not both open: bottom=%#v top=%#v", bottom, top)
			}

			runWorld(w, func(tx *world.Tx) {
				tx.SetBlock(sourcePos, nil, nil)
			})
			w.AdvanceTick()
			runWorld(w, func(tx *world.Tx) {
				bottom, top = tx.Block(bottomPos), tx.Block(topPos)
			})
			if test.isOpen(bottom) || test.isOpen(top) {
				t.Fatalf("unpowered door halves are not both closed: bottom=%#v top=%#v", bottom, top)
			}
		})
	}
}

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
