package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Button is the collision model of an attached redstone button.
type Button struct {
	// Facing is the face pointing away from the supporting block.
	Facing cube.Face
	// Pressed reports whether the button is currently depressed.
	Pressed bool
}

// BBox returns the small collision box attached to the supporting face.
func (b Button) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	thickness := 2.0 / 16
	if b.Pressed {
		thickness = 1.0 / 16
	}
	const min = 5.0 / 16
	const max = 11.0 / 16

	switch b.Facing {
	case cube.FaceDown:
		return []cube.BBox{cube.Box(min, 1-thickness, min, max, 1, max)}
	case cube.FaceUp:
		return []cube.BBox{cube.Box(min, 0, min, max, thickness, max)}
	case cube.FaceNorth:
		return []cube.BBox{cube.Box(min, min, 1-thickness, max, max, 1)}
	case cube.FaceSouth:
		return []cube.BBox{cube.Box(min, min, 0, max, max, thickness)}
	case cube.FaceWest:
		return []cube.BBox{cube.Box(1-thickness, min, min, 1, max, max)}
	case cube.FaceEast:
		return []cube.BBox{cube.Box(0, min, min, thickness, max, max)}
	default:
		return nil
	}
}

// FaceSolid always returns false.
func (Button) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
