package session

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type miningAction uint8

const (
	miningActionNone miningAction = iota
	miningActionStart
	miningActionContinue
	miningActionAbort
	miningActionFinish
)

// classifyMiningAction keeps repeated CONTINUE_DESTROY_BLOCK packets from
// restarting the mining timer. A changed position is treated as a fresh start
// because some clients can switch targets without an intervening START packet.
func classifyMiningAction(action int32, pos, current cube.Pos, active bool) miningAction {
	switch action {
	case protocol.PlayerActionStartBreak:
		return miningActionStart
	case protocol.PlayerActionContinueDestroyBlock:
		if !active || pos != current {
			return miningActionStart
		}
		return miningActionContinue
	case protocol.PlayerActionAbortBreak, protocol.PlayerActionStopBreak:
		return miningActionAbort
	case protocol.PlayerActionPredictDestroyBlock:
		return miningActionFinish
	default:
		return miningActionNone
	}
}
