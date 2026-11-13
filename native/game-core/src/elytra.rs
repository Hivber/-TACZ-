//! Elytra gliding state and durability rules adapted from MC-RS.
//!
//! Upstream: https://github.com/pedrokarim/mc-rs
//! Revision: da93a9501601fcdc284762fde88c24b44604991e
//! Source: crates/mc-rs-server/src/elytra.rs
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).
//!
//! This module models equipment state only. Host movement integration must
//! still supply Bedrock-compatible glide velocity, pitch lift, drag, collision,
//! and firework impulse calculations before this can drive live player physics.

use crate::random::RandomSource;

/// Maximum durability of an undamaged elytra.
pub const MAX_DURABILITY: u16 = 432;
/// Firework boost duration per flight level, in game ticks.
pub const FIREWORK_TICKS_PER_FLIGHT_LEVEL: u32 = 20;

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ElytraState {
    pub gliding: bool,
    pub durability: u16,
    pub firework_boost_ticks: u32,
}

impl ElytraState {
    pub fn new() -> Self {
        Self { gliding: false, durability: MAX_DURABILITY, firework_boost_ticks: 0 }
    }

    pub fn start_gliding(&mut self) -> bool {
        if self.durability == 0 { return false; }
        self.gliding = true;
        true
    }

    pub fn stop_gliding(&mut self) {
        self.gliding = false;
    }

    pub fn boost_with_firework(&mut self, flight_level: u8) {
        self.firework_boost_ticks = u32::from(flight_level).saturating_mul(FIREWORK_TICKS_PER_FLIGHT_LEVEL);
    }

    /// Advances one game tick. Returns true exactly when this tick breaks the elytra.
    /// The RNG is injected so server replays and tests remain deterministic.
    pub fn tick<R: RandomSource>(&mut self, rng: &mut R) -> bool {
        let mut broke = false;
        if self.gliding {
            if self.durability > 0 && rng.range_u32(20) == 0 {
                self.durability -= 1;
                if self.durability == 0 {
                    self.gliding = false;
                    broke = true;
                }
            }
        }
        self.firework_boost_ticks = self.firework_boost_ticks.saturating_sub(1);
        broke
    }

    pub fn is_broken(&self) -> bool { self.durability == 0 }
    pub fn is_boosted(&self) -> bool { self.firework_boost_ticks > 0 }
}

impl Default for ElytraState {
    fn default() -> Self { Self::new() }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::random::SplitMix64;

    #[test]
    fn broken_elytra_cannot_start_gliding() {
        let mut state = ElytraState::new();
        state.durability = 0;
        assert!(!state.start_gliding());
        assert!(!state.gliding);
    }

    #[test]
    fn firework_boost_counts_down_in_ticks() {
        let mut state = ElytraState::new();
        state.boost_with_firework(3);
        assert_eq!(state.firework_boost_ticks, 60);
        let mut rng = SplitMix64::new(1);
        for _ in 0..60 { state.tick(&mut rng); }
        assert!(!state.is_boosted());
    }

    #[test]
    fn seeded_durability_rolls_are_reproducible() {
        let mut a = ElytraState::new();
        let mut b = ElytraState::new();
        a.start_gliding();
        b.start_gliding();
        let mut rng_a = SplitMix64::new(123);
        let mut rng_b = SplitMix64::new(123);
        for _ in 0..2_000 {
            assert_eq!(a.tick(&mut rng_a), b.tick(&mut rng_b));
            assert_eq!(a.durability, b.durability);
        }
        assert!(a.durability <= MAX_DURABILITY);
    }

    #[test]
    fn zero_durability_stops_gliding() {
        let mut state = ElytraState::new();
        state.durability = 1;
        state.start_gliding();
        // Find a deterministic seed whose first bounded roll is zero.
        let seed = (0..10_000).find(|seed| { let mut r = SplitMix64::new(*seed); r.range_u32(20) == 0 }).unwrap();
        let mut rng = SplitMix64::new(seed);
        state.tick(&mut rng);
        assert!(state.is_broken());
        assert!(!state.gliding);
    }
}
