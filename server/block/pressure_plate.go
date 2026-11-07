package block

import (
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	// PressurePlateStone is the stone pressure plate.
	PressurePlateStone = iota
	// PressurePlatePolishedBlackstone is the polished blackstone pressure plate.
	PressurePlatePolishedBlackstone
	// PressurePlateLightWeighted is the gold/light weighted pressure plate.
	PressurePlateLightWeighted
	// PressurePlateHeavyWeighted is the iron/heavy weighted pressure plate.
	PressurePlateHeavyWeighted
	// PressurePlateOak is the oak pressure plate.
	PressurePlateOak
	// PressurePlateSpruce is the spruce pressure plate.
	PressurePlateSpruce
	// PressurePlateBirch is the birch pressure plate.
	PressurePlateBirch
	// PressurePlateJungle is the jungle pressure plate.
	PressurePlateJungle
	// PressurePlateAcacia is the acacia pressure plate.
	PressurePlateAcacia
	// PressurePlateDarkOak is the dark oak pressure plate.
	PressurePlateDarkOak
	// PressurePlateMangrove is the mangrove pressure plate.
	PressurePlateMangrove
	// PressurePlateCherry is the cherry pressure plate.
	PressurePlateCherry
	// PressurePlatePaleOak is the pale oak pressure plate.
	PressurePlatePaleOak
	// PressurePlatePoplar is the poplar pressure plate.
	PressurePlatePoplar
	// PressurePlateBamboo is the bamboo pressure plate.
	PressurePlateBamboo
	// PressurePlateCrimson is the crimson pressure plate.
	PressurePlateCrimson
	// PressurePlateWarped is the warped pressure plate.
	PressurePlateWarped
	pressurePlateKindCount
)

var pressurePlateBlockNames = [...]string{
	"minecraft:stone_pressure_plate",
	"minecraft:polished_blackstone_pressure_plate",
	"minecraft:light_weighted_pressure_plate",
	"minecraft:heavy_weighted_pressure_plate",
	"minecraft:wooden_pressure_plate",
	"minecraft:spruce_pressure_plate",
	"minecraft:birch_pressure_plate",
	"minecraft:jungle_pressure_plate",
	"minecraft:acacia_pressure_plate",
	"minecraft:dark_oak_pressure_plate",
	"minecraft:mangrove_pressure_plate",
	"minecraft:cherry_pressure_plate",
	"minecraft:pale_oak_pressure_plate",
	"minecraft:poplar_pressure_plate",
	"minecraft:bamboo_pressure_plate",
	"minecraft:crimson_pressure_plate",
	"minecraft:warped_pressure_plate",
}

// PressurePlate is a floor-mounted redstone source. Wooden plates detect all entities, stone plates detect living
// entities, and weighted plates derive their output from the number of entities standing on them.
type PressurePlate struct {
	empty
	transparent
	flowingWaterDisplacer

	// Kind identifies the material and behaviour of the pressure plate.
	Kind int
	// Power is the Bedrock redstone_signal state: 0/1 for ordinary plates and 0-15 for weighted plates.
	Power int
}

var (
	_ world.RedstonePowerSource       = PressurePlate{}
	_ world.RedstoneStrongPowerSource = PressurePlate{}
	_ world.RedstoneWeakBlockPowerer  = PressurePlate{}
	_ EntityStepper                   = PressurePlate{}
	_ world.ScheduledTicker           = PressurePlate{}
)

func (p PressurePlate) RedstonePower(cube.Pos, *world.Tx, cube.Face) int {
	return pressurePlateSignal(p.Kind, p.Power)
}

// RedstoneStrongPower returns strong power only towards the block below the pressure plate.
func (p PressurePlate) RedstoneStrongPower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if face == cube.FaceDown {
		return pressurePlateSignal(p.Kind, p.Power)
	}
	return 0
}

// RedstoneWeaklyPowersBlocks reports whether this plate currently emits a signal that may weakly power adjacent blocks.
func (p PressurePlate) RedstoneWeaklyPowersBlocks() bool {
	return p.Power > 0
}

// EntityStepOn activates the plate when an eligible entity stands on it.
func (p PressurePlate) EntityStepOn(pos cube.Pos, tx *world.Tx, e world.Entity) {
	if p.Kind != PressurePlateLightWeighted && p.Kind != PressurePlateHeavyWeighted {
		if p.acceptsEntity(e) {
			p.setPower(pos, tx, 15)
		}
		return
	}

	count := p.entityCount(tx, pos)
	if count == 0 && p.acceptsEntity(e) {
		count = 1
	}
	p.setPower(pos, tx, pressurePlatePower(p.Kind, count))
}

// ScheduledTick updates the plate after an entity leaves and keeps weighted plates refreshed while occupied.
func (p PressurePlate) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(PressurePlate)
	if !ok {
		return
	}
	power := pressurePlatePower(current.Kind, current.entityCount(tx, pos))
	state := pressurePlateState(current.Kind, power)
	if current.Power != state {
		current.Power = state
		tx.SetBlock(pos, current, nil)
	}
	if power > 0 {
		tx.ScheduleBlockUpdate(pos, current, time.Second/20)
	}
}

func (p PressurePlate) acceptsEntity(e world.Entity) bool {
	switch p.Kind {
	case PressurePlateStone, PressurePlatePolishedBlackstone:
		_, ok := e.(livingEntity)
		return ok
	default:
		return true
	}
}

func (p PressurePlate) entityCount(tx *world.Tx, pos cube.Pos) int {
	box := cube.Box(float64(pos[0]), float64(pos[1]), float64(pos[2]), float64(pos[0]+1), float64(pos[1]+1), float64(pos[2]+1))
	count := 0
	for e := range tx.EntitiesWithin(box) {
		if p.acceptsEntity(e) {
			count++
		}
	}
	return count
}

func (p PressurePlate) setPower(pos cube.Pos, tx *world.Tx, power int) {
	state := pressurePlateState(p.Kind, power)
	if p.Power != state {
		p.Power = state
		tx.SetBlock(pos, p, nil)
	}
	if power > 0 {
		tx.ScheduleBlockUpdate(pos, p, time.Second/20)
	}
}

func pressurePlateState(kind, signal int) int {
	if kind != PressurePlateLightWeighted && kind != PressurePlateHeavyWeighted && signal > 0 {
		return 1
	}
	return signal
}

func pressurePlateSignal(kind, state int) int {
	if kind != PressurePlateLightWeighted && kind != PressurePlateHeavyWeighted && state > 0 {
		return 15
	}
	return state
}

func pressurePlatePower(kind, entities int) int {
	entities = max(entities, 0)
	switch kind {
	case PressurePlateLightWeighted:
		return min(entities, 15)
	case PressurePlateHeavyWeighted:
		return min(entities/10, 15)
	default:
		if entities > 0 {
			return 15
		}
		return 0
	}
}

// NeighbourUpdateTick removes the plate if its supporting block no longer has a solid upper face.
func (p PressurePlate) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	support := pos.Side(cube.FaceDown)
	if !tx.Block(support).Model().FaceSolid(support, cube.FaceUp, tx) {
		breakBlock(p, pos, tx)
	}
}

// UseOnBlock places the plate on the upper face of a supporting block.
func (p PressurePlate) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, face, used := firstReplaceable(tx, pos, face, p)
	if !used || face != cube.FaceUp {
		return false
	}
	support := pos.Side(cube.FaceDown)
	if !tx.Block(support).Model().FaceSolid(support, cube.FaceUp, tx) {
		return false
	}
	p.Power = 0
	place(tx, pos, p, user, ctx)
	return placed(ctx)
}

// SideClosed ...
func (PressurePlate) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// BreakInfo ...
func (p PressurePlate) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, nothingEffective, oneOf(PressurePlate{Kind: p.Kind}))
}

// EncodeItem ...
func (p PressurePlate) EncodeItem() (string, int16) {
	return pressurePlateBlockNames[p.Kind], 0
}

// EncodeBlock ...
func (p PressurePlate) EncodeBlock() (string, map[string]any) {
	return pressurePlateBlockNames[p.Kind], map[string]any{"redstone_signal": int32(p.Power)}
}

// Model ...
func (p PressurePlate) Model() world.BlockModel {
	return model.PressurePlate{Pressed: p.Power > 0}
}

// allPressurePlates returns every vanilla pressure plate state.
func pressurePlateStateCount(kind int) int {
	if kind == PressurePlateLightWeighted || kind == PressurePlateHeavyWeighted {
		return 16
	}
	return 2
}

func allPressurePlates() (all []world.Block) {
	for kind := 0; kind < pressurePlateKindCount; kind++ {
		for power := 0; power < pressurePlateStateCount(kind); power++ {
			all = append(all, PressurePlate{Kind: kind, Power: power})
		}
	}
	return
}
