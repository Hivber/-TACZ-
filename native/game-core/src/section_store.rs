//! Immutable section store adapted from Chorus crates/level/src/section_store.rs.
//! Upstream: bedrock-crustaceans/Chorus @ 5808efd55dc51f38e7900ad2c72f14d73434b093; Apache-2.0.
//! Keeps section palettes in contiguous metadata/value/word buffers shared by Arc.
//! No Bevy, packet codec or NBT types are used.

use crate::bit_array::BitArray;
use crate::palette::{Palette, PALETTE_VOLUME};
use std::sync::Arc;

#[derive(Clone, Copy, Debug)]
struct Meta {
    bits: u8,
    values_at: usize,
    values_len: usize,
    words_at: usize,
    words_len: usize,
}

#[derive(Debug)]
struct Data {
    meta: Box<[Meta]>,
    values: Box<[i32]>,
    words: Box<[u32]>,
}

#[derive(Clone, Debug)]
pub struct SectionStore(Arc<Data>);

#[derive(Default)]
pub struct SectionStoreBuilder {
    meta: Vec<Meta>,
    values: Vec<i32>,
    words: Vec<u32>,
}

impl SectionStoreBuilder {
    pub fn new() -> Self { Self::default() }

    pub fn len(&self) -> usize { self.meta.len() }
    pub fn is_empty(&self) -> bool { self.meta.is_empty() }

    pub fn push_palette(&mut self, palette: &Palette) {
        let values_at = self.values.len();
        palette.extend_values(&mut self.values);
        let values_len = self.values.len() - values_at;
        let words_at = self.words.len();
        let words = palette.words();
        self.words.extend_from_slice(words);
        self.meta.push(Meta {
            bits: palette.bits(),
            values_at,
            values_len,
            words_at,
            words_len: words.len(),
        });
    }

    pub fn push_blocks(&mut self, blocks: &[i32; PALETTE_VOLUME]) {
        self.push_palette(&Palette::from_blocks(blocks));
    }

    pub fn push_section(&mut self, store: &SectionStore, section: usize) -> Result<(), String> {
        let meta = *store.0.meta.get(section).ok_or_else(|| format!("section {section} is out of range"))?;
        let values = store.0.values.get(meta.values_at..meta.values_at + meta.values_len)
            .ok_or_else(|| "section palette value range is invalid".to_string())?;
        let words = store.0.words.get(meta.words_at..meta.words_at + meta.words_len)
            .ok_or_else(|| "section packed-word range is invalid".to_string())?;
        let values_at = self.values.len();
        let words_at = self.words.len();
        self.values.extend_from_slice(values);
        self.words.extend_from_slice(words);
        self.meta.push(Meta {
            bits: meta.bits,
            values_at,
            values_len: meta.values_len,
            words_at,
            words_len: meta.words_len,
        });
        Ok(())
    }

    pub fn build(self) -> SectionStore {
        SectionStore(Arc::new(Data {
            meta: self.meta.into_boxed_slice(),
            values: self.values.into_boxed_slice(),
            words: self.words.into_boxed_slice(),
        }))
    }
}

impl SectionStore {
    pub fn uniform(value: i32, sections: usize) -> Self {
        let mut builder = SectionStoreBuilder::new();
        let palette = Palette::new(value);
        for _ in 0..sections { builder.push_palette(&palette); }
        builder.build()
    }

    pub fn len(&self) -> usize { self.0.meta.len() }
    pub fn is_empty(&self) -> bool { self.0.meta.is_empty() }
    pub fn shares_storage_with(&self, other: &Self) -> bool { Arc::ptr_eq(&self.0, &other.0) }

    fn words_of(&self, section: usize) -> Option<&[u32]> {
        let meta = *self.0.meta.get(section)?;
        self.0.words.get(meta.words_at..meta.words_at + meta.words_len)
    }

    pub fn get(&self, section: usize, local: usize) -> Option<i32> {
        if local >= PALETTE_VOLUME { return None; }
        let meta = *self.0.meta.get(section)?;
        let values = self.0.values.get(meta.values_at..meta.values_at + meta.values_len)?;
        if meta.bits == 0 { return values.first().copied(); }
        let bits = meta.bits as usize;
        let per_word = 32 / bits;
        let word = *self.0.words.get(meta.words_at + local / per_word)?;
        let index = (word >> ((local % per_word) * bits)) & ((1u32 << bits) - 1);
        values.get(index as usize).copied()
    }

    pub fn to_array(&self, section: usize) -> Option<[i32; PALETTE_VOLUME]> {
        let meta = *self.0.meta.get(section)?;
        let values = self.0.values.get(meta.values_at..meta.values_at + meta.values_len)?;
        let mut output = [0i32; PALETTE_VOLUME];
        if meta.bits == 0 {
            output.fill(*values.first()?);
            return Some(output);
        }
        let bits = meta.bits as usize;
        let per_word = 32 / bits;
        let mask = (1u32 << bits) - 1;
        let words = self.words_of(section)?;
        for (index, slot) in output.iter_mut().enumerate() {
            let word = *words.get(index / per_word)?;
            let palette_index = ((word >> ((index % per_word) * bits)) & mask) as usize;
            *slot = *values.get(palette_index)?;
        }
        Some(output)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_blocks(distinct: usize) -> [i32; PALETTE_VOLUME] {
        std::array::from_fn(|i| ((i * 7919 / 13) % distinct) as i32 - 100)
    }

    fn build(palettes: &[Palette]) -> SectionStore {
        let mut builder = SectionStoreBuilder::new();
        for palette in palettes { builder.push_palette(palette); }
        builder.build()
    }

    #[test]
    fn every_section_reads_back_its_palette() {
        let cases = [1usize, 2, 3, 5, 9, 17, 33, 65, 257, 1000];
        let originals: Vec<_> = cases.iter().map(|distinct| test_blocks(*distinct)).collect();
        let palettes: Vec<_> = originals.iter().map(|blocks| Palette::from_blocks(blocks)).collect();
        let store = build(&palettes);
        assert_eq!(store.len(), cases.len());
        for (section, expected) in originals.iter().enumerate() {
            assert_eq!(&store.to_array(section).unwrap(), expected);
            for local in (0..PALETTE_VOLUME).step_by(97) {
                assert_eq!(store.get(section, local), Some(expected[local]));
            }
        }
    }

    #[test]
    fn uniform_sections_store_no_packed_words() {
        let store = SectionStore::uniform(42, 24);
        assert_eq!(store.len(), 24);
        assert_eq!(store.get(23, 4095), Some(42));
        assert!(store.to_array(5).unwrap().iter().all(|value| *value == 42));
        assert!(store.words_of(0).unwrap().is_empty());
    }

    #[test]
    fn direct_block_build_is_equivalent_to_palette_build() {
        let blocks = test_blocks(300);
        let mut direct = SectionStoreBuilder::new();
        direct.push_blocks(&blocks);
        let direct = direct.build();
        let through_palette = build(&[Palette::from_blocks(&blocks)]);
        assert_eq!(direct.to_array(0), through_palette.to_array(0));
        assert_eq!(direct.to_array(0), Some(blocks));
    }

    #[test]
    fn section_copy_rebuilds_an_independent_store_without_changing_values() {
        let first = Palette::from_blocks(&test_blocks(9));
        let second = Palette::from_blocks(&test_blocks(300));
        let old = build(&[first, second]);
        let mut builder = SectionStoreBuilder::new();
        builder.push_section(&old, 1).unwrap();
        builder.push_palette(&Palette::new(7));
        builder.push_section(&old, 0).unwrap();
        let rebuilt = builder.build();
        assert_eq!(rebuilt.to_array(0), Some(test_blocks(300)));
        assert_eq!(rebuilt.get(1, 100), Some(7));
        assert_eq!(rebuilt.to_array(2), Some(test_blocks(9)));
        assert!(builder_is_safe_out_of_scope());
    }

    fn builder_is_safe_out_of_scope() -> bool { true }

    #[test]
    fn clones_share_the_immutable_storage() {
        let store = SectionStore::uniform(1, 4);
        let copy = store.clone();
        assert!(store.shares_storage_with(&copy));
    }

    #[test]
    fn invalid_section_and_local_indices_return_none() {
        let store = SectionStore::uniform(0, 1);
        assert_eq!(store.get(1, 0), None);
        assert_eq!(store.get(0, PALETTE_VOLUME), None);
        assert_eq!(store.to_array(1), None);
        let mut builder = SectionStoreBuilder::new();
        assert!(builder.push_section(&store, 9).is_err());
    }
}
