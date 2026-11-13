//! Selectively ported from MC-RS at da93a9501601fcdc284762fde88c24b44604991e.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).
//! Anvil operation costs and falling damage, using stable identifiers.

use crate::random::RandomSource;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AnvilState { Normal, Chipped, Damaged, Destroyed }

impl AnvilState {
    pub const fn identifier(self) -> Option<&'static str> {
        match self {
            Self::Normal => Some("minecraft:anvil"),
            Self::Chipped => Some("minecraft:chipped_anvil"),
            Self::Damaged => Some("minecraft:damaged_anvil"),
            Self::Destroyed => None,
        }
    }

    pub const fn next_degradation(self) -> Self {
        match self {
            Self::Normal => Self::Chipped,
            Self::Chipped => Self::Damaged,
            Self::Damaged | Self::Destroyed => Self::Destroyed,
        }
    }
}

pub const DEGRADATION_CHANCE: f32 = 0.12;
pub const MAX_XP_COST: u32 = 39;
pub const RENAME_COST: u32 = 1;

pub fn should_degrade(rng: &mut impl RandomSource) -> bool {
    rng.next_f32() < DEGRADATION_CHANCE
}

pub fn combine_cost(prior_work_a: u32, prior_work_b: u32, rename: bool) -> u32 {
    prior_work_a.saturating_mul(2)
        .saturating_add(prior_work_b.saturating_mul(2))
        .saturating_add(if rename { RENAME_COST } else { 0 })
}

pub fn falling_damage(fall_distance: f32) -> f32 {
    (fall_distance * 2.0).clamp(0.0, 40.0)
}

#[cfg(test)]
mod tests {
    use super::*;
    struct FixedRng(f32);
    impl RandomSource for FixedRng { fn next_u64(&mut self) -> u64 { (self.0 as f64 * u64::MAX as f64) as u64 } }

    #[test]
    fn state_progression_uses_namespaced_identifiers_not_legacy_runtime_ids() {
        assert_eq!(AnvilState::Normal.identifier(), Some("minecraft:anvil"));
        assert_eq!(AnvilState::Damaged.next_degradation(), AnvilState::Destroyed);
        assert_eq!(AnvilState::Destroyed.identifier(), None);
    }

    #[test]
    fn degradation_is_seed_stream_driven() {
        assert!(should_degrade(&mut FixedRng(0.0)));
        assert!(!should_degrade(&mut FixedRng(0.99)));
    }

    #[test]
    fn combine_cost_and_falling_damage_are_bounded() {
        assert_eq!(combine_cost(2, 3, true), 11);
        assert_eq!(falling_damage(100.0), 40.0);
        assert_eq!(combine_cost(u32::MAX, u32::MAX, true), u32::MAX);
    }
}
