//! Survival block-drop resolution over the data-only component registry.
//! This accepts version-matched data and does not import MC-RS's 1.26.52.3
//! sampled block/drop tables into a 1.26.60 server.

use crate::components::{BlockComponents, LootComponent, LootStack};
use crate::random::RandomSource;

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DropStack {
    pub item: String,
    pub count: u32,
}

#[derive(Clone, Copy, Debug, Default)]
pub struct BlockDropContext {
    pub survival: bool,
    pub silk_touch: bool,
    pub fortune: u8,
}

fn roll_stack<R: RandomSource>(stack: &LootStack, fortune: u8, rng: &mut R) -> DropStack {
    let span = u32::from(stack.max_count) - u32::from(stack.min_count) + 1;
    let base = u32::from(stack.min_count) + rng.range_u32(span);
    let fortune_max = u32::from(stack.fortune_bonus_per_level).saturating_mul(u32::from(fortune));
    let bonus = if fortune_max == 0 { 0 } else { rng.range_u32(fortune_max.saturating_add(1)) };
    DropStack { item: stack.item.clone(), count: base.saturating_add(bonus) }
}

fn roll_stacks<R: RandomSource>(stacks: &[LootStack], fortune: u8, rng: &mut R) -> Vec<DropStack> {
    stacks.iter().map(|stack| roll_stack(stack, fortune, rng))
        .filter(|stack| stack.count > 0).collect()
}

/// Missing LootComponent means the vanilla default (drop the block itself).
/// An explicit empty LootComponent means no drops. Creative mode emits none.
pub fn resolve_block_drops<R: RandomSource>(
    block_identifier: &str,
    components: &BlockComponents,
    context: BlockDropContext,
    rng: &mut R,
) -> Vec<DropStack> {
    if !context.survival { return Vec::new(); }
    let Some(loot) = components.loot() else {
        return vec![DropStack { item: block_identifier.to_string(), count: 1 }];
    };
    if context.silk_touch {
        if let Some(items) = &loot.silk_touch { return roll_stacks(items, 0, rng); }
    }
    roll_weighted_loot(loot, context.fortune, rng)
}

fn roll_weighted_loot<R: RandomSource>(
    loot: &LootComponent,
    fortune: u8,
    rng: &mut R,
) -> Vec<DropStack> {
    let total_weight = loot.outcomes.iter()
        .fold(0u32, |total, outcome| total.saturating_add(outcome.weight));
    if total_weight == 0 { return Vec::new(); }
    let mut roll = rng.range_u32(total_weight);
    for outcome in &loot.outcomes {
        if roll < outcome.weight { return roll_stacks(&outcome.items, fortune, rng); }
        roll -= outcome.weight;
    }
    Vec::new()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::components::{BlockComponent, LootOutcome};

    struct FixedRng(u64);
    impl RandomSource for FixedRng {
        fn next_u64(&mut self) -> u64 {
            let value = self.0;
            self.0 = self.0.wrapping_add(1);
            value
        }
    }

    #[test]
    fn absent_loot_component_drops_the_block_itself() {
        let drops = resolve_block_drops(
            "minecraft:stone", &BlockComponents::new(),
            BlockDropContext { survival: true, ..Default::default() }, &mut FixedRng(0),
        );
        assert_eq!(drops, vec![DropStack { item: "minecraft:stone".into(), count: 1 }]);
    }

    #[test]
    fn explicit_nothing_and_creative_mode_drop_nothing() {
        let mut components = BlockComponents::new();
        components.insert(BlockComponent::Loot(LootComponent::nothing()));
        assert!(resolve_block_drops(
            "minecraft:stone", &components,
            BlockDropContext { survival: true, ..Default::default() }, &mut FixedRng(0),
        ).is_empty());
        assert!(resolve_block_drops(
            "minecraft:stone", &components,
            BlockDropContext { survival: false, ..Default::default() }, &mut FixedRng(0),
        ).is_empty());
    }

    #[test]
    fn weighted_loot_and_fortune_are_resolved_deterministically() {
        let loot = LootComponent::weighted([
            LootOutcome { weight: 1, items: vec![LootStack::fixed("minecraft:coal", 1).with_fortune_bonus(1)] },
            LootOutcome { weight: 1, items: vec![LootStack::fixed("minecraft:diamond", 1)] },
        ]);
        let mut components = BlockComponents::new();
        components.insert(BlockComponent::Loot(loot));
        let drops = resolve_block_drops(
            "minecraft:diamond_ore", &components,
            BlockDropContext { survival: true, fortune: 3, ..Default::default() }, &mut FixedRng(0),
        );
        assert_eq!(drops[0].item, "minecraft:coal");
        assert!((1..=4).contains(&drops[0].count));
    }

    #[test]
    fn silk_touch_table_overrides_normal_loot() {
        let loot = LootComponent::item("minecraft:diamond")
            .with_silk_touch([LootStack::fixed("minecraft:diamond_ore", 1)]);
        let mut components = BlockComponents::new();
        components.insert(BlockComponent::Loot(loot));
        let drops = resolve_block_drops(
            "minecraft:diamond_ore", &components,
            BlockDropContext { survival: true, silk_touch: true, fortune: 3 }, &mut FixedRng(0),
        );
        assert_eq!(drops, vec![DropStack { item: "minecraft:diamond_ore".into(), count: 1 }]);
    }
}
