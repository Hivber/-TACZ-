package session

import (
    "fmt"

    "github.com/df-mc/dragonfly/server/block/cube"
    "github.com/df-mc/dragonfly/server/world"
    "github.com/sandertv/gophertunnel/minecraft/protocol"
    "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// PlayerActionHandler handles the PlayerAction packet.
type PlayerActionHandler struct{}

// Handle ...
func (*PlayerActionHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, c Controllable) error {
    pk := p.(*packet.PlayerAction)
    return handlePlayerAction(pk.ActionType, pk.BlockFace, pk.BlockPosition, pk.EntityRuntimeID, s, c)
}

// handlePlayerAction handles an action performed by a player, found in packet.PlayerAction and packet.PlayerAuthInput.
func handlePlayerAction(action int32, face int32, pos protocol.BlockPos, entityRuntimeID uint64, s *Session, c Controllable) error {
    if entityRuntimeID != selfEntityRuntimeID {
        return errSelfRuntimeID
    }
    blockPos := cube.Pos{int(pos[0]), int(pos[1]), int(pos[2])}
    switch classifyMiningAction(action, blockPos, s.breakingPos) {
    case miningActionStart:
        s.swingingArm.Store(true)
        defer s.swingingArm.Store(false)
        s.breakingPos = blockPos
        c.StartBreaking(blockPos, cube.Face(face))
    case miningActionContinue:
        // StartBreaking aborts and resets the previous animation. CONTINUE
        // packets arrive repeatedly while the button is held and must only
        // advance the existing mining operation.
        s.swingingArm.Store(true)
        defer s.swingingArm.Store(false)
        c.ContinueBreaking(cube.Face(face))
    case miningActionAbort:
        c.AbortBreaking()
    case miningActionFinish:
        s.swingingArm.Store(true)
        defer s.swingingArm.Store(false)
        c.FinishBreaking()
    default:
        switch action {
        case protocol.PlayerActionStartSleeping, protocol.PlayerActionRespawn, protocol.PlayerActionDimensionChangeDone:
            // Don't do anything for these actions.
        case protocol.PlayerActionStopSleeping:
            c.Wake()
        case protocol.PlayerActionCrackBreak:
            // Don't do anything for this action. It is no longer used. Block
            // cracking is done fully server-side.
        case protocol.PlayerActionStartItemUseOn:
            // TODO: Properly utilize these actions.
        case protocol.PlayerActionStopItemUseOn:
            c.ReleaseItem()
        case protocol.PlayerActionInteractWithBlock:
            // Don't do anything for this action.
        case protocol.PlayerActionCreativePlayerDestroyBlock:
            // Don't do anything for this action.
        case protocol.PlayerActionMissedSwing:
            s.swingingArm.Store(true)
            defer s.swingingArm.Store(false)
            c.PunchAir()
        default:
            return fmt.Errorf("unhandled ActionType %v", action)
        }
    }
    return nil
}
