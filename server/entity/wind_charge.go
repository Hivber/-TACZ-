package entity

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	windChargeBurstRadius = 2.0
	windChargeKnockback   = 0.2
	windChargeUpward      = 0.6
)

// NewWindCharge creates a wind charge projectile fired by a player.
func NewWindCharge(opts world.EntitySpawnOpts, owner world.Entity) *world.EntityHandle {
	conf := windChargeConf
	conf.Owner = owner.H()
	return opts.New(WindChargeType, conf)
}

var windChargeConf = ProjectileBehaviourConfig{
	Gravity:       0,
	Drag:          0.01,
	Damage:        0.5,
	Sound:         sound.Custom{Name: "wind_charge.burst", Volume: 1, Pitch: 1},
	Particle:      particle.HugeExplosion{},
	ParticleCount: 1,
	Hit:           windChargeHit,
}

// WindChargeType is the entity type of a player-thrown wind charge.
var WindChargeType windChargeType

type windChargeType struct{}

func (windChargeType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	return &Ent{tx: tx, handle: handle, data: data}
}

func (windChargeType) EncodeEntity() string { return "minecraft:wind_charge_projectile" }

func (windChargeType) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.125, -0.125, -0.125, 0.125, 0.125, 0.125)
}

func (windChargeType) DecodeNBT(_ map[string]any, data *world.EntityData) {
	data.Data = windChargeConf.New()
}

func (windChargeType) EncodeNBT(_ *world.EntityData) map[string]any { return nil }

// windChargeHit applies the burst and block interactions after the generic
// projectile behaviour has processed direct-hit damage and ProjectileHitter blocks.
func windChargeHit(projectile *Ent, tx *world.Tx, target trace.Result) {
	centre := projectile.Position()
	switch result := target.(type) {
	case trace.BlockResult:
		activateWindChargeBlock(result.BlockPosition(), result.Face(), projectile, tx)
		for other := range tx.EntitiesWithin(cube.Box(
			-windChargeBurstRadius, -windChargeBurstRadius, -windChargeBurstRadius,
			windChargeBurstRadius, windChargeBurstRadius, windChargeBurstRadius,
		).Translate(centre)) {
			applyWindChargeBurst(other, centre)
		}
	case trace.EntityResult:
		applyWindChargeBurst(result.Entity(), centre)
	}
}

// activateWindChargeBlock applies the supported Bedrock wind-charge block
// interactions. Decorated pots are handled by the generic ProjectileHitter path.
func activateWindChargeBlock(pos cube.Pos, face cube.Face, projectile *Ent, tx *world.Tx) {
	b := tx.Block(pos)
	switch b.(type) {
	case block.WoodDoor, block.CopperDoor,
		block.WoodTrapdoor, block.CopperTrapdoor,
		block.WoodFenceGate, block.Lever:
		if activatable, ok := b.(block.Activatable); ok {
			user := windChargeBlockUser{position: projectile.Position(), rotation: projectile.Rotation()}
			activatable.Activate(pos, face, tx, user, &item.UseContext{})
		}
	case block.Candle:
		if candle, ok := b.(block.Candle); ok {
			candle.Ignite(pos, tx, projectile)
		}
	}
}

// applyWindChargeBurst applies the Bedrock wind-burst velocity to one living entity.
// Existing velocity is damped by half, then radial and upward impulses are added.
func applyWindChargeBurst(e world.Entity, centre mgl64.Vec3) {
	living, ok := e.(Living)
	if !ok {
		return
	}
	offset := living.Position().Sub(centre)
	if offset.Len() >= windChargeBurstRadius {
		return
	}
	living.SetVelocity(windChargeImpulse(living.Velocity(), offset))
	if resetter, ok := e.(interface{ ResetFallDistance() }); ok {
		resetter.ResetFallDistance()
	}
}

// windChargeImpulse returns the resulting velocity after a wind-charge burst.
// The formula follows the PowerNukkitX Bedrock implementation.
func windChargeImpulse(velocity, offset mgl64.Vec3) mgl64.Vec3 {
	result := velocity.Mul(0.5)
	result[0] += offset[0] * windChargeKnockback
	result[1] += windChargeUpward
	result[2] += offset[2] * windChargeKnockback
	return result
}

// windChargeBlockUser provides the small item.User surface needed by block
// activation methods. It carries the projectile direction but has no inventory.
type windChargeBlockUser struct {
	position mgl64.Vec3
	rotation cube.Rotation
}

func (u windChargeBlockUser) H() *world.EntityHandle       { return nil }
func (u windChargeBlockUser) Close() error                 { return nil }
func (u windChargeBlockUser) Position() mgl64.Vec3         { return u.position }
func (u windChargeBlockUser) Rotation() cube.Rotation      { return u.rotation }
func (windChargeBlockUser) HeldItems() (item.Stack, item.Stack) { return item.Stack{}, item.Stack{} }
func (windChargeBlockUser) SetHeldItems(item.Stack, item.Stack) {}
func (windChargeBlockUser) UsingItem() bool                 { return false }
func (windChargeBlockUser) ReleaseItem()                    {}
func (windChargeBlockUser) UseItem()                        {}
