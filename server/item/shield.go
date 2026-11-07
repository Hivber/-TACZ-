package item

// Shield is an off-hand defensive item. In Bedrock Edition, crouching while
// holding a shield in the off-hand activates its frontal blocking behaviour.
type Shield struct{}

// MaxCount always returns 1.
func (Shield) MaxCount() int { return 1 }

// OffHand reports that the shield may be equipped in the off-hand slot.
func (Shield) OffHand() bool { return true }

// HandEquipped reports that the shield uses the hand-held item orientation.
func (Shield) HandEquipped() bool { return true }

// DurabilityInfo returns the shield's durability settings.
func (Shield) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability: 336,
		BrokenItem:    simpleItem(Stack{}),
	}
}

// EncodeItem ...
func (Shield) EncodeItem() (name string, meta int16) {
	return "minecraft:shield", 0
}
