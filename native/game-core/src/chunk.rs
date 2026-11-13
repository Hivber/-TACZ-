//! In-memory chunk model adapted from Chorus crates/level/src/chunk.rs.
//! Upstream: bedrock-crustaceans/Chorus @ 5808efd55dc51f38e7900ad2c72f14d73434b093; Apache-2.0.
//! Network packet serialization and NBT codec types are intentionally omitted.
//! Block-entity payloads are opaque NBT bytes; coordinates are tracked separately.

use crate::sub_chunk::SubChunk;

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct BlockEntity {
    pub x: i32,
    pub y: i32,
    pub z: i32,
    pub data: Vec<u8>,
}

#[derive(Clone, Debug)]
pub struct Chunk {
    pub x: i32,
    pub z: i32,
    sub_chunks: Vec<SubChunk>,
    min_sub_chunk_y: i8,
    block_entities: Vec<BlockEntity>,
    dirty: bool,
}

impl Chunk {
    pub fn new(x: i32, z: i32, min_sub_chunk_y: i8, count: usize, air_id: i32, biome: i32) -> Self {
        Self {
            x, z,
            sub_chunks: (0..count).map(|_| SubChunk::new(air_id, biome)).collect(),
            min_sub_chunk_y,
            block_entities: Vec::new(),
            dirty: false,
        }
    }

    pub fn from_sub_chunks(
        x: i32,
        z: i32,
        min_sub_chunk_y: i8,
        sub_chunks: Vec<SubChunk>,
        block_entities: Vec<BlockEntity>,
    ) -> Self {
        Self { x, z, sub_chunks, min_sub_chunk_y, block_entities, dirty: false }
    }

    pub fn is_dirty(&self) -> bool { self.dirty }
    pub fn mark_dirty(&mut self) { self.dirty = true; }
    pub fn mark_saved(&mut self) { self.dirty = false; }

    fn sub_chunk_index_for_y(&self, y: i32) -> Option<usize> {
        usize::try_from((y >> 4) - i32::from(self.min_sub_chunk_y)).ok()
    }

    fn sub_chunk_offset(&self, sub_y: i8) -> Option<usize> {
        usize::try_from(i16::from(sub_y) - i16::from(self.min_sub_chunk_y)).ok()
    }

    pub fn set_biome(&mut self, x: u8, y: i32, z: u8, biome: i32) -> bool {
        let Some(index) = self.sub_chunk_index_for_y(y) else { return false };
        let changed = self.sub_chunks.get_mut(index)
            .is_some_and(|sub| sub.set_biome(x, (y & 0xF) as u8, z, biome));
        self.dirty |= changed;
        changed
    }

    pub fn get_sub_chunk(&self, sub_y: i8) -> Option<&SubChunk> {
        self.sub_chunks.get(self.sub_chunk_offset(sub_y)?)
    }

    pub fn get_sub_chunk_mut(&mut self, sub_y: i8) -> Option<&mut SubChunk> {
        let offset = self.sub_chunk_offset(sub_y)?;
        let sub = self.sub_chunks.get_mut(offset);
        self.dirty |= sub.is_some();
        sub
    }

    pub fn get_block(&self, x: u8, y: i32, z: u8, layer: usize) -> Option<i32> {
        let sub = self.sub_chunks.get(self.sub_chunk_index_for_y(y)?)?;
        sub.get(x, (y & 0xF) as u8, z, layer)
    }

    pub fn get_biome(&self, x: u8, y: i32, z: u8) -> Option<i32> {
        let sub = self.sub_chunks.get(self.sub_chunk_index_for_y(y)?)?;
        sub.get_biome(x, (y & 0xF) as u8, z)
    }

    pub fn set_block(&mut self, x: u8, y: i32, z: u8, layer: usize, block_id: i32) -> bool {
        let Some(index) = self.sub_chunk_index_for_y(y) else { return false };
        let changed = self.sub_chunks.get_mut(index)
            .is_some_and(|sub| sub.set(x, (y & 0xF) as u8, z, layer, block_id));
        self.dirty |= changed;
        if changed && layer == 0 {
            self.remove_block_entity(self.x.saturating_mul(16).saturating_add(i32::from(x)), y,
                self.z.saturating_mul(16).saturating_add(i32::from(z)));
        }
        changed
    }

    pub fn highest_non_air_sub_chunk_y_relative(&self) -> i8 {
        for i in (0..self.sub_chunks.len()).rev() {
            if !self.sub_chunks[i].is_all_air() {
                return self.min_sub_chunk_y.wrapping_add(i as i8);
            }
        }
        self.min_sub_chunk_y
    }

    pub fn highest_non_air_sub_chunk_y(&self) -> i8 {
        for (i, sub) in self.sub_chunks.iter().enumerate().rev() {
            if !sub.is_all_air() { return (i + 1) as i8; }
        }
        0
    }

    pub fn sub_chunks(&self) -> &[SubChunk] { &self.sub_chunks }
    pub fn min_sub_chunk_y(&self) -> i8 { self.min_sub_chunk_y }
    pub fn block_entities(&self) -> &[BlockEntity] { &self.block_entities }
    pub fn sub_chunk_count(&self) -> usize { self.sub_chunks.len() }

    pub fn block_entity_at(&self, x: i32, y: i32, z: i32) -> Option<&[u8]> {
        self.block_entities.iter()
            .find(|entity| (entity.x, entity.y, entity.z) == (x, y, z))
            .map(|entity| entity.data.as_slice())
    }

    pub fn set_block_entity(&mut self, x: i32, y: i32, z: i32, data: Vec<u8>) {
        match self.block_entities.iter_mut().find(|entity| (entity.x, entity.y, entity.z) == (x, y, z)) {
            Some(entity) => entity.data = data,
            None => self.block_entities.push(BlockEntity { x, y, z, data }),
        }
        self.dirty = true;
    }

    pub fn remove_block_entity(&mut self, x: i32, y: i32, z: i32) -> bool {
        let before = self.block_entities.len();
        self.block_entities.retain(|entity| (entity.x, entity.y, entity.z) != (x, y, z));
        let removed = self.block_entities.len() != before;
        self.dirty |= removed;
        removed
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const AIR: i32 = -7;
    const PLAINS: i32 = 1;

    fn overworld() -> Chunk { Chunk::new(3, -2, -4, 24, AIR, PLAINS) }

    #[test]
    fn new_chunk_has_expected_vertical_range_and_clean_state() {
        let chunk = overworld();
        assert_eq!((chunk.x, chunk.z), (3, -2));
        assert_eq!(chunk.sub_chunk_count(), 24);
        assert_eq!(chunk.min_sub_chunk_y(), -4);
        assert!(!chunk.is_dirty());
        assert_eq!(chunk.highest_non_air_sub_chunk_y(), 0);
        assert_eq!(chunk.highest_non_air_sub_chunk_y_relative(), -4);
    }

    #[test]
    fn blocks_work_at_height_extremes_and_negative_boundaries() {
        let mut chunk = overworld();
        assert!(chunk.set_block(0, -64, 0, 0, 1));
        assert!(chunk.set_block(15, 319, 15, 0, 2));
        assert_eq!(chunk.get_block(0, -64, 0, 0), Some(1));
        assert_eq!(chunk.get_block(15, 319, 15, 0), Some(2));
        for (y, sub_y, local_y) in [(-1, -1i8, 15u8), (0, 0, 0), (15, 0, 15), (16, 1, 0), (-16, -1, 0), (-17, -2, 15)] {
            assert!(chunk.set_block(4, y, 5, 0, 9));
            assert_eq!(chunk.get_sub_chunk(sub_y).unwrap().get(4, local_y, 5, 0), Some(9));
            assert_eq!(chunk.get_block(4, y, 5, 0), Some(9));
        }
    }

    #[test]
    fn y_outside_height_or_local_coordinate_range_is_rejected() {
        let mut chunk = overworld();
        assert_eq!(chunk.get_block(0, -65, 0, 0), None);
        assert_eq!(chunk.get_block(0, 320, 0, 0), None);
        assert_eq!(chunk.get_block(16, 0, 0, 0), None);
        assert!(!chunk.set_block(0, -65, 0, 0, 1));
        assert!(!chunk.set_block(0, 320, 0, 0, 1));
        assert!(!chunk.set_block(0, 0, 0, 2, 1));
        assert!(!chunk.is_dirty());
    }

    #[test]
    fn block_layers_and_biomes_are_independent() {
        let mut chunk = overworld();
        assert!(chunk.set_block(2, 70, 2, 1, 6));
        assert_eq!(chunk.get_block(2, 70, 2, 0), Some(AIR));
        assert_eq!(chunk.get_block(2, 70, 2, 1), Some(6));
        assert_eq!(chunk.get_biome(0, -64, 0), Some(PLAINS));
        assert!(chunk.set_biome(7, 319, 8, 15));
        assert_eq!(chunk.get_biome(7, 319, 8), Some(15));
        assert!(!chunk.set_biome(0, 320, 0, 5));
    }

    #[test]
    fn dirty_flag_tracks_actual_changes_and_save_reset() {
        let mut chunk = overworld();
        assert!(!chunk.set_block(1, 1, 1, 0, AIR));
        assert!(!chunk.is_dirty());
        assert!(chunk.set_block(1, 1, 1, 0, 4));
        assert!(chunk.is_dirty());
        chunk.mark_saved();
        assert!(!chunk.set_block(1, 1, 1, 0, 4));
        assert!(!chunk.is_dirty());
        assert!(chunk.get_sub_chunk_mut(0).is_some());
        assert!(chunk.is_dirty());
    }

    #[test]
    fn highest_non_air_subchunk_updates_after_block_removal() {
        let mut chunk = overworld();
        chunk.set_block(0, 20, 0, 0, 1);
        assert_eq!(chunk.highest_non_air_sub_chunk_y_relative(), 1);
        assert_eq!(chunk.highest_non_air_sub_chunk_y(), 6);
        chunk.set_block(0, 20, 0, 0, AIR);
        assert_eq!(chunk.highest_non_air_sub_chunk_y(), 0);
    }

    #[test]
    fn block_entities_are_coordinate_scoped_and_replaceable() {
        let mut chunk = overworld();
        assert!(chunk.block_entity_at(3, 64, 5).is_none());
        chunk.set_block_entity(3, 64, 5, vec![10, 0, 0, 0]);
        chunk.set_block_entity(3, 64, 5, vec![11, 0, 0, 0]);
        chunk.set_block_entity(4, 64, 5, vec![12]);
        assert_eq!(chunk.block_entities().len(), 2);
        assert_eq!(chunk.block_entity_at(3, 64, 5), Some(&[11, 0, 0, 0][..]));
        assert!(chunk.remove_block_entity(3, 64, 5));
        assert!(chunk.block_entity_at(3, 64, 5).is_none());
        assert_eq!(chunk.block_entity_at(4, 64, 5), Some(&[12][..]));
    }

    #[test]
    fn replacing_a_solid_block_removes_its_block_entity() {
        let mut chunk = Chunk::new(2, -1, -4, 24, AIR, PLAINS);
        chunk.set_block(3, 64, 5, 0, 9);
        let (x, y, z) = (2 * 16 + 3, 64, -16 + 5);
        chunk.set_block_entity(x, y, z, vec![1, 2, 3]);
        assert!(chunk.block_entity_at(x, y, z).is_some());
        assert!(!chunk.set_block(3, 64, 5, 0, 9));
        assert!(chunk.block_entity_at(x, y, z).is_some());
        assert!(chunk.set_block(3, 64, 5, 0, AIR));
        assert!(chunk.block_entity_at(x, y, z).is_none());
    }

    #[test]
    fn chunks_starting_at_zero_do_not_alias_negative_subchunks() {
        let mut chunk = Chunk::new(0, 0, 0, 8, AIR, PLAINS);
        assert!(chunk.get_sub_chunk(-1).is_none());
        assert!(chunk.get_sub_chunk(7).is_some());
        assert!(chunk.get_sub_chunk(8).is_none());
        assert!(!chunk.set_block(0, -1, 0, 0, 1));
        assert!(chunk.set_block(0, 127, 0, 0, 1));
    }
}
