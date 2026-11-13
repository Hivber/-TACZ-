//! Palette bit array adapted from Chorus crates/level/src/bit_array.rs.
//! Upstream: bedrock-crustaceans/Chorus @ 5808efd55dc51f38e7900ad2c72f14d73434b093
//! Apache-2.0. Uses Bedrock's 32-bit word packing for fixed-size palettes.

#[derive(Clone)]
pub struct BitArray<const N: usize> {
    bits: u8,
    per_word: usize,
    mask: u32,
    blocks: Box<[u32]>,
}

impl<const N: usize> Default for BitArray<N> {
    fn default() -> Self { Self::new() }
}

impl<const N: usize> BitArray<N> {
    const VALID_BITS: [u8; 8] = [1, 2, 3, 4, 5, 6, 8, 16];

    pub fn new() -> Self { Self::with_bits(1) }

    pub fn with_bits(bits: u8) -> Self {
        assert!(Self::VALID_BITS.contains(&bits), "unsupported palette bit width {bits}");
        let per_word = 32 / bits as usize;
        Self {
            bits,
            per_word,
            mask: ((1u64 << bits) - 1) as u32,
            blocks: vec![0u32; N.div_ceil(per_word)].into_boxed_slice(),
        }
    }

    pub fn try_with_bits(bits: u8) -> Result<Self, String> {
        if !Self::VALID_BITS.contains(&bits) {
            return Err(format!("unsupported palette bit width {bits}"));
        }
        Ok(Self::with_bits(bits))
    }

    pub fn from_indices(bits: u8, indices: &[u16; N]) -> Self {
        let mut array = Self::with_bits(bits);
        let per_word = array.per_word;
        for (word, group) in array.blocks.iter_mut().zip(indices.chunks(per_word)) {
            *word = group.iter().enumerate().fold(0u32, |packed, (slot, &index)| {
                packed | ((index as u32) << (slot * bits as usize))
            });
        }
        array
    }

    pub fn indices(&self) -> impl Iterator<Item = u16> + '_ {
        let (bits, mask) = (self.bits as u32, self.mask);
        self.blocks.iter()
            .flat_map(move |&word| (0..self.per_word).map(move |slot| ((word >> ((slot as u32) * bits)) & mask) as u16))
            .take(N)
    }

    pub fn get_blocks(&self) -> &[u32] { &self.blocks }
    pub fn get_bits(&self) -> u8 { self.bits }
    pub fn len(&self) -> usize { N }
    pub fn is_empty(&self) -> bool { N == 0 }

    fn max_value(&self) -> u32 { self.mask }

    pub fn set(&mut self, index: usize, value: u16) -> u16 {
        assert!(index < N, "bit-array index {index} outside length {N}");
        if value > self.max_value() as u16 {
            self.resize(Self::bits_for(value));
        }
        let word = index / self.per_word;
        let offset = (index % self.per_word) * self.bits as usize;
        let mask = self.max_value() << offset;
        let old = (self.blocks[word] & mask) >> offset;
        self.blocks[word] = (self.blocks[word] & !mask) | (((value as u32) << offset) & mask);
        old as u16
    }

    pub fn get(&self, index: usize) -> u16 {
        assert!(index < N, "bit-array index {index} outside length {N}");
        let word = index / self.per_word;
        let offset = (index % self.per_word) * self.bits as usize;
        ((self.blocks[word] >> offset) & self.max_value()) as u16
    }

    pub fn try_get(&self, index: usize) -> Option<u16> {
        (index < N).then(|| self.get(index))
    }

    pub fn bits_for(value: u16) -> u8 {
        Self::VALID_BITS.iter().copied().find(|&bits| (value as u32) < (1u32 << bits)).unwrap_or(16)
    }

    pub fn resize(&mut self, bits: u8) {
        assert!(Self::VALID_BITS.contains(&bits), "unsupported palette bit width {bits}");
        if bits == self.bits { return; }
        let old_bits = self.bits;
        let old_data = std::mem::take(&mut self.blocks);
        self.bits = bits;
        self.per_word = 32 / bits as usize;
        self.mask = ((1u64 << bits) - 1) as u32;
        self.blocks = vec![0u32; N.div_ceil(self.per_word)].into_boxed_slice();

        let old_per_word = 32 / old_bits as usize;
        let old_mask = (1u32 << old_bits) - 1;
        for i in 0..N {
            let old_word = i / old_per_word;
            let old_offset = (i % old_per_word) * old_bits as usize;
            let value = (old_data[old_word] >> old_offset) & old_mask;
            self.set(i, value as u16);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    struct Lcg(u64);
    impl Lcg {
        fn new(seed: u64) -> Self { Self(seed) }
        fn next(&mut self) -> u64 {
            self.0 = self.0.wrapping_mul(6364136223846793005).wrapping_add(1);
            self.0
        }
        fn below(&mut self, upper: u32) -> u32 { ((self.next() as u128 * upper as u128) >> 64) as u32 }
    }

    const WIDTHS: [u8; 8] = [1, 2, 3, 4, 5, 6, 8, 16];

    #[test]
    fn bits_for_picks_smallest_valid_width() {
        for (value, bits) in [(0,1),(1,1),(2,2),(3,2),(4,3),(7,3),(8,4),(15,4),(16,5),(31,5),(32,6),(63,6),(64,8),(255,8),(256,16),(u16::MAX,16)] {
            assert_eq!(BitArray::<4096>::bits_for(value), bits);
        }
    }

    #[test]
    fn packing_round_trips_all_supported_widths() {
        for bits in WIDTHS {
            let mut rng = Lcg::new(bits as u64);
            let max = (1u32 << bits) - 1;
            let values: [u16; 4096] = std::array::from_fn(|_| rng.below(max + 1) as u16);
            let array = BitArray::<4096>::from_indices(bits, &values);
            assert_eq!(array.indices().collect::<Vec<_>>(), values);
            for (index, value) in values.iter().enumerate() {
                assert_eq!(array.get(index), *value);
            }
        }
    }

    #[test]
    fn setting_larger_value_resizes_and_preserves_other_entries() {
        let mut array = BitArray::<4096>::new();
        array.set(0, 1);
        array.set(100, 1);
        array.set(100, 256);
        assert_eq!(array.get_bits(), 16);
        assert_eq!(array.get(0), 1);
        assert_eq!(array.get(100), 256);
    }

    #[test]
    fn resize_down_preserves_values_that_fit() {
        let mut array = BitArray::<4096>::with_bits(8);
        for i in 0..4096 { array.set(i, (i % 4) as u16); }
        array.resize(2);
        assert_eq!(array.get_bits(), 2);
        for i in 0..4096 { assert_eq!(array.get(i), (i % 4) as u16); }
    }

    #[test]
    fn rejects_invalid_width_and_checked_lookup_bounds() {
        assert!(BitArray::<4096>::try_with_bits(7).is_err());
        assert_eq!(BitArray::<4096>::new().try_get(4096), None);
    }
}
