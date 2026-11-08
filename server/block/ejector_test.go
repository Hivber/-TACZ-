package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
)

func TestDispenserAndDropperStates(t *testing.T) {
	states := append(allDispensers(), allDroppers()...)
	if len(states) != 24 {
		t.Fatalf("state count = %d, want 24", len(states))
	}
	seen := make(map[[2]uint64]struct{}, len(states))
	for _, state := range states {
		name, properties := state.EncodeBlock()
		if name != "minecraft:dispenser" && name != "minecraft:dropper" {
			t.Fatalf("unexpected block %q", name)
		}
		if _, ok := properties["facing_direction"]; !ok {
			t.Fatalf("%s has no facing_direction", name)
		}
		if _, ok := properties["triggered_bit"]; !ok {
			t.Fatalf("%s has no triggered_bit", name)
		}
		base, value := state.Hash()
		key := [2]uint64{base, value}
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate hash for %s %+v", name, state)
		}
		seen[key] = struct{}{}
	}
}

func TestDispenserAndDropperItems(t *testing.T) {
	if name, meta := (Dispenser{}).EncodeItem(); name != "minecraft:dispenser" || meta != 0 {
		t.Fatalf("dispenser item = %s:%d", name, meta)
	}
	if name, meta := (Dropper{}).EncodeItem(); name != "minecraft:dropper" || meta != 0 {
		t.Fatalf("dropper item = %s:%d", name, meta)
	}
}

func TestEjectorInventoryAndNBT(t *testing.T) {
	dispenser := NewDispenser()
	dispenser.Facing, dispenser.Triggered, dispenser.CustomName = cube.FaceEast, true, "Test Dispenser"
	if _, err := dispenser.Inventory(nil, cube.Pos{}).AddItem(item.NewStack(item.Stick{}, 12)); err != nil {
		t.Fatal(err)
	}
	if dispenser.ContainerSize() != 9 {
		t.Fatalf("container size = %d, want 9", dispenser.ContainerSize())
	}
	data := dispenser.EncodeNBT()
	if data["id"] != "Dispenser" {
		t.Fatalf("dispenser NBT id = %v", data["id"])
	}
	decoded := (Dispenser{ejector: ejector{Facing: cube.FaceEast, Triggered: true}}).DecodeNBT(data).(Dispenser)
	if decoded.Facing != cube.FaceEast || !decoded.Triggered || decoded.CustomName != "Test Dispenser" {
		t.Fatalf("decoded state = %+v", decoded)
	}
	contents := decoded.Inventory(nil, cube.Pos{}).Items()
	if len(contents) != 1 || contents[0].Count() != 12 {
		t.Fatalf("decoded inventory = %v, want 12 sticks", contents)
	}

	dropper := NewDropper()
	dropper.Facing = cube.FaceDown
	dropperData := dropper.EncodeNBT()
	if dropperData["id"] != "Dropper" {
		t.Fatalf("dropper NBT id = %v", dropperData["id"])
	}
	decodedDropper := (Dropper{ejector: ejector{Facing: cube.FaceDown}}).DecodeNBT(dropperData).(Dropper)
	if decodedDropper.Facing != cube.FaceDown {
		t.Fatalf("dropper facing = %v, want down", decodedDropper.Facing)
	}
}

func TestEjectorRedstonePowerEdges(t *testing.T) {
	dispenser := NewDispenser()
	after, changed := dispenser.RedstonePowerUpdate(cube.Pos{}, nil, 15)
	if !changed || !after.(Dispenser).Triggered {
		t.Fatal("dispenser did not latch on rising power")
	}
	after, changed = after.(Dispenser).RedstonePowerUpdate(cube.Pos{}, nil, 15)
	if changed || !after.(Dispenser).Triggered {
		t.Fatal("steady power retriggered dispenser")
	}
	after, changed = after.(Dispenser).RedstonePowerUpdate(cube.Pos{}, nil, 0)
	if !changed || after.(Dispenser).Triggered {
		t.Fatal("dispenser did not reset after power loss")
	}
	dropper := NewDropper()
	afterDropper, changed := dropper.RedstonePowerUpdate(cube.Pos{}, nil, 15)
	if !changed || !afterDropper.(Dropper).Triggered {
		t.Fatal("dropper did not latch on rising power")
	}
}

func TestEjectorRandomSlotAndInsertion(t *testing.T) {
	if got := randomEjectorSlot(nil); got != -1 {
		t.Fatalf("empty slot = %d, want -1", got)
	}
	slots := []item.Stack{{}, item.NewStack(item.Stick{}, 3), {}, item.NewStack(item.Stick{}, 1)}
	for range 100 {
		got := randomEjectorSlot(slots)
		if got != 1 && got != 3 {
			t.Fatalf("random slot = %d, not a non-empty slot", got)
		}
	}
	destination := inventory.New(1, nil)
	if !insertEjectorItem(destination, item.NewStack(item.Stick{}, 1)) {
		t.Fatal("insertion into an empty inventory failed")
	}
	if _, err := destination.AddItem(item.NewStack(item.Stick{}, 63)); err != nil {
		t.Fatal(err)
	}
	if insertEjectorItem(destination, item.NewStack(item.Stick{}, 1)) {
		t.Fatal("insertion into a full inventory succeeded")
	}
}

func TestEjectorFacingVectors(t *testing.T) {
	tests := []struct {
		face cube.Face
		want [3]float64
	}{
		{cube.FaceDown, [3]float64{0, -1, 0}},
		{cube.FaceUp, [3]float64{0, 1, 0}},
		{cube.FaceNorth, [3]float64{0, 0, -1}},
		{cube.FaceSouth, [3]float64{0, 0, 1}},
		{cube.FaceWest, [3]float64{-1, 0, 0}},
		{cube.FaceEast, [3]float64{1, 0, 0}},
	}
	for _, test := range tests {
		got := ejectorFaceVector(test.face)
		for axis := range 3 {
			if got[axis] != test.want[axis] {
				t.Errorf("face %v axis %d = %v, want %v", test.face, axis, got[axis], test.want[axis])
			}
		}
	}
}

func TestDispenserSpawnEggBehaviourFallback(t *testing.T) {
	behaviour := func(*world.Tx, world.EntitySpawnOpts, item.Stack) bool { return true }
	RegisterDispenserItemBehavior("minecraft:spawn_egg", behaviour)
	defer func() {
		dispenserItemBehaviorRegistry.Lock()
		delete(dispenserItemBehaviorRegistry.behaviours, "minecraft:spawn_egg")
		dispenserItemBehaviorRegistry.Unlock()
	}()
	if got := lookupDispenserItemBehavior("minecraft:pig_spawn_egg"); got == nil {
		t.Fatal("specific spawn egg did not resolve the generic spawn egg behaviour")
	}
	if got := lookupDispenserItemBehavior("minecraft:unregistered_test_item"); got != nil {
		t.Fatal("unregistered item unexpectedly resolved a dispenser behaviour")
	}
}
