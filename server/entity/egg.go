package entity

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/go-gl/mathgl/mgl64"
)

// NewEgg creates an Egg entity. Egg is as a throwable entity that can be used
// to spawn chicks.
func NewEgg(opts world.EntitySpawnOpts, owner world.Entity) *world.EntityHandle {
	conf := eggConf
	conf.Owner = owner.H()
	return opts.New(EggType, conf)
}

var eggConf = ProjectileBehaviourConfig{
	Gravity:       0.03,
	Drag:          0.01,
	Damage:        -1,
	Particle:      particle.EggSmash{},
	ParticleCount: 6,
	Hit:           eggHit,
}

// eggHit implements the vanilla chicken-hatching chance of a thrown egg.
// One chick has a 1/8 chance to spawn; on a successful hatch there is a
// separate 1/32 chance to spawn four chicks instead.
func eggHit(_ *Ent, tx *world.Tx, target trace.Result) {
	if rand.IntN(8) != 0 {
		return
	}
	count := 1
	if rand.IntN(32) == 0 {
		count = 4
	}
	create := tx.World().EntityRegistry().Config().Mob
	if create == nil {
		return
	}
	pos := target.Position().Add(mgl64.Vec3{0, 0.25})
	for range count {
		handle := create(world.EntitySpawnOpts{Position: pos}, "minecraft:chicken")
		if handle != nil {
			tx.AddEntity(handle)
		}
	}
}

// EggType is a world.EntityType implementation for Egg.
var EggType eggType

type eggType struct{}

func (t eggType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	return &Ent{tx: tx, handle: handle, data: data}
}

func (eggType) EncodeEntity() string { return "minecraft:egg" }
func (eggType) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.125, 0, -0.125, 0.125, 0.25, 0.125)
}

func (eggType) DecodeNBT(_ map[string]any, data *world.EntityData) { data.Data = eggConf.New() }
func (eggType) EncodeNBT(_ *world.EntityData) map[string]any       { return nil }
