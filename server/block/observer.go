package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.RedstonePowerSource       = Observer{}
	_ world.RedstoneStrongPowerSource = Observer{}
	_ world.ScheduledTicker           = Observer{}
	_ world.NeighbourUpdateTicker     = Observer{}
)

// Observer watches the block on its facing side. A change there causes a
// two-redstone-tick pulse from the opposite face.
type Observer struct {
	solid

	// Facing is the side of the observer that monitors neighbouring block changes.
	Facing cube.Face
	// Powered indicates that the observer is in the active part of its pulse.
	Powered bool
}

// RedstonePower emits power from the back of the observer while its pulse is active.
func (o Observer) RedstonePower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if o.Powered && face == o.Facing.Opposite() {
		return 15
	}
	return 0
}

// RedstoneStrongPower strongly powers the block directly behind the observer.
func (o Observer) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	return o.RedstonePower(pos, tx, face)
}

// NeighbourUpdateTick schedules a pulse only for changes on the monitored face.
func (o Observer) NeighbourUpdateTick(pos, changedNeighbour cube.Pos, tx *world.Tx) {
	if changedNeighbour != pos.Side(o.Facing) || o.Powered {
		return
	}
	tx.ScheduleBlockUpdate(pos, o, redstoneTicks(1))
}

// ScheduledTick starts the pulse or turns it off after two redstone ticks.
func (o Observer) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(Observer)
	if !ok {
		return
	}
	if current.Powered {
		current.Powered = false
		tx.SetBlock(pos, current, nil)
		return
	}
	current.Powered = true
	tx.SetBlock(pos, current, nil)
	tx.ScheduleBlockUpdate(pos, current, redstoneTicks(2))
}

// UseOnBlock places the observing face towards the block that was clicked.
func (o Observer) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, face, used := firstReplaceable(tx, pos, face, o)
	if !used {
		return false
	}
	o.Facing = face.Opposite()
	o.Powered = false
	place(tx, pos, o, user, ctx)
	return placed(ctx)
}

func (o Observer) BreakInfo() BreakInfo {
	return newBreakInfo(3.5, pickaxeHarvestable, pickaxeEffective, oneOf(Observer{}))
}

func (Observer) EncodeItem() (string, int16) { return "minecraft:observer", 0 }

func (o Observer) EncodeBlock() (string, map[string]any) {
	return "minecraft:observer", map[string]any{
		"minecraft:facing_direction": o.Facing.String(),
		"powered_bit":                o.Powered,
	}
}

func allObservers() (all []world.Block) {
	for _, face := range cube.Faces() {
		all = append(all, Observer{Facing: face}, Observer{Facing: face, Powered: true})
	}
	return
}
