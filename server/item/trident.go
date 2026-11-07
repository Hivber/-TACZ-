package item

// Trident is a durable melee weapon. Its thrown projectile and trident-specific
// enchantments require additional entity and enchantment systems.
type Trident struct{}

// AttackDamage returns the Bedrock melee attack damage of a trident.
func (Trident) AttackDamage() float64 { return 9 }

// MaxCount always returns 1.
func (Trident) MaxCount() int { return 1 }

// HandEquipped reports that the trident is held like a weapon.
func (Trident) HandEquipped() bool { return true }

// EnchantmentValue returns the trident's enchantability.
func (Trident) EnchantmentValue() int { return 1 }

// DurabilityInfo returns the trident's durability settings.
func (Trident) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability:    251,
		BrokenItem:       simpleItem(Stack{}),
		AttackDurability: 1,
	}
}

// EncodeItem ...
func (Trident) EncodeItem() (name string, meta int16) {
	return "minecraft:trident", 0
}
