package entity

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	windChargeBurstRadius = 2.4
	windChargeHorizontal  = 0.75
	windChargeVertical    = 0.9
)

// NewWindCharge creates a wind charge projectile fired by a player.
func NewWindCharge(opts world.EntitySpawnOpts, owner world.Entity) *world.EntityHandle {
	conf := windChargeConf
	conf.Owner = owner.H()
	return opts.New(WindChargeType, conf)
}

var windChargeConf = ProjectileBehaviourConfig{
	Gravity:       0,
	Drag:          0,
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

// windChargeHit applies the burst to nearby living entities. The generic
// projectile behaviour handles the direct-impact damage.
func windChargeHit(_ *Ent, tx *world.Tx, target trace.Result) {
	centre := target.Position()
	r := windChargeBurstRadius
	box := cube.Box(-r, -r, -r, r, r, r).Translate(centre)
	for other := range tx.EntitiesWithin(box) {
		living, ok := other.(Living)
		if !ok {
			continue
		}
		impulse := windChargeImpulse(living.Position().Sub(centre))
		if impulse.LenSqr() == 0 {
			continue
		}
		living.SetVelocity(living.Velocity().Add(impulse))
	}
}

// windChargeImpulse returns the horizontal and upward velocity added by a
// player-fired wind burst. The effect fades linearly to zero at 2.4 blocks.
func windChargeImpulse(offset mgl64.Vec3) mgl64.Vec3 {
	distance := offset.Len()
	if distance >= windChargeBurstRadius {
		return mgl64.Vec3{}
	}
	strength := 1 - distance/windChargeBurstRadius
	horizontal := mgl64.Vec3{offset[0], 0, offset[2]}
	if horizontal.LenSqr() > 0 {
		horizontal = horizontal.Normalize()
	}
	return mgl64.Vec3{
		horizontal[0] * windChargeHorizontal * strength,
		windChargeVertical * strength,
		horizontal[2] * windChargeHorizontal * strength,
	}
}
