package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// RedstoneLamp is a block that lights up when powered. It stays lit for four
// game ticks after losing power, matching Bedrock Edition's redstone lamp.
type RedstoneLamp struct {
	solid

	// Lit indicates whether the lamp emits light.
	Lit bool
}

var (
	_ world.RedstonePowerConsumer = RedstoneLamp{}
	_ world.ScheduledTicker       = RedstoneLamp{}
)

// LightEmissionLevel returns full block light while the lamp is lit.
func (r RedstoneLamp) LightEmissionLevel() uint8 {
	if r.Lit {
		return 15
	}
	return 0
}

// RedstonePowerUpdate turns the lamp on immediately and schedules its delayed
// turn-off when power is removed.
func (r RedstoneLamp) RedstonePowerUpdate(pos cube.Pos, tx *world.Tx, power int) (world.Block, bool) {
	if power > 0 {
		if r.Lit {
			return r, false
		}
		r.Lit = true
		return r, true
	}
	if r.Lit && tx != nil {
		// Two redstone ticks equal four normal 20 Hz game ticks.
		tx.ScheduleBlockUpdate(pos, r, redstoneTicks(2))
	}
	return r, false
}

// ScheduledTick turns the lamp off only if it is still unpowered. Checking
// live power prevents an old off timer from extinguishing a lamp that was
// powered again before the timer expired.
func (r RedstoneLamp) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(RedstoneLamp)
	if !ok || !current.Lit || tx.RedstonePower(pos) > 0 {
		return
	}
	current.Lit = false
	tx.SetBlock(pos, current, nil)
}

// UseOnBlock places an unlit redstone lamp.
func (r RedstoneLamp) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, r)
	if !used {
		return false
	}
	r.Lit = false
	place(tx, pos, r, user, ctx)
	return placed(ctx)
}

func (r RedstoneLamp) BreakInfo() BreakInfo {
	return newBreakInfo(0.3, pickaxeHarvestable, pickaxeEffective, oneOf(RedstoneLamp{}))
}

func (RedstoneLamp) EncodeItem() (string, int16) {
	return "minecraft:redstone_lamp", 0
}

func (r RedstoneLamp) EncodeBlock() (string, map[string]any) {
	if r.Lit {
		return "minecraft:lit_redstone_lamp", nil
	}
	return "minecraft:redstone_lamp", nil
}

func allRedstoneLamps() []world.Block {
	return []world.Block{RedstoneLamp{}, RedstoneLamp{Lit: true}}
}
