package item

import (
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// SpawnEgg is an egg that creates one supported vanilla mob when used on a
// block. Mob is a minecraft entity identifier such as minecraft:pig.
type SpawnEgg struct {
	Mob string
}

// MaxCount returns the maximum stack size of a spawn egg.
func (SpawnEgg) MaxCount() int { return 64 }

// EncodeItem returns the Bedrock item identifier for the mob's spawn egg.
func (s SpawnEgg) EncodeItem() (string, int16) {
	name := strings.TrimPrefix(s.Mob, "minecraft:")
	return "minecraft:" + name + "_spawn_egg", 0
}

// UseOnBlock creates the mob in the space immediately beyond the clicked face.
func (s SpawnEgg) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, _ User, ctx *UseContext) bool {
	create := tx.World().EntityRegistry().Config().Mob
	if create == nil || s.Mob == "" {
		return false
	}
	spawnPos := pos.Side(face)
	if spawnPos.OutOfBounds(tx.Range()) {
		return false
	}
	if _, air := tx.Block(spawnPos).(block.Air); !air {
		return false
	}
	if _, air := tx.Block(spawnPos.Side(cube.FaceUp)).(block.Air); !air {
		return false
	}
	opts := world.EntitySpawnOpts{Position: mgl64.Vec3{float64(spawnPos[0]) + 0.5, float64(spawnPos[1]), float64(spawnPos[2]) + 0.5}}
	handle := create(opts, s.Mob)
	if handle == nil {
		return false
	}
	tx.AddEntity(handle)
	ctx.SubtractFromCount(1)
	return true
}
