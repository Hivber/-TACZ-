//! Entity despawn rules copied from MC-RS.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! Revision: da93a9501601fcdc284762fde88c24b44604991e
//! Original: crates/mc-rs-server/src/despawn_rules.rs
//! License: MIT; see third_party/licenses/MC-RS-MIT.txt.
//!
//! This module only classifies despawn eligibility. It does not schedule entity
//! removal or establish exact Bedrock 1.26.60 despawn parity.

/// Hard despawn at 128+ blocks from any player.
pub const HARD_DESPAWN_RADIUS: f64 = 128.0;
/// Soft despawn (chance-based) at 32-128 blocks.
pub const SOFT_DESPAWN_MIN: f64 = 32.0;

/// Soft despawn chance per tick (1 in 800 for hostile).
pub const SOFT_DESPAWN_CHANCE: u32 = 800;

/// Entity qualifies for despawn?
pub fn should_consider_despawn(
    distance_to_nearest_player: f64,
    is_persistent: bool,
    has_custom_name: bool,
    is_tamed: bool,
) -> bool {
    if is_persistent || has_custom_name || is_tamed {
        return false;
    }
    distance_to_nearest_player > SOFT_DESPAWN_MIN
}

/// Hard despawn check.
pub fn should_hard_despawn(distance_to_nearest_player: f64) -> bool {
    distance_to_nearest_player > HARD_DESPAWN_RADIUS
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn named_mob_doesnt_despawn() {
        assert!(!should_consider_despawn(200.0, false, true, false));
    }

    #[test]
    fn hard_despawn_at_128() {
        assert!(should_hard_despawn(200.0));
    }
}
