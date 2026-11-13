//! Ore and furnace experience values adapted from MC-RS drop_xp.rs.
//! Upstream: https://github.com/pedrokarim/mc-rs @ da93a9501601fcdc284762fde88c24b44604991e; MIT.
//! Original MC-RS used legacy/versioned numeric block IDs. This port uses
//! namespaced block identifiers to avoid coupling gameplay to wire/runtime IDs.
//! Validate the values against Bedrock 1.26.60 before enabling as authority.

use crate::random::RandomSource;

pub fn ore_xp_range(identifier: &str) -> (u32, u32) {
    match identifier {
        "minecraft:coal_ore" | "minecraft:deepslate_coal_ore" => (0, 2),
        "minecraft:iron_ore" | "minecraft:deepslate_iron_ore" => (0, 0),
        "minecraft:gold_ore" | "minecraft:deepslate_gold_ore" | "minecraft:nether_gold_ore" => (0, 1),
        "minecraft:diamond_ore" | "minecraft:deepslate_diamond_ore" => (3, 7),
        "minecraft:emerald_ore" | "minecraft:deepslate_emerald_ore" => (3, 7),
        "minecraft:redstone_ore" | "minecraft:deepslate_redstone_ore" => (2, 5),
        "minecraft:lit_redstone_ore" | "minecraft:lit_deepslate_redstone_ore" => (2, 5),
        "minecraft:lapis_ore" | "minecraft:deepslate_lapis_ore" => (2, 5),
        "minecraft:quartz_ore" | "minecraft:nether_quartz_ore" => (2, 5),
        _ => (0, 0),
    }
}

pub fn roll_ore_xp<R: RandomSource>(identifier: &str, rng: &mut R) -> u32 {
    let (min, max) = ore_xp_range(identifier);
    if max <= min { return min; }
    min + rng.range_u32(max - min + 1)
}

pub fn smelt_xp(item: &str) -> f32 {
    match item {
        "minecraft:iron_ore" | "minecraft:raw_iron" => 0.7,
        "minecraft:gold_ore" | "minecraft:raw_gold" => 1.0,
        "minecraft:copper_ore" | "minecraft:raw_copper" => 0.7,
        "minecraft:diamond_ore" => 1.0,
        "minecraft:emerald_ore" => 1.0,
        "minecraft:nether_quartz_ore" | "minecraft:quartz" => 0.2,
        "minecraft:coal_ore" | "minecraft:coal" => 0.1,
        "minecraft:redstone_ore" => 0.7,
        "minecraft:lapis_ore" | "minecraft:lapis_lazuli" => 0.2,
        "minecraft:cobblestone" => 0.1,
        "minecraft:sand" => 0.1,
        "minecraft:clay" => 0.35,
        "minecraft:kelp" => 0.1,
        _ => 0.0,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    struct FixedRng(u64);
    impl RandomSource for FixedRng { fn next_u64(&mut self) -> u64 { self.0 } }

    #[test]
    fn ore_xp_ranges_are_identifier_based_and_bounded() {
        assert_eq!(ore_xp_range("minecraft:diamond_ore"), (3, 7));
        assert_eq!(ore_xp_range("minecraft:iron_ore"), (0, 0));
        assert_eq!(ore_xp_range("minecraft:unknown_ore"), (0, 0));
    }

    #[test]
    fn ore_xp_roll_includes_both_endpoints() {
        let mut low = FixedRng(0);
        let mut high = FixedRng(u64::MAX);
        assert_eq!(roll_ore_xp("minecraft:diamond_ore", &mut low), 3);
        assert_eq!(roll_ore_xp("minecraft:diamond_ore", &mut high), 7);
    }

    #[test]
    fn furnace_xp_values_match_reference_samples() {
        assert_eq!(smelt_xp("minecraft:raw_iron"), 0.7);
        assert_eq!(smelt_xp("minecraft:clay"), 0.35);
        assert_eq!(smelt_xp("minecraft:unknown"), 0.0);
    }
}
