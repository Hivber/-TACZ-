package cmd

import (
	"fmt"
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

const maxWorldBorderTransition = 30 * 24 * time.Hour

func init() {
	Register(New("worldborder", "Queries and changes the current world's horizontal movement border.", nil,
		worldBorderGetCommand{}, worldBorderSetCommand{}, worldBorderAddCommand{}, worldBorderCenterCommand{}))
}

type worldBorderGetCommand struct {
	Get SubCommand
}

func (worldBorderGetCommand) Allow(src Source) bool { return commandOperator(src) }

func (worldBorderGetCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("worldborder requires a world transaction.")
		return
	}
	border := tx.WorldBorder()
	x, z := border.Center()
	out.Printf("World border diameter: %.2f blocks (centre %.2f, %.2f).", border.Size(), x, z)
}

type worldBorderSetCommand struct {
	Set     SubCommand
	Size    float64
	Seconds Optional[float64]
}

func (worldBorderSetCommand) Allow(src Source) bool { return commandOperator(src) }

func (c worldBorderSetCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("worldborder requires a world transaction.")
		return
	}
	if err := validateWorldBorderSize(c.Size); err != nil {
		out.Error(err)
		return
	}
	duration, err := worldBorderDuration(c.Seconds)
	if err != nil {
		out.Error(err)
		return
	}
	tx.SetWorldBorder(tx.WorldBorder().WithSize(c.Size, duration))
	out.Printf("World border is transitioning to %.2f blocks over %s.", c.Size, duration)
}

type worldBorderAddCommand struct {
	Add      SubCommand
	Distance float64
	Seconds  Optional[float64]
}

func (worldBorderAddCommand) Allow(src Source) bool { return commandOperator(src) }

func (c worldBorderAddCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("worldborder requires a world transaction.")
		return
	}
	border := tx.WorldBorder()
	size := border.Size() + c.Distance
	if err := validateWorldBorderSize(size); err != nil {
		out.Error(err)
		return
	}
	duration, err := worldBorderDuration(c.Seconds)
	if err != nil {
		out.Error(err)
		return
	}
	tx.SetWorldBorder(border.WithSize(size, duration))
	out.Printf("World border is transitioning to %.2f blocks over %s.", size, duration)
}

type worldBorderCenterCommand struct {
	Center SubCommand
	X      float64
	Z      float64
}

func (worldBorderCenterCommand) Allow(src Source) bool { return commandOperator(src) }

func (c worldBorderCenterCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("worldborder requires a world transaction.")
		return
	}
	if !validBorderCoordinate(c.X) || !validBorderCoordinate(c.Z) {
		out.Error("world border centre coordinates must be finite and within +/- 30,000,000.")
		return
	}
	tx.SetWorldBorder(tx.WorldBorder().WithCenter(c.X, c.Z))
	out.Printf("World border centre set to %.2f, %.2f.", c.X, c.Z)
}

func validateWorldBorderSize(size float64) error {
	if math.IsNaN(size) || math.IsInf(size, 0) || size < 1 || size > world.DefaultWorldBorderSize {
		return fmt.Errorf("world border diameter must be between 1 and %.0f blocks", world.DefaultWorldBorderSize)
	}
	return nil
}

func validBorderCoordinate(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= -30000000 && value <= 30000000
}

func worldBorderDuration(seconds Optional[float64]) (time.Duration, error) {
	value := seconds.LoadOr(0)
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > maxWorldBorderTransition.Seconds() {
		return 0, fmt.Errorf("world border transition time must be between 0 and %s seconds", maxWorldBorderTransition)
	}
	return time.Duration(value * float64(time.Second)), nil
}
