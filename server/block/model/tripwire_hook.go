package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// TripwireHook is the collision model of a wall-mounted tripwire hook.
type TripwireHook struct {
	// Facing is the horizontal direction in which the hook points away from its support block.
	Facing cube.Direction
}

// BBox returns the thin wall plate and hook collision box.
func (h TripwireHook) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	const (
		minY = 4.0 / 16
		maxY = 12.0 / 16
	)
	switch h.Facing {
	case cube.North:
		return []cube.BBox{cube.Box(6.0/16, minY, 14.0/16, 10.0/16, maxY, 1)}
	case cube.South:
		return []cube.BBox{cube.Box(6.0/16, minY, 0, 10.0/16, maxY, 2.0/16)}
	case cube.West:
		return []cube.BBox{cube.Box(14.0/16, minY, 6.0/16, 1, maxY, 10.0/16)}
	case cube.East:
		return []cube.BBox{cube.Box(0, minY, 6.0/16, 2.0/16, maxY, 10.0/16)}
	default:
		return nil
	}
}

// FaceSolid always returns false.
func (TripwireHook) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
