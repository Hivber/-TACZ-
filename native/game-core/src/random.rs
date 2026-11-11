//! Small deterministic random-source abstraction shared by gameplay algorithms.

pub trait RandomSource {
    fn next_u64(&mut self) -> u64;

    fn next_f32(&mut self) -> f32 {
        let value = (self.next_u64() >> 40) as u32;
        value as f32 / 16_777_216.0
    }

    fn range_u32(&mut self, upper_exclusive: u32) -> u32 {
        if upper_exclusive == 0 { return 0; }
        ((self.next_u64() as u128 * upper_exclusive as u128) >> 64) as u32
    }
}

/// SplitMix64 is deterministic and is not intended for cryptographic use.
#[derive(Clone, Copy, Debug)]
pub struct SplitMix64 { state: u64 }

impl SplitMix64 {
    pub fn new(seed: u64) -> Self { Self { state: seed } }
}

impl RandomSource for SplitMix64 {
    fn next_u64(&mut self) -> u64 {
        self.state = self.state.wrapping_add(0x9E37_79B9_7F4A_7C15);
        let mut value = self.state;
        value = (value ^ (value >> 30)).wrapping_mul(0xBF58_476D_1CE4_E5B9);
        value = (value ^ (value >> 27)).wrapping_mul(0x94D0_49BB_1331_11EB);
        value ^ (value >> 31)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn splitmix_is_reproducible_for_a_seed() {
        let mut a = SplitMix64::new(42);
        let mut b = SplitMix64::new(42);
        for _ in 0..16 { assert_eq!(a.next_u64(), b.next_u64()); }
    }

    #[test]
    fn bounded_values_stay_in_range() {
        let mut rng = SplitMix64::new(7);
        for _ in 0..1000 { assert!(rng.range_u32(17) < 17); }
    }
}
