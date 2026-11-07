package entity

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func runItemBehaviourTestWorld(w *world.World, f func(*world.Tx)) {
	w.Do(f).Wait(context.Background())
}

func TestItemEntityHopperCollectionAndLocking(t *testing.T) {
	tests := []struct {
		name            string
		powered         bool
		collectCooldown int64
		wantCollected   bool
	}{
		{name: "unpowered hopper collects item", wantCollected: true},
		{name: "powered hopper does not collect item", powered: true},
		{name: "hopper collection cooldown prevents collection", collectCooldown: 8},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := world.Config{Synchronous: true, Entities: DefaultRegistry}.New()
			defer w.Close()

			hopperPos := cube.Pos{0, 64, 0}
			var handle *world.EntityHandle
			var slots []item.Stack
			var cooldown int64
			runItemBehaviourTestWorld(w, func(tx *world.Tx) {
				hopper := block.NewHopper()
				hopper.Facing = cube.FaceDown
				hopper.Powered = test.powered
				hopper.CollectCooldown = test.collectCooldown
				tx.SetBlock(hopperPos, hopper, nil)

				handle = NewItem(world.EntitySpawnOpts{
					Position: mgl64.Vec3{0.5, 65.1, 0.5},
				}, item.NewStack(item.Stick{}, 5))
				ent := tx.AddEntity(handle).(*Ent)
				ent.Behaviour().(*ItemBehaviour).Tick(ent, tx)

				after := tx.Block(hopperPos).(block.Hopper)
				slots = after.Inventory(tx, hopperPos).Slots()
				cooldown = after.CollectCooldown
			})

			collected := len(slots) > 0 && slots[0].Count() == 5
			if collected != test.wantCollected {
				t.Fatalf("hopper collected item = %v, want %v (slots=%v)", collected, test.wantCollected, slots)
			}
			if test.wantCollected {
				if !handle.Closed() {
					t.Fatal("item entity remained active after its full stack was collected")
				}
				if cooldown != 8 {
					t.Fatalf("hopper collection cooldown = %d, want 8", cooldown)
				}
			} else if handle.Closed() {
				t.Fatal("item entity was removed while hopper collection was locked")
			}
		})
	}
}
