//! Dropped-item entity state adapted from Chorus crates/item/src/item_entity.rs.
//! Upstream: https://github.com/bedrock-crustaceans/Chorus
//! Revision: 5808efd55dc51f38e7900ad2c72f14d73434b093; Apache-2.0.
//! ECS and glam types are replaced with host-neutral semantic state.

use crate::item_stack::ItemStack;

pub const PICKUP_DELAY_TICKS: u32 = 10;
pub const THROWN_PICKUP_DELAY_TICKS: u32 = 40;
pub const PICKUP_RADIUS: f32 = 1.75;
pub const DESPAWN_TICKS: u32 = 6_000;

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ItemEntity {
    stack: ItemStack,
    pickup_delay: u32,
    age: u32,
    pub on_ground: bool,
    user_data: Vec<u8>,
}

impl ItemEntity {
    pub fn new(stack: ItemStack) -> Self {
        Self {
            stack,
            pickup_delay: PICKUP_DELAY_TICKS,
            age: 0,
            on_ground: false,
            user_data: Vec::new(),
        }
    }

    pub fn thrown(stack: ItemStack) -> Self {
        Self {
            pickup_delay: THROWN_PICKUP_DELAY_TICKS,
            ..Self::new(stack)
        }
    }

    pub fn with_user_data(mut self, user_data: Vec<u8>) -> Self {
        self.user_data = user_data;
        self
    }

    pub fn user_data(&self) -> &[u8] { &self.user_data }
    pub fn stack(&self) -> &ItemStack { &self.stack }
    pub fn age(&self) -> u32 { self.age }
    pub fn pickup_delay(&self) -> u32 { self.pickup_delay }
    pub fn can_be_picked_up(&self) -> bool { self.pickup_delay == 0 }

    /// Increment age and report whether the 5-minute despawn lifetime elapsed.
    pub fn tick_age(&mut self) -> bool {
        self.age = self.age.saturating_add(1);
        self.age >= DESPAWN_TICKS
    }

    pub fn tick_pickup_delay(&mut self) {
        self.pickup_delay = self.pickup_delay.saturating_sub(1);
    }

    /// Merge another compatible dropped stack into this entity. Both stacks
    /// must have identical identity, metadata and opaque user data.
    pub fn absorb(&mut self, other: &mut ItemEntity, maximum_stack_size: u16) -> u16 {
        if !self.stack.same_kind(&other.stack) || self.user_data != other.user_data {
            return 0;
        }
        let moved = maximum_stack_size
            .saturating_sub(self.stack.count)
            .min(other.stack.count);
        if moved == 0 { return 0; }
        self.stack.count += moved;
        other.stack.count -= moved;
        self.age = self.age.min(other.age);
        self.pickup_delay = self.pickup_delay.max(other.pickup_delay);
        moved
    }

    /// Restore persisted age and pickup delay without replaying spawn defaults.
    pub fn restore(stack: ItemStack, age: u32, pickup_delay: u32) -> Self {
        Self {
            stack,
            pickup_delay,
            age,
            on_ground: false,
            user_data: Vec::new(),
        }
    }
}

pub fn within_pickup_reach(item: [f32; 3], player: [f32; 3]) -> bool {
    if item.iter().chain(player.iter()).any(|value| !value.is_finite()) { return false; }
    let dx = item[0] - player[0];
    let dy = item[1] - player[1];
    let dz = item[2] - player[2];
    dx * dx + dy * dy + dz * dz <= PICKUP_RADIUS * PICKUP_RADIUS
}

#[cfg(test)]
mod tests {
    use super::*;

    fn stack(count: u16, metadata: u32) -> ItemStack {
        ItemStack::new("minecraft:diamond", count, metadata, vec![]).unwrap()
    }

    #[test]
    fn spawned_and_thrown_items_have_different_pickup_delays() {
        let normal = ItemEntity::new(stack(1, 0));
        let thrown = ItemEntity::thrown(stack(1, 0));
        assert_eq!(normal.pickup_delay(), PICKUP_DELAY_TICKS);
        assert_eq!(thrown.pickup_delay(), THROWN_PICKUP_DELAY_TICKS);
    }

    #[test]
    fn pickup_delay_counts_down_and_age_despawns_after_six_thousand_ticks() {
        let mut item = ItemEntity::new(stack(1, 0));
        for _ in 0..PICKUP_DELAY_TICKS { item.tick_pickup_delay(); }
        assert!(item.can_be_picked_up());
        for _ in 0..DESPAWN_TICKS - 1 { assert!(!item.tick_age()); }
        assert!(item.tick_age());
    }

    #[test]
    fn absorb_merges_compatible_stack_and_carries_earlier_age_and_longer_delay() {
        let mut target = ItemEntity::new(stack(40, 0));
        let mut source = ItemEntity::thrown(stack(40, 0));
        target.age = 100;
        source.age = 20;
        assert_eq!(target.absorb(&mut source, 64), 24);
        assert_eq!(target.stack().count, 64);
        assert_eq!(source.stack().count, 16);
        assert_eq!(target.age(), 20);
        assert_eq!(target.pickup_delay(), THROWN_PICKUP_DELAY_TICKS);
    }

    #[test]
    fn absorb_rejects_different_metadata_or_user_data() {
        let mut target = ItemEntity::new(stack(1, 0));
        let mut metadata = ItemEntity::new(stack(1, 1));
        let mut nbt = ItemEntity::new(stack(1, 0)).with_user_data(vec![1]);
        assert_eq!(target.absorb(&mut metadata, 64), 0);
        assert_eq!(target.absorb(&mut nbt, 64), 0);
        assert_eq!(target.stack().count, 1);
    }

    #[test]
    fn pickup_reach_uses_three_dimensional_distance_and_rejects_non_finite_positions() {
        assert!(within_pickup_reach([0.0, 0.0, 0.0], [1.0, 0.0, 0.0]));
        assert!(!within_pickup_reach([0.0, 0.0, 0.0], [2.0, 0.0, 0.0]));
        assert!(!within_pickup_reach([f32::NAN, 0.0, 0.0], [0.0, 0.0, 0.0]));
    }
}
