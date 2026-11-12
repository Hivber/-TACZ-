package session

import (
	"github.com/df-mc/dragonfly/server/gamecore"
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


// emitGameCoreInventorySnapshot mirrors the player's primary inventory (hotbar
// and main storage) to Rust. Dragonfly remains authoritative until item-stack
// metadata and transaction handling have been fully migrated.
func (s *Session) emitGameCoreInventorySnapshot(c Controllable) {
	if s.gameCore == nil || s.inv == nil {
		return
	}
	slots := s.inv.Slots()
	snapshot := make([]*gamecore.ItemStack, len(slots))
	for slot, stack := range slots {
		if stack.Empty() {
			continue
		}
		it := stack.Item()
		if it == nil {
			continue
		}
		identifier, metadata := it.EncodeItem()
		count := stack.Count()
		if count <= 0 || count > 65535 {
			continue
		}
		snapshot[slot] = &gamecore.ItemStack{
			Identifier: identifier,
			Count:      uint16(count),
			Metadata:   uint32(uint16(metadata)),
		}
	}
	s.gameCore.Enqueue(gamecore.Event{
		Type:  "inventory_sync",
		UUID:  c.UUID().String(),
		Slots: snapshot,
	})
}
