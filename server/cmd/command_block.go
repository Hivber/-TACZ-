package cmd

import (
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func init() {
	block.RegisterCommandBlockExecutor(executeCommandBlockChain)
}

type commandBlockSource struct {
	position mgl64.Vec3
	messages []string
	errors   []string
}

func (s *commandBlockSource) Position() mgl64.Vec3 { return s.position }

func (s *commandBlockSource) SendCommandOutput(output *Output) {
	for _, message := range output.Messages() {
		s.messages = append(s.messages, message.String())
	}
	for _, err := range output.Errors() {
		s.errors = append(s.errors, err.Error())
	}
}

func (s *commandBlockSource) outputText() string {
	parts := make([]string, 0, len(s.messages)+len(s.errors))
	parts = append(parts, s.messages...)
	parts = append(parts, s.errors...)
	return strings.Join(parts, "\n")
}

// executeCommandBlockChain runs the source block and contiguous chain blocks.
func executeCommandBlockChain(pos cube.Pos, current block.CommandBlock, tx *world.Tx) bool {
	if !tx.World().GameRuleBool("commandblocksenabled", true) {
		return false
	}
	limit := maxCommandChainLength(tx)
	success, updated := executeCommandBlock(pos, current, tx)
	tx.SetBlock(pos, updated, &world.SetOpts{DisableBlockUpdates: true, DisableRedstoneUpdates: true})

	lastPos, last := pos, updated
	for i := 0; i < limit; i++ {
		nextPos := lastPos.Side(last.Facing)
		next, ok := tx.Block(nextPos).(block.CommandBlock)
		if !ok || next.Mode != block.CommandBlockChain || next.Facing != last.Facing {
			break
		}
		if next.NeedsRedstone && !next.Powered {
			break
		}
		if next.Conditional && !success {
			next.SuccessCount = 0
			next.ConditionMet = false
			next.LastOutput = ""
			tx.SetBlock(nextPos, next, &world.SetOpts{DisableBlockUpdates: true, DisableRedstoneUpdates: true})
			success = false
			lastPos, last = nextPos, next
			continue
		}
		success, next = executeCommandBlock(nextPos, next, tx)
		tx.SetBlock(nextPos, next, &world.SetOpts{DisableBlockUpdates: true, DisableRedstoneUpdates: true})
		lastPos, last = nextPos, next
	}
	return success
}

func maxCommandChainLength(tx *world.Tx) int {
	const hardLimit = 65536
	value, ok := tx.World().GameRule("maxcommandchainlength")
	if !ok {
		return hardLimit
	}
	switch limit := value.(type) {
	case uint32:
		return min(int(limit), hardLimit)
	case int:
		return min(limit, hardLimit)
	default:
		return hardLimit
	}
}

func executeCommandBlock(pos cube.Pos, b block.CommandBlock, tx *world.Tx) (bool, block.CommandBlock) {
	commandLine := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(b.Command), "/"))
	if commandLine == "" || !utf8.ValidString(commandLine) || len(commandLine) > 32767 {
		b.SuccessCount, b.ConditionMet, b.LastOutput = 0, false, ""
		return false, b
	}
	name, args := commandLine, ""
	if i := strings.IndexAny(commandLine, " \t"); i >= 0 {
		name = commandLine[:i]
		args = strings.TrimSpace(commandLine[i+1:])
	}
	command, ok := ByAlias(strings.ToLower(name))
	source := &commandBlockSource{position: pos.Vec3Centre()}
	if !ok {
		output := &Output{}
		output.Errort(MessageUnknown, name)
		source.SendCommandOutput(output)
	} else {
		command.Execute(args, source, tx)
	}
	success := len(source.errors) == 0
	output := ""
	if b.TrackOutput && tx.World().GameRuleBool("commandblockoutput", true) {
		output = source.outputText()
		if len(output) > 4096 {
			output = output[:4096]
		}
	}
	if success {
		b.SuccessCount = 1
	} else {
		b.SuccessCount = 0
	}
	b.ConditionMet, b.LastOutput = success, output
	return success, b
}
