package block

import (
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// CommandBlockMode identifies the execution mode of a Bedrock command block.
type CommandBlockMode uint8

const (
	// CommandBlockImpulse executes once when it receives a rising redstone edge.
	CommandBlockImpulse CommandBlockMode = iota
	// CommandBlockRepeating executes at the configured interval while active.
	CommandBlockRepeating
	// CommandBlockChain executes after the command block in front of it.
	CommandBlockChain
)

// CommandBlock is a vanilla Bedrock command block. Its command and execution state
// are saved as block actor NBT and are synchronised to viewers through BlockActorData.
type CommandBlock struct {
	solid

	Mode                CommandBlockMode
	Facing              cube.Face
	Conditional         bool
	Command             string
	CustomName          string
	LastOutput          string
	TrackOutput         bool
	TickDelay           uint32
	NeedsRedstone       bool
	Powered             bool
	ConditionMet        bool
	ExecuteOnFirstTick  bool
	LastExecutionTick   int64
	SuccessCount        int32
	HasExecuted         bool
}

// NewCommandBlock constructs a command block with Bedrock's initial settings.
func NewCommandBlock(mode CommandBlockMode) CommandBlock {
	return CommandBlock{
		Mode:               mode,
		Facing:             cube.FaceNorth,
		TrackOutput:        true,
		NeedsRedstone:      mode != CommandBlockChain,
		ExecuteOnFirstTick: true,
	}
}

// allCommandBlocks returns every block-state permutation registered by the server.
func allCommandBlocks() []world.Block {
	blocks := make([]world.Block, 0, 3*6*2)
	for mode := CommandBlockImpulse; mode <= CommandBlockChain; mode++ {
		for _, face := range cube.Faces() {
			for _, conditional := range []bool{false, true} {
				b := NewCommandBlock(mode)
				b.Facing, b.Conditional = face, conditional
				blocks = append(blocks, b)
			}
		}
	}
	return blocks
}

// EncodeItem ...
func (b CommandBlock) EncodeItem() (string, int16) {
	return commandBlockID(b.Mode), 0
}

// EncodeBlock ...
func (b CommandBlock) EncodeBlock() (string, map[string]any) {
	return commandBlockID(b.Mode), map[string]any{
		"conditional_bit":  b.Conditional,
		"facing_direction": int32(b.Facing),
	}
}

func commandBlockID(mode CommandBlockMode) string {
	switch mode {
	case CommandBlockRepeating:
		return "minecraft:repeating_command_block"
	case CommandBlockChain:
		return "minecraft:chain_command_block"
	default:
		return "minecraft:command_block"
	}
}

// DecodeNBT ...
func (b CommandBlock) DecodeNBT(data map[string]any) any {
	b.Command = nbtconv.String(data, "Command")
	b.CustomName = nbtconv.String(data, "CustomName")
	b.LastOutput = nbtconv.String(data, "LastOutput")
	b.SuccessCount = nbtconv.Int32(data, "SuccessCount")
	b.LastExecutionTick = nbtconv.Int64(data, "LastExecution")
	b.TickDelay = uint32(max(nbtconv.Int32(data, "TickDelay"), 0))
	b.Powered = nbtconv.Bool(data, "powered")
	b.ConditionMet = nbtconv.Bool(data, "conditionMet")
	b.HasExecuted = nbtconv.Bool(data, "hasExecuted")
	if _, ok := data["TrackOutput"]; ok {
		b.TrackOutput = nbtconv.Bool(data, "TrackOutput")
	} else {
		b.TrackOutput = true
	}
	if _, ok := data["auto"]; ok {
		b.NeedsRedstone = !nbtconv.Bool(data, "auto")
	} else {
		b.NeedsRedstone = b.Mode != CommandBlockChain
	}
	if _, ok := data["conditionalMode"]; ok {
		b.Conditional = nbtconv.Bool(data, "conditionalMode")
	}
	if _, ok := data["executeOnFirstTick"]; ok {
		b.ExecuteOnFirstTick = nbtconv.Bool(data, "executeOnFirstTick")
	} else {
		b.ExecuteOnFirstTick = true
	}
	return b
}

// EncodeNBT ...
func (b CommandBlock) EncodeNBT() map[string]any {
	return map[string]any{
		"id":                  "CommandBlock",
		"Command":             b.Command,
		"CustomName":          b.CustomName,
		"LastOutput":          b.LastOutput,
		"TrackOutput":         b.TrackOutput,
		"TickDelay":           int32(b.TickDelay),
		"auto":                !b.NeedsRedstone,
		"powered":             b.Powered,
		"conditionMet":        b.ConditionMet,
		"conditionalMode":     b.Conditional,
		"executeOnFirstTick":  b.ExecuteOnFirstTick,
		"LastExecution":       b.LastExecutionTick,
		"SuccessCount":        b.SuccessCount,
		"LPCommandMode":       uint32(b.Mode),
		"hasExecuted":         b.HasExecuted,
	}
}

// Activate tells the Bedrock client to open its native command block editor.
func (b CommandBlock) Activate(cube.Pos, cube.Face, *world.Tx, item.User, *item.UseContext) bool {
	return true
}

// UseOnBlock places a command block facing the direction the player is looking.
func (b CommandBlock) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, b)
	if !used {
		return false
	}
	b.Facing = user.Rotation().Direction().Face()
	place(tx, pos, b, user, ctx)
	return placed(ctx)
}

// RedstonePowerUpdate records the current power state for the command block.
func (b CommandBlock) RedstonePowerUpdate(_ cube.Pos, _ *world.Tx, power int) (world.Block, bool) {
	powered := power > 0
	if b.Powered == powered {
		return b, false
	}
	b.Powered = powered
	return b, true
}

// RedstonePowerAction executes impulse command blocks only on a rising edge.
func (b CommandBlock) RedstonePowerAction(pos cube.Pos, tx *world.Tx, oldPower, newPower int) {
	if b.Mode != CommandBlockImpulse || !b.NeedsRedstone || oldPower > 0 || newPower == 0 {
		return
	}
	if !tx.World().GameRuleBool("commandblocksenabled", true) {
		return
	}
	b.Powered = true
	b.HasExecuted = true
	tx.SetBlockEntity(pos, b)
	if commandBlockExecutor != nil {
		commandBlockExecutor(pos, b, tx)
	}
}

// Tick runs always-active impulse command blocks once and repeating command blocks at their configured interval.
func (b CommandBlock) Tick(currentTick int64, pos cube.Pos, tx *world.Tx) {
	if !tx.World().GameRuleBool("commandblocksenabled", true) {
		return
	}
	switch b.Mode {
	case CommandBlockImpulse:
		if b.NeedsRedstone || b.HasExecuted || !b.ExecuteOnFirstTick {
			return
		}
		b.HasExecuted = true
		tx.SetBlockEntity(pos, b)
		if commandBlockExecutor != nil {
			commandBlockExecutor(pos, b, tx)
		}
	case CommandBlockRepeating:
		if b.NeedsRedstone && !b.Powered {
			return
		}
		delay := int64(max(b.TickDelay, 1))
		if b.LastExecutionTick == 0 && !b.ExecuteOnFirstTick {
			b.LastExecutionTick = currentTick
			tx.SetBlockEntity(pos, b)
			return
		}
		if b.LastExecutionTick != 0 && currentTick-b.LastExecutionTick < delay {
			return
		}
		b.LastExecutionTick = currentTick
		tx.SetBlockEntity(pos, b)
		if commandBlockExecutor != nil {
			commandBlockExecutor(pos, b, tx)
		}
	}
}

// CommandBlockExecutor executes a command block and its following chain blocks.
// The callback is registered by the cmd package to avoid an import cycle from block to cmd.
type CommandBlockExecutor func(pos cube.Pos, commandBlock CommandBlock, tx *world.Tx) bool

var commandBlockExecutor CommandBlockExecutor

// RegisterCommandBlockExecutor installs the server's command block execution callback.
// It is called once during package initialisation by the cmd package.
func RegisterCommandBlockExecutor(executor CommandBlockExecutor) {
	commandBlockExecutor = executor
}

// SetCommandBlockOutput stores output from a command execution and returns the updated block.
func SetCommandBlockOutput(b CommandBlock, success bool, output string) CommandBlock {
	if success {
		b.SuccessCount = 1
	} else {
		b.SuccessCount = 0
	}
	b.ConditionMet = success
	b.LastOutput = strings.TrimSpace(output)
	return b
}
