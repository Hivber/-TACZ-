package gamecore

import (
	"encoding/json"
	"fmt"

	"github.com/df-mc/dragonfly/server/item"
)

// EncodeItemStack converts a Dragonfly stack into the protocol-independent
// Rust mirror representation. UserData contains a versioned JSON envelope
// around Dragonfly's complete disk-NBT item payload, preserving display data,
// durability, enchantments, repair cost, unbreakable state, item-specific NBT,
// and Dragonfly custom values without making Rust depend on Dragonfly types.
func EncodeItemStack(stack item.Stack) (*ItemStack, error) {
	if stack.Empty() {
		return nil, nil
	}
	it := stack.Item()
	if it == nil {
		return nil, fmt.Errorf("non-empty stack has no item")
	}
	count := stack.Count()
	if count <= 0 || count > stack.MaxCount() || count > 65535 {
		return nil, fmt.Errorf("invalid count %d for item stack", count)
	}
	identifier, metadata := it.EncodeItem()
	if identifier == "" {
		return nil, fmt.Errorf("item has an empty identifier")
	}

	payload, err := json.Marshal(struct {
		Format string         `json:"format"`
		NBT    map[string]any `json:"nbt"`
	}{
		Format: "dragonfly-item-nbt-json-v1",
		NBT:    item.WriteNBT(stack, true),
	})
	if err != nil {
		return nil, fmt.Errorf("encode item NBT: %w", err)
	}
	if len(payload) > 65536 {
		return nil, fmt.Errorf("encoded item data exceeds 65536 bytes")
	}
	userData := make([]uint16, len(payload))
	for i, value := range payload {
		userData[i] = uint16(value)
	}
	return &ItemStack{
		Identifier: identifier,
		Count:      uint16(count),
		Metadata:   uint32(uint16(metadata)),
		UserData:   userData,
	}, nil
}
