//! Selectively ported from MC-RS at da93a9501601fcdc284762fde88c24b44604991e.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).
//! Horse / Donkey / Mule attribute and variant generation.
//! Randomness is injected from the server's seeded world RNG for deterministic spawning.

use crate::random::RandomSource;

pub fn random_horse_health(rng: &mut impl RandomSource) -> f32 {
    15.0 + (rng.next_f32() * 9.0 + rng.next_f32() * 9.0 + rng.next_f32() * 9.0) / 3.0
}

pub fn random_horse_jump_strength(rng: &mut impl RandomSource) -> f32 {
    0.4 + rng.next_f32() * 0.2 + rng.next_f32() * 0.2 + rng.next_f32() * 0.2
}

pub fn random_horse_movement_speed(rng: &mut impl RandomSource) -> f32 {
    (0.45 + rng.next_f32() * 0.3 + rng.next_f32() * 0.3 + rng.next_f32() * 0.3) * 0.25
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct HorseVariant {
    pub base_color: u8,
    pub markings: u8,
}

impl HorseVariant {
    pub fn random(rng: &mut impl RandomSource) -> Self {
        Self { base_color: rng.range_u32(7) as u8, markings: rng.range_u32(5) as u8 }
    }

    pub fn as_meta(&self) -> u32 {
        (self.base_color as u32) | ((self.markings as u32) << 8)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::random::SplitMix64;

    #[test]
    fn horse_attributes_stay_in_vanilla_ranges() {
        let mut rng = SplitMix64::new(42);
        let health = random_horse_health(&mut rng);
        let jump = random_horse_jump_strength(&mut rng);
        let speed = random_horse_movement_speed(&mut rng);
        assert!((15.0..=24.0).contains(&health));
        assert!((0.4..=1.0).contains(&jump));
        assert!((0.1125..=0.3375).contains(&speed));
    }

    #[test]
    fn horse_variants_are_bounded_and_seeded() {
        let mut a = SplitMix64::new(9);
        let mut b = SplitMix64::new(9);
        let av = HorseVariant::random(&mut a);
        let bv = HorseVariant::random(&mut b);
        assert_eq!(av, bv);
        assert!(av.base_color < 7);
        assert!(av.markings < 5);
    }
}
