package block

import (
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// String is an item obtained from spiders and cobwebs. When placed, it creates a tripwire that
// detects entities passing through it.
type String struct {
	empty
	transparent

	// Attached is true if the tripwire is connected to valid tripwire hooks on both sides.
	Attached bool
	// Disarmed is true if the tripwire was cut using shears, preventing it from activating.
	Disarmed bool
	// Powered is true if the tripwire is currently activated by an entity passing through it.
	Powered bool
	// Suspended is true if the tripwire is not resting on a solid surface.
	Suspended bool
	// Connections holds the sides that the tripwire connects to.
	Connections Connections
}

var (
	_ EntityInsider               = String{}
	_ world.ScheduledTicker       = String{}
	_ world.NeighbourUpdateTicker = String{}
)

// UseOnBlock places the string as a tripwire on the target surface.
func (s String) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, canPlace := firstReplaceable(tx, pos, face, s)
	if !canPlace {
		return false
	}
	below := pos.Side(cube.FaceDown)
	s.Suspended = !tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx)
	s.Connections = calculateConnections(s, tx, pos)
	place(tx, pos, s, user, ctx)
	updateTripwireLinesForWire(pos, tx)
	return placed(ctx)
}

// EntityInside powers an armed tripwire when an entity intersects the wire.
func (s String) EntityInside(pos cube.Pos, tx *world.Tx, e world.Entity) {
	current, ok := tx.Block(pos).(String)
	if !ok || current.Disarmed || current.Powered || !tripwireEntityIntersects(pos, e) {
		return
	}
	current.Powered = true
	tx.SetBlock(pos, current, nil)
	tx.ScheduleBlockUpdate(pos, current, tripwireReleaseInterval)
	updateTripwireLinesForWire(pos, tx)
}

// ScheduledTick releases a powered tripwire after all entities leave its wire plane.
func (s String) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(String)
	if !ok || !current.Powered {
		return
	}
	if tripwireEntityStillInside(pos, tx) {
		tx.ScheduleBlockUpdate(pos, current, tripwireReleaseInterval)
		return
	}
	current.Powered = false
	tx.SetBlock(pos, current, nil)
	updateTripwireLinesForWire(pos, tx)
}

const tripwireReleaseInterval = time.Second / 2

func tripwireWireBox(pos cube.Pos) cube.BBox {
	return cube.Box(
		float64(pos[0]), float64(pos[1])+7.0/16, float64(pos[2]),
		float64(pos[0]+1), float64(pos[1])+9.0/16, float64(pos[2]+1),
	)
}

func tripwireEntityIntersects(pos cube.Pos, e world.Entity) bool {
	entityBox := e.H().Type().BBox(e).Translate(e.Position())
	return entityBox.IntersectsWith(tripwireWireBox(pos))
}

func tripwireEntityStillInside(pos cube.Pos, tx *world.Tx) bool {
	searchBox := cube.Box(
		float64(pos[0])-0.25, float64(pos[1])-0.25, float64(pos[2])-0.25,
		float64(pos[0])+1.25, float64(pos[1])+1.25, float64(pos[2])+1.25,
	)
	for e := range tx.EntitiesWithin(searchBox) {
		if tripwireEntityIntersects(pos, e) {
			return true
		}
	}
	return false
}

// NeighbourUpdateTick refreshes tripwire suspension, visual connections and hook circuits.
func (s String) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	below := pos.Side(cube.FaceDown)
	suspended, connections := !tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx), calculateConnections(s, tx, pos)
	if suspended == s.Suspended && connections == s.Connections {
		return
	}
	s.Suspended, s.Connections = suspended, connections
	tx.SetBlock(pos, s, nil)
	updateTripwireLinesForWire(pos, tx)
}

// DeriveState ...
//
// Suspended is stored on disk, so only Connections is derived.
func (s String) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	s.Connections = calculateConnections(s, src, pos)
	return s
}

// Connects returns true if the tripwire connects to the block at the face
// passed. Tripwire connects to other tripwire.
func (String) Connects(pos cube.Pos, face cube.Face, src world.BlockSource) bool {
	neighbour := src.Block(pos.Side(face))
	if _, ok := neighbour.(String); ok {
		return true
	}
	if hook, ok := neighbour.(TripwireHook); ok && face.Axis() != cube.Y {
		return hook.Facing == face.Opposite().Direction()
	}
	return false
}

// HasLiquidDrops ...
func (s String) HasLiquidDrops() bool {
	return true
}

// BreakInfo ...
func (s String) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(String{})).
		withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
			updateTripwireLinesForWire(pos, tx)
		})
}

// EncodeItem ...
func (s String) EncodeItem() (name string, meta int16) {
	return "minecraft:string", 0
}

// EncodeBlock ...
func (s String) EncodeBlock() (name string, properties map[string]any) {
	properties = s.Connections.properties()
	properties["attached_bit"] = boolByte(s.Attached)
	properties["disarmed_bit"] = boolByte(s.Disarmed)
	properties["powered_bit"] = boolByte(s.Powered)
	properties["suspended_bit"] = boolByte(s.Suspended)
	return "minecraft:trip_wire", properties
}

// allString ...
func allString() (blocks []world.Block) {
	for meta := 0; meta < 16; meta++ {
		for _, c := range allConnections() {
			blocks = append(blocks, String{
				Powered:     meta&0x1 != 0,
				Suspended:   meta&0x2 != 0,
				Attached:    meta&0x4 != 0,
				Disarmed:    meta&0x8 != 0,
				Connections: c,
			})
		}
	}
	return blocks
}
