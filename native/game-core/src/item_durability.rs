//! Identifier-based durable tool handling adapted from MC-RS durability.rs.
//! Upstream: https://github.com/pedrokarim/mc-rs @ da93a9501601fcdc284762fde88c24b44604991e
//! MIT. This version removes MC-RS packet ItemStack and network item-ID imports.
//! For this bridge model, ItemStack.metadata stores damage for durable tools;
//! callers must preserve the game's item metadata convention at the Go adapter.

use crate::item_stack::ItemStack;
use crate::random::RandomSource;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ToolTier { Wood, Stone, Gold, Iron, Diamond, Netherite }

impl ToolTier {
    pub const fn max_durability(self) -> u32 {
        match self {
            Self::Wood => 59, Self::Stone => 131, Self::Gold => 32,
            Self::Iron => 250, Self::Diamond => 1561, Self::Netherite => 2031,
        }
    }

    pub const fn mining_tier(self) -> u8 {
        match self {
            Self::Wood | Self::Gold => 0, Self::Stone => 1, Self::Iron => 2,
            Self::Diamond => 3, Self::Netherite => 4,
        }
    }

    pub const fn base_attack_points(self) -> u32 {
        match self {
            Self::Wood | Self::Gold => 2, Self::Stone => 3, Self::Iron => 4,
            Self::Diamond => 5, Self::Netherite => 6,
        }
    }

    pub const fn mining_speed(self) -> f32 {
        match self {
            Self::Wood => 2.0, Self::Stone => 4.0, Self::Iron => 6.0,
            Self::Gold => 12.0, Self::Diamond => 8.0, Self::Netherite => 9.0,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ToolType { Pickaxe, Axe, Shovel, Hoe, Sword, Shears, Armor }

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DurableInfo {
    pub tier: ToolTier,
    pub tool_type: ToolType,
}

pub fn durable_info(identifier: &str) -> Option<DurableInfo> {
    use ToolTier::*;
    use ToolType::*;
    let (tier, tool_type) = match identifier {
        "minecraft:wooden_pickaxe" => (Wood, Pickaxe),
        "minecraft:stone_pickaxe" => (Stone, Pickaxe),
        "minecraft:iron_pickaxe" => (Iron, Pickaxe),
        "minecraft:golden_pickaxe" => (Gold, Pickaxe),
        "minecraft:diamond_pickaxe" => (Diamond, Pickaxe),
        "minecraft:netherite_pickaxe" => (Netherite, Pickaxe),
        "minecraft:wooden_axe" => (Wood, Axe),
        "minecraft:stone_axe" => (Stone, Axe),
        "minecraft:iron_axe" => (Iron, Axe),
        "minecraft:golden_axe" => (Gold, Axe),
        "minecraft:diamond_axe" => (Diamond, Axe),
        "minecraft:netherite_axe" => (Netherite, Axe),
        "minecraft:wooden_shovel" => (Wood, Shovel),
        "minecraft:stone_shovel" => (Stone, Shovel),
        "minecraft:iron_shovel" => (Iron, Shovel),
        "minecraft:golden_shovel" => (Gold, Shovel),
        "minecraft:diamond_shovel" => (Diamond, Shovel),
        "minecraft:netherite_shovel" => (Netherite, Shovel),
        "minecraft:wooden_sword" => (Wood, Sword),
        "minecraft:stone_sword" => (Stone, Sword),
        "minecraft:iron_sword" => (Iron, Sword),
        "minecraft:golden_sword" => (Gold, Sword),
        "minecraft:diamond_sword" => (Diamond, Sword),
        "minecraft:netherite_sword" => (Netherite, Sword),
        "minecraft:wooden_hoe" => (Wood, Hoe),
        "minecraft:stone_hoe" => (Stone, Hoe),
        "minecraft:iron_hoe" => (Iron, Hoe),
        "minecraft:golden_hoe" => (Gold, Hoe),
        "minecraft:diamond_hoe" => (Diamond, Hoe),
        "minecraft:netherite_hoe" => (Netherite, Hoe),
        "minecraft:shears" => (Iron, Shears),
        _ => return None,
    };
    Some(DurableInfo { tier, tool_type })
}

pub fn max_durability(identifier: &str) -> Option<u32> {
    durable_info(identifier).map(|info| info.tier.max_durability())
}

pub fn is_broken(stack: &ItemStack) -> bool {
    max_durability(&stack.identifier).is_some_and(|maximum| stack.metadata >= maximum)
}

/// Apply an Unbreaking roll and then damage the item. Returns true when the
/// item reaches its maximum durability; the caller must remove it from inventory.
pub fn apply_damage<R: RandomSource>(
    stack: &mut ItemStack,
    amount: u32,
    unbreaking_level: u8,
    rng: &mut R,
) -> Result<bool, String> {
    let Some(info) = durable_info(&stack.identifier) else { return Ok(false) };
    if amount == 0 || is_broken(stack) { return Ok(is_broken(stack)); }
    let applied = if unbreaking_level == 0 {
        amount
    } else {
        let cancellation_chance = 1.0 / (f32::from(unbreaking_level) + 1.0);
        (0..amount).filter(|_| rng.next_f32() <= cancellation_chance).count() as u32
    };
    let maximum = info.tier.max_durability();
    let new_damage = stack.metadata.saturating_add(applied).min(maximum);
    stack.metadata = new_damage;
    Ok(new_damage >= maximum)
}

#[cfg(test)]
mod tests {
    use super::*;

    struct FixedRng(f32);
    impl RandomSource for FixedRng {
        fn next_u64(&mut self) -> u64 { (self.0 * u64::MAX as f32) as u64 }
        fn next_f32(&mut self) -> f32 { self.0 }
    }

    fn item(name: &str, damage: u32) -> ItemStack {
        ItemStack::new(name, 1, damage, vec![]).unwrap()
    }

    #[test]
    fn durability_and_tier_values_match_mc_rs_reference() {
        assert_eq!(ToolTier::Wood.max_durability(), 59);
        assert_eq!(ToolTier::Netherite.max_durability(), 2031);
        assert_eq!(ToolTier::Gold.mining_tier(), 0);
        assert_eq!(ToolTier::Gold.mining_speed(), 12.0);
    }

    #[test]
    fn vanilla_tool_identifier_lookup_covers_all_six_tiers() {
        assert_eq!(durable_info("minecraft:netherite_pickaxe").unwrap().tier, ToolTier::Netherite);
        assert_eq!(durable_info("minecraft:shears").unwrap().tool_type, ToolType::Shears);
        assert!(durable_info("minecraft:stick").is_none());
    }

    #[test]
    fn applying_damage_breaks_tool_only_at_maximum() {
        let mut stack = item("minecraft:wooden_pickaxe", 58);
        assert!(!apply_damage(&mut stack, 1, 0, &mut FixedRng(0.0)).unwrap());
        assert_eq!(stack.metadata, 59);
        assert!(is_broken(&stack));
        assert!(apply_damage(&mut stack, 1, 0, &mut FixedRng(0.0)).unwrap());
    }

    #[test]
    fn unbreaking_reduces_per_point_damage_using_random_rolls() {
        let mut protected = item("minecraft:iron_pickaxe", 0);
        assert!(!apply_damage(&mut protected, 8, 3, &mut FixedRng(0.0)).unwrap());
        assert_eq!(protected.metadata, 8);
        let mut worn = item("minecraft:iron_pickaxe", 0);
        assert!(!apply_damage(&mut worn, 8, 3, &mut FixedRng(0.99)).unwrap());
        assert_eq!(worn.metadata, 0);
    }

    #[test]
    fn non_durable_items_are_unchanged() {
        let mut stack = item("minecraft:stone", 0);
        assert!(!apply_damage(&mut stack, 10, 0, &mut FixedRng(0.0)).unwrap());
        assert_eq!(stack.metadata, 0);
    }
}
