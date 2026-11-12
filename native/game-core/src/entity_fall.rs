//! Imported from MC-RS at da93a9501601fcdc284762fde88c24b44604991e.
//! Original path: crates/mc-rs-server/src/entity_fall.rs
//! License: MIT; see third_party/licenses/MC-RS-MIT.txt.
//! Transport-independent gameplay logic only; no MC-RS networking or packet code.

//! Entity fall damage calculation.

/// Fall damage per block above threshold.
pub const FALL_DAMAGE_PER_BLOCK: f32 = 1.0;
/// Threshold below which no damage.
pub const FALL_THRESHOLD: f32 = 3.0;

/// Feather falling enchant reduces fall damage.
pub fn feather_falling_reduction(level: u8) -> f32 {
    0.12 * level as f32
}

/// Compute fall damage.
pub fn compute_damage(fall_distance: f32, feather_falling: u8) -> f32 {
    if !fall_distance.is_finite() || fall_distance <= 0.0 {
        return 0.0;
    }
    let base = (fall_distance - FALL_THRESHOLD).max(0.0) * FALL_DAMAGE_PER_BLOCK;
    let reduction = feather_falling_reduction(feather_falling).min(0.8);
    base * (1.0 - reduction)
}

/// Hay bales reduce fall damage to 20%.
pub const HAY_BALE_REDUCTION: f32 = 0.2;
/// Slime block cancels damage if not sneaking.
pub const SLIME_BLOCK_CANCELS: bool = true;
/// Honey block reduces by 80%.
pub const HONEY_BLOCK_REDUCTION: f32 = 0.2;
/// Water cancels damage.
pub const WATER_CANCELS: bool = true;
/// Bed reduces damage to 50%.
pub const BED_REDUCTION: f32 = 0.5;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn below_threshold_no_damage() {
        assert_eq!(compute_damage(2.0, 0), 0.0);
    }

    #[test]
    fn feather_falling_reduces() {
        let with = compute_damage(10.0, 4);
        let without = compute_damage(10.0, 0);
        assert!(with < without);
    }

    #[test]
    fn invalid_fall_distances_do_not_create_nan_damage() {
        assert_eq!(compute_damage(f32::NAN, 0), 0.0);
        assert_eq!(compute_damage(f32::INFINITY, 0), 0.0);
        assert_eq!(compute_damage(-10.0, 0), 0.0);
    }

    #[test]
    fn feather_falling_reduction_is_capped() {
        assert_eq!(compute_damage(10.0, u8::MAX), 1.4);
    }
}
