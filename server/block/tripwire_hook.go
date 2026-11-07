package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

const maxTripwireCircuitLength = 42

// TripwireHook is a wall-mounted redstone component that connects to another
// hook through a line of String blocks.
type TripwireHook struct {
	empty
	transparent
	flowingWaterDisplacer

	// Facing is the horizontal direction in which the hook points away from its support.
	Facing cube.Direction
	// Attached reports whether a valid tripwire line connects this hook to another hook.
	Attached bool
	// Powered reports whether an entity currently activates the connected line.
	Powered bool
}

var (
	_ world.RedstonePowerSource       = TripwireHook{}
	_ world.RedstoneStrongPowerSource = TripwireHook{}
	_ world.RedstoneWeakBlockPowerer  = TripwireHook{}
	_ world.NeighbourUpdateTicker     = TripwireHook{}
)

// RedstonePower emits weak power while a connected tripwire is activated.
func (h TripwireHook) RedstonePower(cube.Pos, *world.Tx, cube.Face) int {
	if h.Powered {
		return 15
	}
	return 0
}

// RedstoneStrongPower strongly powers the block supporting the hook.
func (h TripwireHook) RedstoneStrongPower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if h.Powered && face == h.Facing.Opposite().Face() {
		return 15
	}
	return 0
}

// RedstoneWeaklyPowersBlocks reports whether the hook currently emits redstone power.
func (h TripwireHook) RedstoneWeaklyPowersBlocks() bool {
	return h.Powered
}

// NeighbourUpdateTick validates the support block and refreshes the tripwire circuit.
func (h TripwireHook) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	support := pos.Side(h.Facing.Opposite().Face())
	if !tx.Block(support).Model().FaceSolid(support, h.Facing.Face(), tx) {
		breakBlock(h, pos, tx)
		return
	}
	updateTripwireHookLine(pos, h, tx)
}

// UseOnBlock places the hook on a horizontal face of a supporting block.
func (h TripwireHook) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	if face.Axis() == cube.Y {
		return false
	}
	pos, face, used := firstReplaceable(tx, pos, face, h)
	if !used || face.Axis() == cube.Y {
		return false
	}
	support := pos.Side(face.Opposite())
	if !tx.Block(support).Model().FaceSolid(support, face, tx) {
		return false
	}
	h.Facing = face.Direction()
	h.Attached, h.Powered = false, false
	place(tx, pos, h, user, ctx)
	updateTripwireHookLine(pos, h, tx)
	return placed(ctx)
}

// BreakInfo returns the hardness and drops of the tripwire hook. Removing a
// hook also detaches the line and updates its remaining endpoint.
func (h TripwireHook) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(TripwireHook{})).
		withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
			updateTripwireLinesForWire(pos, tx)
		})
}

// EncodeItem returns the item identifier of the tripwire hook.
func (TripwireHook) EncodeItem() (string, int16) {
	return "minecraft:tripwire_hook", 0
}

// EncodeBlock returns the Bedrock identifier and state properties of the hook.
func (h TripwireHook) EncodeBlock() (string, map[string]any) {
	return "minecraft:tripwire_hook", map[string]any{
		"direction":    tripwireHookBedrockDirection(h.Facing),
		"attached_bit": h.Attached,
		"powered_bit":  h.Powered,
	}
}

func tripwireHookBedrockDirection(direction cube.Direction) int32 {
	switch direction {
	case cube.South:
		return 0
	case cube.West:
		return 1
	case cube.North:
		return 2
	case cube.East:
		return 3
	default:
		return 0
	}
}

func tripwireOffset(direction cube.Direction, n int) cube.Pos {
	switch direction {
	case cube.North:
		return cube.Pos{0, 0, -n}
	case cube.South:
		return cube.Pos{0, 0, n}
	case cube.West:
		return cube.Pos{-n, 0, 0}
	case cube.East:
		return cube.Pos{n, 0, 0}
	default:
		return cube.Pos{}
	}
}

// updateTripwireHookLine recomputes the line starting at pos, including its
// opposite endpoint. Gaps and disarmed segments invalidate the circuit.
func updateTripwireHookLine(pos cube.Pos, hook TripwireHook, tx *world.Tx) {
	current, ok := tx.Block(pos).(TripwireHook)
	if !ok {
		return
	}
	hook = current

	var (
		line           []cube.Pos
		otherPos       cube.Pos
		foundOtherHook bool
		pathClear      = true
		linePowered    bool
	)
	for distance := 1; distance < maxTripwireCircuitLength; distance++ {
		scanPos := pos.Add(tripwireOffset(hook.Facing, distance))
		switch b := tx.Block(scanPos).(type) {
		case String:
			line = append(line, scanPos)
			if b.Disarmed {
				pathClear = false
			}
			if b.Powered && !b.Disarmed {
				linePowered = true
			}
		case TripwireHook:
			if distance > 1 && b.Facing == hook.Facing.Opposite() {
				otherPos = scanPos
				foundOtherHook = true
			}
			distance = maxTripwireCircuitLength
		default:
			pathClear = false
		}
	}

	attached := foundOtherHook && pathClear && len(line) > 0
	powered := attached && linePowered
	if hook.Attached != attached || hook.Powered != powered {
		hook.Attached, hook.Powered = attached, powered
		tx.SetBlock(pos, hook, nil)
	}
	if foundOtherHook {
		if other, ok := tx.Block(otherPos).(TripwireHook); ok && (other.Attached != attached || other.Powered != powered) {
			other.Attached, other.Powered = attached, powered
			tx.SetBlock(otherPos, other, nil)
		}
	}
	for _, wirePos := range line {
		wire, ok := tx.Block(wirePos).(String)
		if !ok {
			continue
		}
		connections := calculateConnections(wire, tx, wirePos)
		if wire.Attached == attached && wire.Connections == connections {
			continue
		}
		wire.Attached, wire.Connections = attached, connections
		tx.SetBlock(wirePos, wire, nil)
	}
}

// updateTripwireLinesForWire searches in all four horizontal directions for
// endpoints whose circuit may have changed because a string segment changed.
func updateTripwireLinesForWire(pos cube.Pos, tx *world.Tx) {
	for _, direction := range cube.Directions() {
		for distance := 1; distance < maxTripwireCircuitLength; distance++ {
			scanPos := pos.Add(tripwireOffset(direction, distance))
			if hook, ok := tx.Block(scanPos).(TripwireHook); ok {
				if hook.Facing == direction.Opposite() {
					updateTripwireHookLine(scanPos, hook, tx)
				}
				break
			}
		}
	}
}

// allTripwireHooks returns every valid combination of tripwire hook states.
func allTripwireHooks() (all []world.Block) {
	for _, facing := range cube.Directions() {
		for state := range 4 {
			all = append(all, TripwireHook{
				Facing:   facing,
				Attached: state&1 != 0,
				Powered:  state&2 != 0,
			})
		}
	}
	return
}
