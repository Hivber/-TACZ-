package entity

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/go-gl/mathgl/mgl64"
)

// init connects the block package's dispenser hook to entity constructors without creating an import cycle.
func init() {
	block.RegisterDispenserItemBehavior("minecraft:item", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) block.DispenserItemBehaviorResult {
		tx.AddEntity(NewItem(opts, stack))
		tx.PlaySound(opts.Position, sound.Click{})
		return block.DispenserItemHandled
	})
	block.RegisterDispenserItemBehavior("minecraft:arrow", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) block.DispenserItemBehaviorResult {
		arrow, ok := stack.Item().(item.Arrow)
		if !ok {
			return block.DispenserItemFailed
		}
		conf := arrowConf
		conf.Potion = arrow.Tip
		conf.PickupItem = item.NewStack(arrow, 1)
		tx.AddEntity(opts.New(ArrowType, conf))
		tx.PlaySound(opts.Position, sound.BowShoot{})
		return block.DispenserItemHandled
	})
	block.RegisterDispenserItemBehavior("minecraft:egg", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) block.DispenserItemBehaviorResult {
		if _, ok := stack.Item().(item.Egg); !ok {
			return block.DispenserItemFailed
		}
		tx.AddEntity(opts.New(EggType, eggConf))
		tx.PlaySound(opts.Position, sound.BowShoot{})
		return block.DispenserItemHandled
	})
	block.RegisterDispenserItemBehavior("minecraft:snowball", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) block.DispenserItemBehaviorResult {
		if _, ok := stack.Item().(item.Snowball); !ok {
			return block.DispenserItemFailed
		}
		tx.AddEntity(opts.New(SnowballType, snowballConf))
		tx.PlaySound(opts.Position, sound.BowShoot{})
		return block.DispenserItemHandled
	})
	block.RegisterDispenserItemBehavior("minecraft:firework_rocket", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) block.DispenserItemBehaviorResult {
		firework, ok := stack.Item().(item.Firework)
		if !ok {
			return block.DispenserItemFailed
		}
		tx.AddEntity(NewFirework(opts, firework))
		tx.PlaySound(opts.Position, sound.BowShoot{})
		return block.DispenserItemHandled
	})
	block.RegisterDispenserItemBehavior("minecraft:tnt", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) block.DispenserItemBehaviorResult {
		if _, ok := stack.Item().(block.TNT); !ok {
			return block.DispenserItemFailed
		}
		pos := cube.Pos{
			int(math.Floor(opts.Position[0])),
			int(math.Floor(opts.Position[1])),
			int(math.Floor(opts.Position[2])),
		}
		opts.Position = mgl64.Vec3{
			float64(pos[0]) + 0.5,
			float64(pos[1]),
			float64(pos[2]) + 0.5,
		}
		opts.Velocity = mgl64.Vec3{
			rand.NormFloat64() * 0.02,
			0.2,
			rand.NormFloat64() * 0.02,
		}
		create := tx.World().EntityRegistry().Config().TNT
		if create == nil {
			return block.DispenserItemFailed
		}
		handle := create(opts, 4*time.Second)
		if handle == nil {
			return block.DispenserItemFailed
		}
		tx.AddEntity(handle)
		tx.PlaySound(opts.Position, sound.TNT{})
		return block.DispenserItemHandled
	})
	block.RegisterDispenserItemBehavior("minecraft:flint_and_steel", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) block.DispenserItemBehaviorResult {
		if _, ok := stack.Item().(item.FlintAndSteel); !ok {
			return block.DispenserItemFailed
		}
		pos := cube.Pos{
			int(math.Floor(opts.Position[0])),
			int(math.Floor(opts.Position[1])),
			int(math.Floor(opts.Position[2])),
		}
		if igniter, ok := tx.Block(pos).(interface {
			Ignite(cube.Pos, *world.Tx, world.Entity) bool
		}); ok {
			if igniter.Ignite(pos, tx, nil) {
				tx.PlaySound(opts.Position, sound.Ignite{})
				return block.DispenserItemDamaged
			}
			return block.DispenserItemFailed
		}
		before, _ := tx.Block(pos).EncodeBlock()
		block.Fire{}.Start(tx, pos)
		after, _ := tx.Block(pos).EncodeBlock()
		if before != after && (after == "minecraft:fire" || after == "minecraft:portal") {
			tx.PlaySound(opts.Position, sound.Ignite{})
			return block.DispenserItemDamaged
		}
		return block.DispenserItemFailed
	})
	block.RegisterDispenserItemBehavior("minecraft:bone_meal", func(tx *world.Tx, opts world.EntitySpawnOpts, _ item.Stack) block.DispenserItemBehaviorResult {
		pos := cube.Pos{
			int(math.Floor(opts.Position[0])),
			int(math.Floor(opts.Position[1])),
			int(math.Floor(opts.Position[2])),
		}
		affected, ok := tx.Block(pos).(item.BoneMealAffected)
		if !ok {
			return block.DispenserItemFailed
		}
		result := affected.BoneMeal(pos, tx)
		if result == item.BoneMealResultNone {
			return block.DispenserItemFailed
		}
		tx.AddParticle(pos.Vec3(), particle.BoneMeal{Area: result == item.BoneMealResultArea})
		return block.DispenserItemHandled
	})
	block.RegisterDispenserItemBehavior("minecraft:spawn_egg", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) block.DispenserItemBehaviorResult {
		egg, ok := stack.Item().(item.SpawnEgg)
		if !ok || egg.Mob == "" {
			return block.DispenserItemFailed
		}
		create := tx.World().EntityRegistry().Config().Mob
		if create == nil {
			return block.DispenserItemFailed
		}
		spawnPos := cube.Pos{
			int(math.Floor(opts.Position[0])),
			int(math.Floor(opts.Position[1])),
			int(math.Floor(opts.Position[2])),
		}
		isAir := func(pos cube.Pos) bool {
			name, _ := tx.Block(pos).EncodeBlock()
			return name == "minecraft:air"
		}
		if !isAir(spawnPos) || !isAir(spawnPos.Side(cube.FaceUp)) {
			return block.DispenserItemFailed
		}
		spawnOpts := world.EntitySpawnOpts{
			Position: mgl64.Vec3{
				float64(spawnPos[0]) + 0.5,
				float64(spawnPos[1]),
				float64(spawnPos[2]) + 0.5,
			},
		}
		handle := create(spawnOpts, egg.Mob)
		if handle == nil {
			return block.DispenserItemFailed
		}
		tx.AddEntity(handle)
		tx.PlaySound(opts.Position, sound.Click{})
		return block.DispenserItemHandled
	})
}
