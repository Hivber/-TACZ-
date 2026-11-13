//! Deterministic composter gameplay state adapted from MC-RS.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! Source: crates/mc-rs-server/src/composter.rs
//! Revision: da93a9501601fcdc284762fde88c24b44604991e (MIT).
//!
//! Randomness is injected through RandomSource rather than using thread-local
//! randomness, so world ticks and tests can be replayed deterministically.

use crate::random::RandomSource;

/// Vanilla composter chance for an item, or None when it is not compostable.
pub fn composting_chance(item_identifier: &str) -> Option<f32> {
    let chance = match item_identifier {
        "minecraft:beetroot_seeds"
        | "minecraft:dried_kelp"
        | "minecraft:grass"
        | "minecraft:kelp"
        | "minecraft:leaves"
        | "minecraft:melon_seeds"
        | "minecraft:pumpkin_seeds"
        | "minecraft:sapling"
        | "minecraft:seagrass"
        | "minecraft:sweet_berries"
        | "minecraft:wheat_seeds"
        | "minecraft:moss_carpet"
        | "minecraft:pink_petals"
        | "minecraft:small_dripleaf" => 0.30,

        "minecraft:cactus"
        | "minecraft:dried_kelp_block"
        | "minecraft:flower"
        | "minecraft:melon_slice"
        | "minecraft:sugar_cane"
        | "minecraft:tall_grass"
        | "minecraft:vine"
        | "minecraft:wheat"
        | "minecraft:sea_pickle" => 0.50,

        "minecraft:apple"
        | "minecraft:beetroot"
        | "minecraft:carrot"
        | "minecraft:cocoa_beans"
        | "minecraft:fern"
        | "minecraft:lily_pad"
        | "minecraft:melon_block"
        | "minecraft:mushroom"
        | "minecraft:mushroom_block"
        | "minecraft:nether_wart"
        | "minecraft:potato"
        | "minecraft:pumpkin"
        | "minecraft:sunflower"
        | "minecraft:lily_of_the_valley"
        | "minecraft:tall_grass_top" => 0.65,

        "minecraft:baked_potato"
        | "minecraft:bread"
        | "minecraft:cookie"
        | "minecraft:hay_block"
        | "minecraft:brown_mushroom_block"
        | "minecraft:nether_wart_block"
        | "minecraft:warped_wart_block"
        | "minecraft:pumpkin_pie" => 0.85,

        "minecraft:cake" | "minecraft:cocoa_beans_plant" => 1.0,
        _ => return None,
    };
    Some(chance)
}

/// Persistent block-entity state. Levels 0-6 are filling; level 7 is ready.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct ComposterState {
    level: u8,
}

impl ComposterState {
    pub const MAX_LEVEL: u8 = 7;

    pub fn new() -> Self {
        Self::default()
    }

    pub fn from_level(level: u8) -> Result<Self, &'static str> {
        if level > Self::MAX_LEVEL {
            return Err("composter level must be between 0 and 7");
        }
        Ok(Self { level })
    }

    pub fn level(self) -> u8 {
        self.level
    }

    pub fn is_full(self) -> bool {
        self.level == Self::MAX_LEVEL
    }

    /// Attempts to compost one item. Returns true only when the level advances.
    pub fn try_compost(&mut self, item_identifier: &str, rng: &mut impl RandomSource) -> bool {
        if self.is_full() {
            return false;
        }
        let Some(chance) = composting_chance(item_identifier) else {
            return false;
        };
        if rng.next_f32() < chance {
            self.level += 1;
            true
        } else {
            false
        }
    }

    /// Extracts one bonemeal and resets a full composter to level zero.
    pub fn take_bonemeal(&mut self) -> bool {
        if !self.is_full() {
            return false;
        }
        self.level = 0;
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    struct FixedRng(f32);
    impl RandomSource for FixedRng {
        fn next_u64(&mut self) -> u64 {
            (self.0.clamp(0.0, 0.999_999_94) * u64::MAX as f32) as u64
        }
        fn next_f32(&mut self) -> f32 {
            self.0
        }
    }

    #[test]
    fn compostable_item_chances_match_the_source_table() {
        assert_eq!(composting_chance("minecraft:wheat_seeds"), Some(0.30));
        assert_eq!(composting_chance("minecraft:carrot"), Some(0.65));
        assert_eq!(composting_chance("minecraft:bread"), Some(0.85));
        assert_eq!(composting_chance("minecraft:cake"), Some(1.0));
        assert_eq!(composting_chance("minecraft:stone"), None);
    }

    #[test]
    fn composting_uses_injected_rng_and_advances_only_on_success() {
        let mut composter = ComposterState::new();
        assert!(!composter.try_compost("minecraft:bread", &mut FixedRng(0.90)));
        assert_eq!(composter.level(), 0);
        assert!(composter.try_compost("minecraft:bread", &mut FixedRng(0.80)));
        assert_eq!(composter.level(), 1);
        assert!(composter.try_compost("minecraft:cake", &mut FixedRng(0.999)));
        assert_eq!(composter.level(), 2);
    }

    #[test]
    fn full_composter_rejects_items_and_bonemeal_resets_it() {
        let mut composter = ComposterState::from_level(7).unwrap();
        assert!(!composter.try_compost("minecraft:cake", &mut FixedRng(0.0)));
        assert!(composter.take_bonemeal());
        assert_eq!(composter.level(), 0);
        assert!(!composter.take_bonemeal());
    }

    #[test]
    fn invalid_persisted_level_is_rejected() {
        assert!(ComposterState::from_level(8).is_err());
    }
}
