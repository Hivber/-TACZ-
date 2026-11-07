package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// PressurePlate is the collision model of a floor-mounted pressure plate.
type PressurePlate struct {
	// Pressed reports whether the plate is currently depressed.
	Pressed bool
}

// BBox returns the plate's thin collision box, lowered while pressed.
func (p PressurePlate) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	height := 0.0625
	if p.Pressed {
		height = 0.03125
	}
	return []cube.BBox{cube.Box(0.0625, 0, 0.0625, 0.9375, height, 0.9375)}
}

// FaceSolid always returns false.
func (PressurePlate) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
