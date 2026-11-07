package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.RedstonePowerSource        = RedstoneRepeater{}
	_ world.RedstoneStrongPowerSource  = RedstoneRepeater{}
	_ world.RedstonePowerContextAction = RedstoneRepeater{}
	_ world.ScheduledTicker            = RedstoneRepeater{}
	_ world.NeighbourUpdateTicker      = RedstoneRepeater{}
)

// RedstoneRepeater is a directional redstone component. It reads power from
// behind, emits a delayed signal in front, and locks when a powered repeater
// or comparator points into either side.
type RedstoneRepeater struct {
	empty

	// Direction is the horizontal direction in which the repeater outputs.
	Direction cube.Direction
	// Delay is the signal delay in redstone ticks, from 1 through 4.
	Delay int
	// Powered indicates whether the repeater currently emits power.
	Powered bool
}

type repeaterModel struct{}

func (repeaterModel) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{cube.Box(0, 0, 0, 1, 0.125, 1)}
}

func (repeaterModel) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool { return false }

// Model returns the thin collision model of a repeater.
func (RedstoneRepeater) Model() world.BlockModel { return repeaterModel{} }

// RedstonePower emits a full signal only towards the output face.
func (r RedstoneRepeater) RedstonePower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if r.Powered && face == r.Direction.Face() {
		return 15
	}
	return 0
}

// RedstoneStrongPower strongly powers the block directly in front of the repeater.
func (r RedstoneRepeater) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	return r.RedstonePower(pos, tx, face)
}

func (r RedstoneRepeater) RedstoneOutputDirection() cube.Direction { return r.Direction }
func (r RedstoneRepeater) RedstoneOutputPowered() bool             { return r.Powered }

type repeaterLockSource interface {
	RedstoneOutputDirection() cube.Direction
	RedstoneOutputPowered() bool
}

// locked reports whether a powered repeater/comparator on either side points
// into this repeater. Side redstone dust does not lock a repeater.
func (r RedstoneRepeater) locked(pos cube.Pos, tx *world.Tx) bool {
	for _, side := range []cube.Direction{r.Direction.RotateLeft(), r.Direction.RotateRight()} {
		neighbourPos := pos.Side(side.Face())
		if diode, ok := tx.Block(neighbourPos).(repeaterLockSource); ok &&
			diode.RedstoneOutputPowered() && diode.RedstoneOutputDirection() == side.Opposite() {
			return true
		}
	}
	return false
}

// inputPowered reports whether the block behind the repeater provides power.
func (r RedstoneRepeater) inputPowered(pos cube.Pos, tx *world.Tx) bool {
	return tx.RedstonePowerFrom(pos, r.Direction.Opposite().Face()) > 0
}

// RedstonePowerActionUpdate schedules a delayed state change when the rear
// input changes. Context actions also let side-lock changes be evaluated when
// the total power level at the repeater remains unchanged.
func (r RedstoneRepeater) RedstonePowerActionUpdate(pos cube.Pos, tx *world.Tx, _ world.RedstoneUpdate) {
	if r.locked(pos, tx) || r.inputPowered(pos, tx) == r.Powered {
		return
	}
	tx.ScheduleBlockUpdate(pos, r, redstoneTicks(r.effectiveDelay()))
}

func (r RedstoneRepeater) effectiveDelay() int {
	return min(max(r.Delay, 1), 4)
}

// NeighbourUpdateTick validates the support block and queues a redstone
// reevaluation after neighbouring components change.
func (r RedstoneRepeater) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	below := pos.Side(cube.FaceDown)
	if !tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx) {
		breakBlock(r, pos, tx)
		return
	}
	tx.Redstone().ScheduleUpdate(pos)
}

// ScheduledTick commits the current rear-input state after the repeater delay.
func (r RedstoneRepeater) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(RedstoneRepeater)
	if !ok || current.locked(pos, tx) {
		return
	}
	powered := current.inputPowered(pos, tx)
	if powered == current.Powered {
		return
	}
	current.Powered = powered
	tx.SetBlock(pos, current, nil)
	if powered {
		tx.PlaySound(pos.Vec3Centre(), sound.PowerOn{})
	} else {
		tx.PlaySound(pos.Vec3Centre(), sound.PowerOff{})
	}
}

// UseOnBlock places a repeater facing away from the player and on a solid top face.
func (r RedstoneRepeater) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, face, used := firstReplaceable(tx, pos, face, r)
	if !used || face != cube.FaceUp {
		return false
	}
	below := pos.Side(cube.FaceDown)
	if !tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx) {
		return false
	}
	r.Direction = user.Rotation().Direction()
	r.Delay, r.Powered = 1, false
	place(tx, pos, r, user, ctx)
	if placed(ctx) {
		tx.Redstone().ScheduleUpdate(pos)
		return true
	}
	return false
}

// Activate cycles the repeater delay from one to four redstone ticks.
func (r RedstoneRepeater) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User, _ *item.UseContext) bool {
	r.Delay = r.effectiveDelay()%4 + 1
	tx.SetBlock(pos, r, nil)
	return true
}

func (r RedstoneRepeater) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(RedstoneRepeater{}))
}

func (RedstoneRepeater) EncodeItem() (string, int16) { return "minecraft:repeater", 0 }

func (r RedstoneRepeater) EncodeBlock() (string, map[string]any) {
	name := "minecraft:unpowered_repeater"
	if r.Powered {
		name = "minecraft:powered_repeater"
	}
	return name, map[string]any{
		"minecraft:cardinal_direction": r.Direction.Opposite().String(),
		"repeater_delay":               int32(r.effectiveDelay() - 1),
	}
}

func allRedstoneRepeaters() (all []world.Block) {
	for _, direction := range cube.Directions() {
		for delay := 1; delay <= 4; delay++ {
			all = append(all,
				RedstoneRepeater{Direction: direction, Delay: delay},
				RedstoneRepeater{Direction: direction, Delay: delay, Powered: true},
			)
		}
	}
	return
}
