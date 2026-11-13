//! Chunk block/biome palette adapted from Chorus crates/level/src/palette.rs.
//! Upstream: bedrock-crustaceans/Chorus @ 5808efd55dc51f38e7900ad2c72f14d73434b093
//! Apache-2.0. Protocol serializers are intentionally excluded; Go remains
//! responsible for Bedrock chunk packet encoding.

use crate::bit_array::BitArray;
use std::collections::HashMap;

pub const PALETTE_VOLUME: usize = 4096;

#[derive(Clone, Debug)]
pub enum Palette {
    Uniform { value: i32 },
    Indexed {
        values: Vec<i32>,
        counts: Vec<u16>,
        indices: BitArray<PALETTE_VOLUME>,
        lookup: HashMap<i32, u16>,
    },
}

impl Palette {
    pub fn new(value: i32) -> Self { Self::Uniform { value } }

    pub fn bits(&self) -> u8 {
        match self { Self::Uniform { .. } => 0, Self::Indexed { indices, .. } => indices.get_bits() }
    }

    pub fn extend_values(&self, out: &mut Vec<i32>) {
        match self {
            Self::Uniform { value } => out.push(*value),
            Self::Indexed { values, .. } => out.extend(values.iter().copied()),
        }
    }

    pub fn words(&self) -> &[u32] {
        match self { Self::Uniform { .. } => &[], Self::Indexed { indices, .. } => indices.get_blocks() }
    }

    pub fn is_all(&self, value: i32) -> bool { self.count(value) == PALETTE_VOLUME as u32 }

    pub fn from_blocks(blocks: &[i32; PALETTE_VOLUME]) -> Self {
        if blocks.iter().all(|value| *value == blocks[0]) {
            return Self::Uniform { value: blocks[0] };
        }
        let mut values = Vec::with_capacity(16);
        let mut counts: Vec<u16> = Vec::with_capacity(16);
        let mut lookup = HashMap::with_capacity(16);
        let mut raw_indices = [0u16; PALETTE_VOLUME];
        for (slot, &value) in blocks.iter().enumerate() {
            let palette_index = match lookup.get(&value).copied() {
                Some(index) => index as usize,
                None => {
                    let index = values.len() as u16;
                    values.push(value);
                    counts.push(0);
                    lookup.insert(value, index);
                    index as usize
                }
            };
            counts[palette_index] = counts[palette_index].saturating_add(1);
            raw_indices[slot] = palette_index as u16;
        }
        if values.len() == 1 {
            return Self::Uniform { value: values[0] };
        }
        let bits = BitArray::<PALETTE_VOLUME>::bits_for((values.len() - 1) as u16);
        Self::Indexed {
            values, counts, indices: BitArray::from_indices(bits, &raw_indices), lookup,
        }
    }

    pub fn get(&self, index: usize) -> i32 {
        assert!(index < PALETTE_VOLUME, "palette index outside section");
        match self {
            Self::Uniform { value } => *value,
            Self::Indexed { values, indices, .. } => values[indices.get(index) as usize],
        }
    }

    pub fn try_get(&self, index: usize) -> Option<i32> {
        (index < PALETTE_VOLUME).then(|| self.get(index))
    }

    pub fn to_array(&self) -> [i32; PALETTE_VOLUME] {
        match self {
            Self::Uniform { value } => [*value; PALETTE_VOLUME],
            Self::Indexed { values, indices, .. } => {
                let mut blocks = [0i32; PALETTE_VOLUME];
                for (block, index) in blocks.iter_mut().zip(indices.indices()) {
                    *block = values[index as usize];
                }
                blocks
            }
        }
    }

    pub fn count(&self, value: i32) -> u32 {
        match self {
            Self::Uniform { value: current } => if *current == value { PALETTE_VOLUME as u32 } else { 0 },
            Self::Indexed { counts, lookup, .. } => lookup.get(&value).map(|index| counts[*index as usize] as u32).unwrap_or(0),
        }
    }

    pub fn set(&mut self, index: usize, value: i32) {
        assert!(index < PALETTE_VOLUME, "palette index outside section");
        match self {
            Self::Uniform { value: current } => {
                if *current == value { return; }
                let old = *current;
                let mut lookup = HashMap::with_capacity(2);
                lookup.insert(old, 0);
                lookup.insert(value, 1);
                let mut indices = BitArray::<PALETTE_VOLUME>::new();
                indices.set(index, 1);
                *self = Self::Indexed {
                    values: vec![old, value],
                    counts: vec![PALETTE_VOLUME as u16 - 1, 1],
                    indices,
                    lookup,
                };
            }
            Self::Indexed { values, counts, indices, lookup } => {
                let old_index = indices.get(index) as usize;
                let old_value = values[old_index];
                if old_value == value { return; }

                let new_index = match lookup.get(&value).copied() {
                    Some(index) => index as usize,
                    None => {
                        let index = values.len();
                        values.push(value);
                        counts.push(0);
                        lookup.insert(value, index as u16);
                        index
                    }
                };

                counts[new_index] = counts[new_index].saturating_add(1);
                indices.set(index, new_index as u16);
                counts[old_index] = counts[old_index].saturating_sub(1);

                // Keep dead middle entries stable so existing packed indices do
                // not need remapping. When the complete tail is dead, trim it.
                if counts[old_index] == 0 && counts[old_index..].iter().all(|count| *count == 0) {
                    values.truncate(old_index);
                    counts.truncate(old_index);
                    lookup.retain(|_, palette_index| (*palette_index as usize) < old_index);
                    if values.len() == 1 {
                        *self = Self::Uniform { value: values[0] };
                        return;
                    }
                    if !values.is_empty() {
                        let wanted_bits = BitArray::<PALETTE_VOLUME>::bits_for((values.len() - 1) as u16);
                        if wanted_bits < indices.get_bits() { indices.resize(wanted_bits); }
                    }
                }
            }
        }
    }

    /// Write the Chorus disk palette representation. This is a storage format,
    /// not a Bedrock network packet codec.
    pub fn write_blocks_disk(&self, output: &mut Vec<u8>, mut write_value: impl FnMut(i32, &mut Vec<u8>)) {
        match self {
            Self::Uniform { value } => {
                output.push(0);
                output.extend(1u32.to_le_bytes());
                write_value(*value, output);
            }
            Self::Indexed { values, indices, .. } => {
                write_indices(output, indices);
                output.extend((values.len() as u32).to_le_bytes());
                for &value in values { write_value(value, output); }
            }
        }
    }

    pub fn write_biomes_disk(&self, output: &mut Vec<u8>) {
        match self {
            Self::Uniform { value } => {
                output.push(0);
                output.extend(value.to_le_bytes());
            }
            Self::Indexed { values, indices, .. } => {
                write_indices(output, indices);
                output.extend((values.len() as u32).to_le_bytes());
                for value in values { output.extend(value.to_le_bytes()); }
            }
        }
    }
}

fn write_indices(output: &mut Vec<u8>, indices: &BitArray<PALETTE_VOLUME>) {
    output.push(indices.get_bits() << 1);
    for word in indices.get_blocks() { output.extend(word.to_le_bytes()); }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn uniform_palette_reads_counts_and_disk_encoding() {
        let palette = Palette::new(9);
        assert_eq!(palette.get(0), 9);
        assert_eq!(palette.get(4095), 9);
        assert_eq!(palette.count(9), 4096);
        assert_eq!(palette.count(8), 0);
        assert!(palette.is_all(9));
        let mut bytes = Vec::new();
        palette.write_biomes_disk(&mut bytes);
        assert_eq!(bytes, [0, 9, 0, 0, 0]);
    }

    #[test]
    fn changing_one_uniform_entry_creates_indexed_palette_and_reverting_collapses() {
        let mut palette = Palette::new(9);
        palette.set(77, 4);
        assert_eq!(palette.bits(), 1);
        assert_eq!(palette.get(77), 4);
        assert_eq!(palette.count(9), 4095);
        palette.set(77, 9);
        assert!(matches!(palette, Palette::Uniform { value: 9 }));
    }

    #[test]
    fn construction_from_blocks_preserves_first_seen_palette_order_and_counts() {
        let mut blocks = [1i32; 4096];
        blocks[0] = 2;
        blocks[1] = 2;
        blocks[2] = 3;
        let palette = Palette::from_blocks(&blocks);
        assert_eq!((palette.get(0), palette.get(1), palette.get(2), palette.get(3)), (2, 2, 3, 1));
        assert_eq!((palette.count(1), palette.count(2), palette.count(3)), (4093, 2, 1));
        assert_eq!(palette.to_array(), blocks);
    }

    #[test]
    fn bit_width_grows_at_palette_boundaries() {
        let mut palette = Palette::new(0);
        let mut next = 1usize;
        for (length, bits) in [(2usize,1u8),(3,2),(5,3),(9,4),(17,5),(33,6),(65,8),(257,16)] {
            while match &palette { Palette::Uniform { .. } => 1, Palette::Indexed { values, .. } => values.len() } < length {
                palette.set(next, next as i32);
                next += 1;
            }
            assert_eq!(palette.bits(), bits);
        }
    }

    #[test]
    fn overwrites_update_counts_without_corrupting_other_indices() {
        let mut blocks = [1i32; 4096];
        blocks[0] = 2;
        blocks[1] = 2;
        let mut palette = Palette::from_blocks(&blocks);
        palette.set(0, 3);
        assert_eq!((palette.count(1), palette.count(2), palette.count(3)), (4093, 1, 1));
        assert_eq!(palette.get(1), 2);
        assert_eq!(palette.get(2), 1);
    }

    #[test]
    fn all_cells_can_be_rewritten_to_one_value() {
        let blocks = std::array::from_fn(|i| (i % 2) as i32);
        let mut palette = Palette::from_blocks(&blocks);
        for index in 0..4096 { palette.set(index, 7); }
        assert!(palette.is_all(7));
        assert_eq!(palette.count(7), 4096);
        assert_eq!(palette.get(2048), 7);
    }

    #[test]
    fn invalid_lookup_is_rejected_without_panics() {
        assert_eq!(Palette::new(1).try_get(4096), None);
    }
}
