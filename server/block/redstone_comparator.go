package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.RedstonePowerSource   = RedstoneComparator{}
	_ world.RedstoneStrongPowerSource = RedstoneComparator{}
	_ world.ScheduledTicker       = RedstoneComparator{}
	_ world.NeighbourUpdateTicker = RedstoneComparator{}
	_ world.TickerBlock           = RedstoneComparator{}
)

// RedstoneComparator compares the redstone input at its rear with the strongest
// side input, or subtracts the side input from the rear input. Its output signal
// is block-entity data because the numeric value is not a Bedrock block state.
type RedstoneComparator struct {
	empty

	// Direction is the horizontal direction in which the comparator outputs.
	Direction cube.Direction
	// Subtract selects subtract mode instead of compare mode.
	Subtract bool
	// Powered selects the powered_comparator state and lights the comparator.
	Powered bool

	output int
}

// comparatorInputOverride is implemented by blocks that expose a direct
// comparator output instead of relying on container fullness.
type comparatorInputOverride interface {
	ComparatorOutput(pos cube.Pos, tx *world.Tx) int
}

// RedstonePower emits the saved comparator signal towards the output face.
func (c RedstoneComparator) RedstonePower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if c.Powered && face == c.Direction.Face() {
		return world.ClampRedstonePower(c.output)
	}
	return 0
}

// RedstoneStrongPower strongly powers the block directly in front of the comparator.
func (c RedstoneComparator) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	return c.RedstonePower(pos, tx, face)
}

func (c RedstoneComparator) RedstoneOutputDirection() cube.Direction { return c.Direction }
func (c RedstoneComparator) RedstoneOutputPowered() bool             { return c.Powered }

// comparatorContainerSignal returns the standard 0-15 inventory fullness signal.
func comparatorContainerSignal(inv *inventory.Inventory) int {
	if inv == nil {
		return 0
	}
	slots := inv.Slots()
	if len(slots) == 0 {
		return 0
	}
	fullness := 0.0
	for _, stack := range slots {
		if stack.Empty() {
			continue
		}
		maxCount := stack.MaxCount()
		if maxCount <= 0 {
			continue
		}
		fullness += float64(stack.Count()) / float64(maxCount)
	}
	fullness /= float64(len(slots))
	if fullness <= 0 {
		return 0
	}
	return min(15, int(fullness*14)+1)
}

func comparatorBlockInput(pos cube.Pos, tx *world.Tx) (int, bool) {
	b := tx.Block(pos)
	if override, ok := b.(comparatorInputOverride); ok {
		return world.ClampRedstonePower(override.ComparatorOutput(pos, tx)), true
	}
	if container, ok := b.(Container); ok {
		return comparatorContainerSignal(container.Inventory(tx, pos)), true
	}
	return 0, false
}

// comparatorFullCube checks whether a block has all six solid faces.
func comparatorFullCube(pos cube.Pos, b world.Block, tx *world.Tx) bool {
	for _, face := range cube.Faces() {
		if !b.Model().FaceSolid(pos, face, tx) {
			return false
		}
	}
	return true
}

// inputStrength reads a direct comparator override/container first, then
// ordinary redstone power. A full opaque block may relay a comparator override
// from the block entity immediately behind it, matching vanilla comparator
// container-reading behaviour.
func (c RedstoneComparator) inputStrength(pos cube.Pos, tx *world.Tx) int {
	inputFace := c.Direction.Opposite().Face()
	inputPos := pos.Side(inputFace)
	if signal, ok := comparatorBlockInput(inputPos, tx); ok {
		return signal
	}
	power := tx.RedstonePowerFrom(pos, inputFace)
	inputBlock := tx.Block(inputPos)
	if power < 15 && comparatorFullCube(inputPos, inputBlock, tx) {
		if signal, ok := comparatorBlockInput(inputPos.Side(inputFace), tx); ok {
			return signal
		}
	}
	return world.ClampRedstonePower(power)
}

func (c RedstoneComparator) sideStrength(pos cube.Pos, tx *world.Tx) int {
	left := pos.Side(c.Direction.RotateLeft().Face())
	right := pos.Side(c.Direction.RotateRight().Face())
	return max(tx.RedstonePowerFrom(pos, c.Direction.RotateLeft().Face()), tx.RedstonePowerFrom(pos, c.Direction.RotateRight().Face()))
}

func comparatorOutput(input, side int, subtract bool) int {
	input, side = world.ClampRedstonePower(input), world.ClampRedstonePower(side)
	if subtract {
		return max(input-side, 0)
	}
	if input < side {
		return 0
	}
	return input
}

func (c RedstoneComparator) calculatedOutput(pos cube.Pos, tx *world.Tx) int {
	return comparatorOutput(c.inputStrength(pos, tx), c.sideStrength(pos, tx), c.Subtract)
}

// Tick watches the rear and side inputs. A changed output is applied after the
// comparator's two-game-tick update delay using the block's scheduled ticker.
func (c RedstoneComparator) Tick(_ int64, pos cube.Pos, tx *world.Tx) {
	if current, ok := tx.Block(pos).(RedstoneComparator); ok && current.calculatedOutput(pos, tx) != current.output {
		tx.ScheduleBlockUpdate(pos, current, redstoneTicks(1))
	}
}

// NeighbourUpdateTick validates support and schedules the delayed comparator update.
func (c RedstoneComparator) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	below := pos.Side(cube.FaceDown)
	if !tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx) {
		breakBlock(c, pos, tx)
		return
	}
	tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
}

// ScheduledTick recalculates and commits the numeric output signal.
func (c RedstoneComparator) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(RedstoneComparator)
	if !ok {
		return
	}
	output := current.calculatedOutput(pos, tx)
	if output == current.output {
		return
	}
	previouslyPowered := current.Powered
	current.output = output
	current.Powered = output > 0
	if previouslyPowered != current.Powered {
		tx.SetBlock(pos, current, nil)
	} else {
		tx.SetBlockEntity(pos, current)
	}
	tx.Redstone().ScheduleUpdate(pos)
}

// Activate toggles between compare and subtract mode.
func (c RedstoneComparator) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User, _ *item.UseContext) bool {
	c.Subtract = !c.Subtract
	tx.SetBlock(pos, c, nil)
	tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
	return true
}

// UseOnBlock places a comparator facing away from the player on a solid block top.
func (c RedstoneComparator) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, face, used := firstReplaceable(tx, pos, face, c)
	if !used || face != cube.FaceUp {
		return false
	}
	below := pos.Side(cube.FaceDown)
	if !tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx) {
		return false
	}
	c.Direction = user.Rotation().Direction().Opposite()
	c.Subtract, c.Powered, c.output = false, false, 0
	place(tx, pos, c, user, ctx)
	if placed(ctx) {
		tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
		return true
	}
	return false
}

func (RedstoneComparator) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(RedstoneComparator{}))
}

func (RedstoneComparator) EncodeItem() (string, int16) { return "minecraft:comparator", 0 }

func (c RedstoneComparator) EncodeBlock() (string, map[string]any) {
	name := "minecraft:unpowered_comparator"
	if c.Powered {
		name = "minecraft:powered_comparator"
	}
	return name, map[string]any{
		"minecraft:cardinal_direction": c.Direction.String(),
		"output_subtract_bit":          c.Subtract,
		"output_lit_bit":               c.Powered,
	}
}

// DecodeNBT restores the comparator's signal value, which is not part of its
// block state palette.
func (c RedstoneComparator) DecodeNBT(data map[string]any) any {
	c.output = int(nbtconv.Int32(data, "OutputSignal"))
	c.output = world.ClampRedstonePower(c.output)
	c.Powered = c.output > 0
	return c
}

// EncodeNBT persists the numeric output so comparators retain container signals
// after chunks are saved and loaded.
func (c RedstoneComparator) EncodeNBT() map[string]any {
	return map[string]any{"OutputSignal": int32(world.ClampRedstonePower(c.output))}
}

func allRedstoneComparators() (all []world.Block) {
	for _, direction := range cube.Directions() {
		for _, subtract := range []bool{false, true} {
			all = append(all,
				RedstoneComparator{Direction: direction, Subtract: subtract},
				RedstoneComparator{Direction: direction, Subtract: subtract, Powered: true, output: 1},
			)
		}
	}
	return
}
