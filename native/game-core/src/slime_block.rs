//! Selectively ported from MC-RS at da93a9501601fcdc284762fde88c24b44604991e.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).
//! Slime block — bouncy, sticky piston.

/// Jump multiplier when jumping off slime.
pub const JUMP_MULTIPLIER: f32 = 1.0;
/// Fall damage cancelled entirely if not sneaking.
pub fn cancels_fall_damage(sneaking: bool) -> bool {
    !sneaking
}
/// Bounce preserves momentum.
pub const BOUNCE_FACTOR: f64 = -1.0;

/// Slime stops bouncing after low velocity.
pub const MIN_BOUNCE_VELOCITY: f64 = 0.1;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn not_sneaking_no_damage() {
        assert!(cancels_fall_damage(false));
    }

    #[test]
    fn sneaking_takes_damage() {
        assert!(!cancels_fall_damage(true));
    }
}
