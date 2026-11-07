package item

import (
	"time"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
)

// WindCharge is a throwable projectile that creates a short-range wind burst.
type WindCharge struct{}

// MaxCount returns the maximum stack size of a wind charge.
func (WindCharge) MaxCount() int { return 64 }

// Cooldown returns the ten-tick use cooldown shared by all wind charge stacks.
func (WindCharge) Cooldown() time.Duration { return 10 * time.Second / 20 }

// Use launches a wind charge projectile in the direction the user is facing.
func (WindCharge) Use(tx *world.Tx, user User, ctx *UseContext) bool {
	create := tx.World().EntityRegistry().Config().WindCharge
	if create == nil {
		return false
	}
	opts := world.EntitySpawnOpts{
		Position: eyePosition(user),
		Rotation: user.Rotation().Neg(),
		Velocity: user.Rotation().Vec3().Mul(1.5),
	}
	tx.AddEntity(create(opts, user))
	tx.PlaySound(user.Position(), sound.Custom{Name: "random.bow", Volume: 0.5, Pitch: 0.42})
	ctx.SubtractFromCount(1)
	return true
}

// EncodeItem ...
func (WindCharge) EncodeItem() (name string, meta int16) {
	return "minecraft:wind_charge", 0
}
