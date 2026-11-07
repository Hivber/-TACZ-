package entity

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
)

// init connects the block package's dispenser hook to entity constructors without creating an import cycle.
func init() {
	block.RegisterDispenserItemBehavior("minecraft:item", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) bool {
		tx.AddEntity(NewItem(opts, stack))
		tx.PlaySound(opts.Position, sound.Click{})
		return true
	})
	block.RegisterDispenserItemBehavior("minecraft:arrow", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) bool {
		arrow, ok := stack.Item().(item.Arrow)
		if !ok {
			return false
		}
		conf := arrowConf
		conf.Potion = arrow.Tip
		conf.PickupItem = item.NewStack(arrow, 1)
		tx.AddEntity(opts.New(ArrowType, conf))
		tx.PlaySound(opts.Position, sound.BowShoot{})
		return true
	})
	block.RegisterDispenserItemBehavior("minecraft:egg", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) bool {
		if _, ok := stack.Item().(item.Egg); !ok {
			return false
		}
		tx.AddEntity(opts.New(EggType, eggConf))
		tx.PlaySound(opts.Position, sound.BowShoot{})
		return true
	})
	block.RegisterDispenserItemBehavior("minecraft:snowball", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) bool {
		if _, ok := stack.Item().(item.Snowball); !ok {
			return false
		}
		tx.AddEntity(opts.New(SnowballType, snowballConf))
		tx.PlaySound(opts.Position, sound.BowShoot{})
		return true
	})
	block.RegisterDispenserItemBehavior("minecraft:firework_rocket", func(tx *world.Tx, opts world.EntitySpawnOpts, stack item.Stack) bool {
		firework, ok := stack.Item().(item.Firework)
		if !ok {
			return false
		}
		tx.AddEntity(NewFirework(opts, firework))
		tx.PlaySound(opts.Position, sound.BowShoot{})
		return true
	})
}
