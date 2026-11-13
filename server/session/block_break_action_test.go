package session

import (
    "testing"

    "github.com/df-mc/dragonfly/server/block/cube"
    "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestClassifyMiningActionDoesNotRestartRepeatedContinue(t *testing.T) {
    current := cube.Pos{12, 64, -8}
    tests := []struct {
        name string
        action int32
        pos cube.Pos
        want miningAction
    }{
        {"start", protocol.PlayerActionStartBreak, current, miningActionStart},
        {"continue same target", protocol.PlayerActionContinueDestroyBlock, current, miningActionContinue},
        {"continue changed target starts new mining", protocol.PlayerActionContinueDestroyBlock, cube.Pos{13, 64, -8}, miningActionStart},
        {"abort", protocol.PlayerActionAbortBreak, current, miningActionAbort},
        {"predicted finish", protocol.PlayerActionPredictDestroyBlock, current, miningActionFinish},
        {"stop finish", protocol.PlayerActionStopBreak, current, miningActionFinish},
        {"unrelated action", protocol.PlayerActionMissedSwing, current, miningActionNone},
    }
    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            if got := classifyMiningAction(test.action, test.pos, current); got != test.want {
                t.Fatalf("classifyMiningAction() = %v, want %v", got, test.want)
            }
        })
    }
}
