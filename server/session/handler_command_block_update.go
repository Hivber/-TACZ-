package session

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// CommandBlockUpdateHandler handles edits submitted by the native Bedrock command
// block editor. Only operators within interaction range may change a block.
type CommandBlockUpdateHandler struct{}

// Handle ...
func (*CommandBlockUpdateHandler) Handle(p packet.Packet, _ *Session, tx *world.Tx, c Controllable) error {
	pk := p.(*packet.CommandBlockUpdate)
	if !pk.Block {
		// Command block minecarts are not implemented by Dragonfly.
		return nil
	}
	if tx == nil {
		return fmt.Errorf("command block update without a world transaction")
	}
	if operator, ok := c.(interface{ Operator() bool }); !ok || !operator.Operator() {
		return fmt.Errorf("operator permission is required to edit command blocks")
	}
	pos := cube.Pos{int(pk.Position[0]), int(pk.Position[1]), int(pk.Position[2])}
	if !canReach(c, pos.Vec3Middle()) {
		return fmt.Errorf("command block at %v is not within reach", pos)
	}
	if !tx.World().GameRuleBool("commandblocksenabled", true) {
		return fmt.Errorf("command blocks are disabled by the commandblocksenabled game rule")
	}
	if pk.Mode > packet.CommandBlockChain {
		return fmt.Errorf("invalid command block mode %d", pk.Mode)
	}
	if !utf8.ValidString(pk.Command) || len(pk.Command) > 32767 {
		return fmt.Errorf("command text is invalid or exceeds 32767 bytes")
	}
	name := strings.TrimSpace(pk.Name)
	if !utf8.ValidString(name) || len(name) > 256 {
		return fmt.Errorf("command block name is invalid or too long")
	}
	if pk.TickDelay > 1_000_000 {
		return fmt.Errorf("command block tick delay exceeds 1000000 ticks")
	}

	current, ok := tx.Block(pos).(block.CommandBlock)
	if !ok {
		return fmt.Errorf("no command block exists at %v", pos)
	}
	current.Mode = block.CommandBlockMode(pk.Mode)
	current.NeedsRedstone = pk.NeedsRedstone
	current.Conditional = pk.Conditional
	current.Command = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(pk.Command), "/"))
	current.CustomName = name
	current.TrackOutput = pk.ShouldTrackOutput
	current.TickDelay = pk.TickDelay
	current.ExecuteOnFirstTick = pk.ExecuteOnFirstTick
	current.HasExecuted = false
	current.LastExecutionTick = 0
	current.SuccessCount = 0
	current.ConditionMet = false
	current.LastOutput = ""

	tx.SetBlock(pos, current, &world.SetOpts{DisableBlockUpdates: true, DisableRedstoneUpdates: true})
	return nil
}
