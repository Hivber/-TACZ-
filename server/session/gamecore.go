package session

import (
	"context"
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/gamecore"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
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

// playerCarriedInventoryID is stable across transient Bedrock window IDs.
func playerCarriedInventoryID(c Controllable) string {
	return "player:" + c.UUID().String() + ":carried"
}

// encodeGameCoreSlots preserves every slot, including empty positions.
func encodeGameCoreSlots(stacks []item.Stack) ([]*gamecore.ItemStack, error) {
	snapshot := make([]*gamecore.ItemStack, 0, len(stacks))
	for _, stack := range stacks {
		encoded, err := gamecore.EncodeItemStack(stack)
		if err != nil {
			return nil, err
		}
		snapshot = append(snapshot, encoded)
	}
	return snapshot, nil
}

// gameCorePlayerInventorySnapshot returns the fixed carried-slot order:
// main inventory, off-hand, ender chest, then armour.
func (s *Session) gameCorePlayerInventorySnapshot() ([]*gamecore.ItemStack, error) {
	if s.inv == nil || s.offHand == nil || s.enderChest == nil || s.armour == nil {
		return nil, fmt.Errorf("player inventory is not fully initialized")
	}
	stacks := make([]item.Stack, 0, s.inv.Size()+s.offHand.Size()+s.enderChest.Size()+s.armour.Inventory().Size())
	stacks = append(stacks, s.inv.Slots()...)
	stacks = append(stacks, s.offHand.Slots()...)
	stacks = append(stacks, s.enderChest.Slots()...)
	stacks = append(stacks, s.armour.Inventory().Slots()...)
	return encodeGameCoreSlots(stacks)
}

// emitGameCoreInventorySnapshot mirrors the player's persistent carried slots.
// Dragonfly remains authoritative until item transaction handling is migrated.
func (s *Session) emitGameCoreInventorySnapshot(c Controllable) {
	if s.gameCore == nil {
		return
	}
	snapshot, err := s.gameCorePlayerInventorySnapshot()
	if err != nil {
		return
	}
	s.gameCore.Enqueue(gamecore.Event{
		Type: "inventory_sync", InventoryID: playerCarriedInventoryID(c), Slots: snapshot,
	})
}

// emitGameCoreWorldContainerSnapshot mirrors a persistent block-owned inventory.
// The key is independent of the transient Bedrock window ID and player session.
func (s *Session) emitGameCoreWorldContainerSnapshot(inventoryID string, inv *inventory.Inventory) {
	if s.gameCore == nil || inventoryID == "" || inv == nil {
		return
	}
	snapshot := make([]*gamecore.ItemStack, 0, inv.Size())
	for _, stack := range inv.Slots() {
		encoded, err := gamecore.EncodeItemStack(stack)
		if err != nil {
			return
		}
		snapshot = append(snapshot, encoded)
	}
	s.gameCore.Enqueue(gamecore.Event{
		Type:        "inventory_sync",
		InventoryID: inventoryID,
		Slots:       snapshot,
	})
}

// gameCoreInventoryID maps a live Dragonfly inventory to its stable Rust mirror.
// Crafting/UI inventories are intentionally excluded because they are transient.
func (s *Session) gameCoreInventoryID(inv *inventory.Inventory, c Controllable) string {
	if inv == nil {
		return ""
	}
	if inv == s.inv || inv == s.offHand || inv == s.enderChest || (s.armour != nil && inv == s.armour.Inventory()) {
		return playerCarriedInventoryID(c)
	}
	if inv == s.openedWindow.Load() && s.openedInventoryID != "" {
		return s.openedInventoryID
	}
	return ""
}

// gameCoreInventorySnapshot encodes the canonical snapshot for a mapped inventory.
func (s *Session) gameCoreInventorySnapshot(id string, inv *inventory.Inventory, c Controllable) ([]*gamecore.ItemStack, error) {
	if id == playerCarriedInventoryID(c) {
		return s.gameCorePlayerInventorySnapshot()
	}
	if inv == nil {
		return nil, fmt.Errorf("inventory %q is nil", id)
	}
	return encodeGameCoreSlots(inv.Slots())
}

// emitGameCoreInventoryPair publishes both sides of an accepted slot transfer or swap
// in one Rust IPC event. Per-slot callbacks may already have queued intermediate
// snapshots; this final pair atomically replaces both mirrors after the Go transaction.
func (s *Session) emitGameCoreInventoryPair(first, second *inventory.Inventory, c Controllable) {
	if s.gameCore == nil {
		return
	}
	firstID := s.gameCoreInventoryID(first, c)
	secondID := s.gameCoreInventoryID(second, c)
	if firstID == "" || secondID == "" {
		return
	}
	firstSlots, err := s.gameCoreInventorySnapshot(firstID, first, c)
	if err != nil {
		return
	}
	if firstID == secondID {
		s.gameCore.Enqueue(gamecore.Event{Type: "inventory_sync", InventoryID: firstID, Slots: firstSlots})
		return
	}
	secondSlots, err := s.gameCoreInventorySnapshot(secondID, second, c)
	if err != nil {
		return
	}
	s.gameCore.Enqueue(gamecore.Event{
		Type:               "inventory_sync_pair",
		InventoryID:        firstID,
		Slots:              firstSlots,
		RelatedInventoryID: secondID,
		RelatedSlots:       secondSlots,
	})
}

// worldContainerInventoryID identifies a block container by dimension and block
// coordinates. It deliberately excludes the Bedrock window ID and viewer UUID.
func worldContainerInventoryID(dimension world.Dimension, pos cube.Pos) string {
	dimensionID, ok := world.DimensionID(dimension)
	if !ok {
		return ""
	}
	return fmt.Sprintf("world-container:%d:%d:%d:%d", dimensionID, pos[0], pos[1], pos[2])
}


// GameCoreMiningAction synchronously asks the optional Rust gameplay core to
// advance one player's mining state. It is deliberately narrow: packet
// decoding and world/block mutation remain with the Go Bedrock adapter.
func (s *Session) GameCoreMiningAction(event gamecore.Event) (gamecore.MiningResult, bool) {
	if s == nil || s.gameCore == nil {
		return gamecore.MiningResult{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	response, err := s.gameCore.Call(ctx, event)
	if err != nil || response.ProcessedEvents != 1 || len(response.MiningResults) != 1 {
		return gamecore.MiningResult{}, false
	}
	result := response.MiningResults[0]
	if result.UUID != event.UUID {
		return gamecore.MiningResult{}, false
	}
	return result, true
}
