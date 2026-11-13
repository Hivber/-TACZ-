package player

import (
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/gamecore"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
)

const (
	rustToolSword   uint8 = 1 << 0
	rustToolShovel  uint8 = 1 << 1
	rustToolPickaxe uint8 = 1 << 2
	rustToolAxe     uint8 = 1 << 3
	rustToolShears  uint8 = 1 << 4
	rustToolHoe     uint8 = 1 << 5
)

func rustToolMask(tool item.Tool) uint8 {
	switch tool.ToolType() {
	case item.TypeSword:
		return rustToolSword
	case item.TypeShovel:
		return rustToolShovel
	case item.TypePickaxe:
		return rustToolPickaxe
	case item.TypeAxe:
		return rustToolAxe
	case item.TypeShears:
		return rustToolShears
	case item.TypeHoe:
		return rustToolHoe
	default:
		return 0
	}
}

func rustMiningLevel(level int) uint8 {
	return uint8(max(0, min(level, 255)))
}

// rustMiningParameters derives Rust's mining inputs from the current block,
// selected item, armour and status effects. The block drop and world mutation
// rules remain in Dragonfly until those systems are migrated.
func (p *Player) rustMiningParameters(pos cube.Pos) (gamecore.BlockBreakProfile, gamecore.MiningTool, gamecore.MinerState, bool) {
	b := p.tx.Block(pos)
	breakable, ok := b.(block.Breakable)
	if !ok {
		return gamecore.BlockBreakProfile{}, gamecore.MiningTool{}, gamecore.MinerState{}, false
	}
	info := breakable.BreakInfo()
	if info.Hardness <= 0 || math.IsNaN(info.Hardness) || math.IsInf(info.Hardness, 0) {
		return gamecore.BlockBreakProfile{}, gamecore.MiningTool{}, gamecore.MinerState{}, false
	}

	held, _ := p.HeldItems()
	tool, ok := held.Item().(item.Tool)
	if !ok {
		tool = item.ToolNone{}
	}
	toolMask := rustToolMask(tool)
	harvestable := info.Harvestable(tool)
	effective := info.Effective(tool)

	profileToolMask := uint8(0)
	if effective {
		profileToolMask = toolMask
	}
	profileHarvestLevel := uint8(0)
	if !harvestable {
		// Encode a correct tool with an insufficient tier separately from an
		// ineffective tool, matching Dragonfly's harvest and speed decisions.
		if profileToolMask == 0 {
			profileToolMask = 1 << 7
		}
		profileHarvestLevel = rustMiningLevel(tool.HarvestLevel() + 1)
		if profileHarvestLevel == 0 {
			profileHarvestLevel = 1
		}
	}

	profile := gamecore.BlockBreakProfile{
		Hardness:            float32(info.Hardness),
		ToolType:            profileToolMask,
		HarvestLevel:        profileHarvestLevel,
		RequiresCorrectTool: !harvestable,
	}
	miningTool := gamecore.MiningTool{
		ToolType:            toolMask,
		HarvestLevel:        rustMiningLevel(tool.HarvestLevel()),
		EfficiencyCorrect:   1,
		EfficiencyIncorrect: 1,
	}
	if effective && harvestable {
		speed := tool.BaseMiningEfficiency(b)
		if speed > 0 && !math.IsNaN(speed) && !math.IsInf(speed, 0) && speed <= math.MaxFloat32 {
			miningTool.EfficiencyCorrect = float32(speed)
		}
		if efficiency, found := held.Enchantment(enchantment.Efficiency); found {
			miningTool.EfficiencyEnchantment = rustMiningLevel(efficiency.Level())
		}
	}

	ctx := p.breakContext()
	miner := gamecore.MinerState{
		OnGround:                      p.OnGround(),
		Flying:                        p.Flying(),
		UnderwaterWithoutAquaAffinity: ctx.Underwater && !ctx.AquaAffinity,
		Haste:                         rustMiningLevel(max(ctx.HasteLevel, ctx.ConduitPowerLevel)),
		MiningFatigue:                 rustMiningLevel(ctx.MiningFatigueLevel),
	}
	return profile, miningTool, miner, true
}

func (p *Player) rustMiningTarget(pos cube.Pos, face cube.Face) gamecore.BlockTarget {
	return gamecore.BlockTarget{
		Position: gamecore.BlockPos{X: int32(pos[0]), Y: int32(pos[1]), Z: int32(pos[2])},
		Face:     uint8(face),
	}
}

func (p *Player) rustMiningEyePosition() [3]float32 {
	eye := entity.EyePosition(p)
	return [3]float32{float32(eye[0]), float32(eye[1]), float32(eye[2])}
}

func (p *Player) startRustMining(pos cube.Pos, face cube.Face) {
	p.rustMiningActive = false
	p.rustMiningTick = 0
	if !p.GameMode().AllowsEditing() || p.GameMode().CreativeInventory() {
		return
	}
	profile, tool, miner, ok := p.rustMiningParameters(pos)
	if !ok {
		return
	}
	result, ok := p.session().GameCoreMiningAction(gamecore.Event{
		Type:        "mining_start",
		UUID:        p.UUID().String(),
		Target:      rustMiningPointer(p.rustMiningTarget(pos, face)),
		Profile:     rustMiningPointer(profile),
		Tool:        rustMiningPointer(tool),
		Miner:       rustMiningPointer(miner),
		EyePosition: p.rustMiningEyePosition(),
		MaxReach:    8,
		Tick:        0,
	})
	if ok && (result.Status == "started" || result.Status == "progress") {
		p.rustMiningActive = true
	}
}

func (p *Player) advanceRustMining(face cube.Face) (completed, handled bool) {
	if !p.rustMiningActive {
		return false, false
	}
	pos := p.breakingPos
	profile, tool, miner, ok := p.rustMiningParameters(pos)
	if !ok {
		p.rustMiningActive = false
		p.rustMiningTick = 0
		return false, false
	}
	p.rustMiningTick++
	result, ok := p.session().GameCoreMiningAction(gamecore.Event{
		Type:        "mining_advance",
		UUID:        p.UUID().String(),
		Target:      rustMiningPointer(p.rustMiningTarget(pos, face)),
		Profile:     rustMiningPointer(profile),
		Tool:        rustMiningPointer(tool),
		Miner:       rustMiningPointer(miner),
		EyePosition: p.rustMiningEyePosition(),
		MaxReach:    8,
		Tick:        p.rustMiningTick,
	})
	if !ok {
		p.rustMiningActive = false
		p.rustMiningTick = 0
		return false, false
	}
	switch result.Status {
	case "progress":
		return false, true
	case "completed":
		p.rustMiningActive = false
		p.rustMiningTick = 0
		return true, true
	default:
		p.rustMiningActive = false
		p.rustMiningTick = 0
		return false, false
	}
}

func (p *Player) abortRustMining() {
	if !p.rustMiningActive {
		return
	}
	_, _ = p.session().GameCoreMiningAction(gamecore.Event{
		Type: "mining_abort",
		UUID: p.UUID().String(),
	})
	p.rustMiningActive = false
	p.rustMiningTick = 0
}

func rustMiningPointer[T any](value T) *T { return &value }
