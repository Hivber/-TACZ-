package session

import (
	"github.com/df-mc/dragonfly/server/gamecore"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

func (s *Session) emitGameCorePlayerJoin(c Controllable, dimension world.Dimension) {
	if s.gameCore == nil {
		return
	}
	dimensionID, _ := world.DimensionID(dimension)
	gameModeID, _ := world.GameModeID(c.GameMode())
	yaw, pitch := c.Rotation().Elem()
	pos := c.Position()
	s.gameCore.Enqueue(gamecore.Event{
		Type:      "player_join",
		UUID:      c.UUID().String(),
		Name:      c.Name(),
		Dimension: int32(dimensionID),
		GameMode:  uint8(gameModeID),
		Position: gamecore.Position{
			X: pos[0], Y: pos[1], Z: pos[2],
			Yaw: float32(yaw), Pitch: float32(pitch),
		},
	})
}

func (s *Session) emitGameCorePlayerMove(c Controllable, dimension world.Dimension) {
	if s.gameCore == nil {
		return
	}
	dimensionID, _ := world.DimensionID(dimension)
	yaw, pitch := c.Rotation().Elem()
	pos := c.Position()
	s.gameCore.Enqueue(gamecore.Event{
		Type:      "player_move",
		UUID:      c.UUID().String(),
		Dimension: int32(dimensionID),
		Position: gamecore.Position{
			X: pos[0], Y: pos[1], Z: pos[2],
			Yaw: float32(yaw), Pitch: float32(pitch),
		},
	})
}

func (s *Session) emitGameCorePlayerLeave(c Controllable) {
	if s.gameCore == nil {
		return
	}
	s.gameCore.Enqueue(gamecore.Event{
		Type: "player_leave",
		UUID: c.UUID().String(),
	})
}

// emitGameCoreInventorySnapshot mirrors the player's persistent carried slots:
// main inventory, off-hand, then armour. Dragonfly remains authoritative until
// item metadata and transaction handling have been fully migrated.
func (s *Session) emitGameCoreInventorySnapshot(c Controllable) {
	if s.gameCore == nil || s.inv == nil || s.offHand == nil || s.enderChest == nil || s.armour == nil {
		return
	}
	snapshot := make([]*gamecore.ItemStack, 0, s.inv.Size()+s.offHand.Size()+s.enderChest.Size()+s.armour.Inventory().Size())
	appendSlots := func(stacks []item.Stack) error {
		for _, stack := range stacks {
			encoded, err := gamecore.EncodeItemStack(stack)
			if err != nil {
				return err
			}
			snapshot = append(snapshot, encoded)
		}
		return nil
	}
	if err := appendSlots(s.inv.Slots()); err != nil {
		return
	}
	if err := appendSlots(s.offHand.Slots()); err != nil {
		return
	}
	if err := appendSlots(s.enderChest.Slots()); err != nil {
		return
	}
	if err := appendSlots(s.armour.Inventory().Slots()); err != nil {
		return
	}
	s.gameCore.Enqueue(gamecore.Event{
		Type:  "inventory_sync",
		UUID:  c.UUID().String(),
		Slots: snapshot,
	})
}
