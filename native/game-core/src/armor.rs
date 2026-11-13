//! Armor loadout aggregation adapted from MC-RS armor/armor_tier logic.
//! The item registry and packet ItemStack dependency are intentionally replaced
//! with identifier-independent slot/material inputs.
//! Upstream: https://github.com/pedrokarim/mc-rs @ da93a9501601fcdc284762fde88c24b44604991e
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).

use crate::armor_tier::{ArmorSlot, ArmorTier};

#[derive(Clone, Copy, Debug, Default, Eq, PartialEq)]
pub struct ProtectionEnchantments {
    pub protection: u32,
    pub fire_protection: u32,
    pub blast_protection: u32,
    pub projectile_protection: u32,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct ArmorPiece {
    pub material: ArmorTier,
    pub slot: ArmorSlot,
    pub enchantments: ProtectionEnchantments,
}

#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct ArmorStats {
    pub defense_points: u32,
    pub toughness: f32,
    pub knockback_resistance: f32,
    pub enchantments: ProtectionEnchantments,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct ArmorLoadout {
    slots: [Option<ArmorPiece>; 4],
}

impl Default for ArmorLoadout {
    fn default() -> Self { Self::empty() }
}

impl ArmorLoadout {
    pub const fn empty() -> Self { Self { slots: [None; 4] } }

    pub fn equip(&mut self, piece: ArmorPiece) -> Result<Option<ArmorPiece>, String> {
        let index = piece.slot.index();
        let previous = self.slots[index].replace(piece);
        Ok(previous)
    }

    pub fn unequip(&mut self, slot: ArmorSlot) -> Option<ArmorPiece> {
        self.slots[slot.index()].take()
    }

    pub fn get(&self, slot: ArmorSlot) -> Option<&ArmorPiece> {
        self.slots[slot.index()].as_ref()
    }

    pub fn stats(&self) -> ArmorStats {
        let mut stats = ArmorStats::default();
        for piece in self.slots.iter().flatten() {
            stats.defense_points = stats.defense_points.saturating_add(
                u32::from(piece.material.defense(piece.slot)),
            );
            stats.toughness += piece.material.toughness();
            stats.knockback_resistance += piece.material.knockback_resistance();
            stats.enchantments.protection =
                stats.enchantments.protection.saturating_add(piece.enchantments.protection);
            stats.enchantments.fire_protection = stats.enchantments.fire_protection
                .saturating_add(piece.enchantments.fire_protection);
            stats.enchantments.blast_protection = stats.enchantments.blast_protection
                .saturating_add(piece.enchantments.blast_protection);
            stats.enchantments.projectile_protection = stats.enchantments.projectile_protection
                .saturating_add(piece.enchantments.projectile_protection);
        }
        stats.knockback_resistance = stats.knockback_resistance.clamp(0.0, 1.0);
        stats
    }
}

impl ArmorSlot {
    pub const fn index(self) -> usize {
        match self {
            Self::Helmet => 0,
            Self::Chestplate => 1,
            Self::Leggings => 2,
            Self::Boots => 3,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn full_netherite_set_aggregates_defence_toughness_and_knockback() {
        let mut loadout = ArmorLoadout::empty();
        for slot in [ArmorSlot::Helmet, ArmorSlot::Chestplate, ArmorSlot::Leggings, ArmorSlot::Boots] {
            loadout.equip(ArmorPiece {
                material: ArmorTier::Netherite,
                slot,
                enchantments: ProtectionEnchantments::default(),
            }).unwrap();
        }
        let stats = loadout.stats();
        assert_eq!(stats.defense_points, 20);
        assert_eq!(stats.toughness, 12.0);
        assert!((stats.knockback_resistance - 0.4).abs() < f32::EPSILON);
    }

    #[test]
    fn armor_enchantments_sum_across_slots() {
        let mut loadout = ArmorLoadout::empty();
        loadout.equip(ArmorPiece {
            material: ArmorTier::Iron,
            slot: ArmorSlot::Helmet,
            enchantments: ProtectionEnchantments { protection: 2, fire_protection: 1, ..Default::default() },
        }).unwrap();
        loadout.equip(ArmorPiece {
            material: ArmorTier::Iron,
            slot: ArmorSlot::Chestplate,
            enchantments: ProtectionEnchantments { protection: 3, ..Default::default() },
        }).unwrap();
        let stats = loadout.stats();
        assert_eq!(stats.enchantments.protection, 5);
        assert_eq!(stats.enchantments.fire_protection, 1);
    }

    #[test]
    fn replacing_a_slot_returns_the_previous_piece() {
        let mut loadout = ArmorLoadout::empty();
        let first = ArmorPiece { material: ArmorTier::Leather, slot: ArmorSlot::Helmet, enchantments: Default::default() };
        let second = ArmorPiece { material: ArmorTier::Diamond, slot: ArmorSlot::Helmet, enchantments: Default::default() };
        assert_eq!(loadout.equip(first).unwrap(), None);
        assert_eq!(loadout.equip(second).unwrap(), Some(first));
        assert_eq!(loadout.get(ArmorSlot::Helmet), Some(&second));
    }
}
