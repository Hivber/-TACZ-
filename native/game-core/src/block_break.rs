//! Bedrock block-mining calculations adapted from MC-RS.
//! Source: https://github.com/pedrokarim/mc-rs
//! File: crates/mc-rs-server/src/block_break.rs
//! Upstream revision: da93a9501601fcdc284762fde88c24b44604991e
//! MIT. Ports tool compatibility, BDS break-time multipliers, haste/fatigue,
//! Efficiency enchantment, item-specific hardness and tool wear.
//! The 1.26.52.3 sampled block/drop JSON is deliberately not used for 1.26.60.

use crate::components::{BlockComponents, MineableComponent};
use crate::random::RandomSource;

pub mod tool_type {
    pub const NONE: u8 = 0;
    pub const SWORD: u8 = 1 << 0;
    pub const SHOVEL: u8 = 1 << 1;
    pub const PICKAXE: u8 = 1 << 2;
    pub const AXE: u8 = 1 << 3;
    pub const SHEARS: u8 = 1 << 4;
    pub const HOE: u8 = 1 << 5;
}

const COMPATIBLE_TOOL_MULTIPLIER: f32 = 1.5;
const INCOMPATIBLE_TOOL_MULTIPLIER: f32 = 5.0;

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct MiningTool {
    pub tool_type: u8,
    pub harvest_level: u8,
    pub efficiency_correct: f32,
    pub efficiency_incorrect: f32,
    pub efficiency_enchantment: u8,
    pub silk_touch: bool,
    pub fortune: u8,
    pub unbreaking: u8,
}

impl Default for MiningTool {
    fn default() -> Self {
        Self {
            tool_type: tool_type::NONE, harvest_level: 0,
            efficiency_correct: 1.0, efficiency_incorrect: 1.0,
            efficiency_enchantment: 0, silk_touch: false, fortune: 0, unbreaking: 0,
        }
    }
}

impl MiningTool {
    pub fn hand() -> Self { Self::default() }

    /// MC-RS/PMMP: Efficiency adds (level² + 1) times the tool's correct-tool bonus.
    pub fn mining_efficiency(self, correct_tool: bool) -> f32 {
        let value = if correct_tool {
            let level = f32::from(self.efficiency_enchantment);
            let bonus = if level > 0.0 { level.powi(2) + 1.0 } else { 0.0 };
            self.efficiency_correct + bonus * self.efficiency_incorrect
        } else {
            self.efficiency_incorrect
        };
        if value.is_finite() { value.max(0.01) } else { 0.01 }
    }
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct BlockBreakProfile {
    pub hardness: f32,
    pub tool_type: u8,
    pub harvest_level: u8,
    pub requires_correct_tool: bool,
}

impl BlockBreakProfile {
    pub fn from_mineable(component: &MineableComponent, item: Option<&str>) -> Self {
        Self {
            hardness: component.hardness_for_item(item),
            tool_type: component.tool_type,
            harvest_level: component.harvest_level,
            requires_correct_tool: component.requires_correct_tool,
        }
    }

    pub fn from_components(components: &BlockComponents) -> Option<Self> {
        components.mineable().map(|component| Self::from_mineable(component, None))
    }

    pub fn from_components_for_item(components: &BlockComponents, item: Option<&str>) -> Option<Self> {
        components.mineable().map(|component| Self::from_mineable(component, item))
    }

    pub fn is_breakable(self) -> bool { self.hardness >= 0.0 }
    pub fn breaks_instantly(self) -> bool { self.hardness == 0.0 }

    pub fn is_tool_compatible(self, tool: MiningTool) -> bool {
        self.is_breakable() && (
            self.tool_type == tool_type::NONE || self.harvest_level == 0
                || (self.tool_type & tool.tool_type != 0 && tool.harvest_level >= self.harvest_level)
        )
    }

    pub fn break_time(self, tool: MiningTool) -> f32 {
        if !self.is_breakable() { return f32::INFINITY; }
        if self.breaks_instantly() { return 0.0; }
        let multiplier = if self.is_tool_compatible(tool) || !self.requires_correct_tool {
            COMPATIBLE_TOOL_MULTIPLIER
        } else {
            INCOMPATIBLE_TOOL_MULTIPLIER
        };
        let correct_tool = self.tool_type & tool.tool_type != 0;
        self.hardness * multiplier / tool.mining_efficiency(correct_tool)
    }
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct MinerState {
    pub on_ground: bool,
    pub flying: bool,
    pub underwater_without_aqua_affinity: bool,
    /// Potion amplifier + 1; zero means no Haste.
    pub haste: u8,
    /// Potion amplifier + 1; zero means no Mining Fatigue.
    pub mining_fatigue: u8,
}

/// Block-breaking progress added per game tick (20 ticks/second).
pub fn break_progress_per_tick(profile: BlockBreakProfile, tool: MiningTool, miner: MinerState) -> f32 {
    if !profile.is_breakable() { return 0.0; }
    let mut break_time_ticks = profile.break_time(tool) * 20.0;
    if !miner.on_ground && !miner.flying { break_time_ticks *= 5.0; }
    if miner.underwater_without_aqua_affinity { break_time_ticks *= 5.0; }
    if break_time_ticks <= 0.0 { return 1.0; }
    if !break_time_ticks.is_finite() { return 0.0; }
    let mut progress = 1.0 / break_time_ticks;
    if miner.haste > 0 {
        let level = f32::from(miner.haste);
        progress *= (1.0 + 0.2 * level) * 1.2_f32.powf(level);
    }
    if miner.mining_fatigue > 0 {
        progress *= 0.21_f32.powf(f32::from(miner.mining_fatigue));
    }
    progress
}

/// Durability cost from MC-RS/PMMP Item::onDestroyBlock.
pub fn destroy_damage(profile: BlockBreakProfile, tool: MiningTool) -> u32 {
    match tool.tool_type {
        tool_type::SHEARS => 1,
        _ if profile.breaks_instantly() => 0,
        tool_type::SWORD => 2,
        tool_type::PICKAXE | tool_type::AXE | tool_type::SHOVEL | tool_type::HOE => 1,
        _ => 0,
    }
}

/// Number of durability points cancelled by Unbreaking.
pub fn unbreaking_reduction(amount: u32, unbreaking: u8, rng: &mut impl RandomSource) -> u32 {
    if unbreaking == 0 { return 0; }
    let cancellation_chance = 1.0 / (f32::from(unbreaking) + 1.0);
    (0..amount).filter(|_| rng.next_f32() > cancellation_chance).count() as u32
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::components::{BlockComponent, BlockComponents, MineableComponent};

    fn stone() -> BlockBreakProfile {
        BlockBreakProfile {
            hardness: 1.5, tool_type: tool_type::PICKAXE,
            harvest_level: 1, requires_correct_tool: true,
        }
    }

    fn iron_pickaxe() -> MiningTool {
        MiningTool {
            tool_type: tool_type::PICKAXE, harvest_level: 2,
            efficiency_correct: 6.0, efficiency_incorrect: 1.0,
            ..Default::default()
        }
    }

    struct FixedRng(f32);
    impl RandomSource for FixedRng {
        fn next_u64(&mut self) -> u64 {
            (self.0.clamp(0.0, 0.999_999_94) * u64::MAX as f32) as u64
        }
        fn next_f32(&mut self) -> f32 { self.0 }
    }

    #[test]
    fn correct_tool_breaks_stone_faster_than_hand() {
        assert!(stone().break_time(iron_pickaxe()) < stone().break_time(MiningTool::hand()));
    }

    #[test]
    fn efficiency_enchantment_increases_correct_tool_speed() {
        let base = iron_pickaxe();
        let enchanted = MiningTool { efficiency_enchantment: 3, ..base };
        assert!(stone().break_time(enchanted) < stone().break_time(base));
    }

    #[test]
    fn airborne_and_underwater_penalties_are_applied() {
        let base = MinerState { on_ground: true, ..Default::default() };
        let air = MinerState::default();
        let water = MinerState { on_ground: true, underwater_without_aqua_affinity: true, ..Default::default() };
        assert!(break_progress_per_tick(stone(), iron_pickaxe(), base) > break_progress_per_tick(stone(), iron_pickaxe(), air));
        assert!(break_progress_per_tick(stone(), iron_pickaxe(), base) > break_progress_per_tick(stone(), iron_pickaxe(), water));
    }

    #[test]
    fn haste_and_fatigue_modify_progress() {
        let base = MinerState { on_ground: true, ..Default::default() };
        let haste = MinerState { haste: 1, ..base };
        let fatigue = MinerState { mining_fatigue: 1, ..base };
        assert!(break_progress_per_tick(stone(), iron_pickaxe(), haste) > break_progress_per_tick(stone(), iron_pickaxe(), base));
        assert!(break_progress_per_tick(stone(), iron_pickaxe(), fatigue) < break_progress_per_tick(stone(), iron_pickaxe(), base));
    }

    #[test]
    fn unbreakable_blocks_never_progress() {
        let profile = BlockBreakProfile { hardness: -1.0, ..stone() };
        assert_eq!(break_progress_per_tick(profile, iron_pickaxe(), MinerState::default()), 0.0);
        assert!(profile.break_time(iron_pickaxe()).is_infinite());
    }

    #[test]
    fn block_profile_is_derived_from_active_permutation_components() {
        let mut components = BlockComponents::new();
        components.insert(BlockComponent::Mineable(
            MineableComponent::hardness(3.0)
                .with_tool_requirement(tool_type::PICKAXE, 2)
                .with_item_hardness("minecraft:diamond_pickaxe", 0.7),
        ));
        assert_eq!(BlockBreakProfile::from_components(&components).unwrap().hardness, 3.0);
        assert_eq!(BlockBreakProfile::from_components_for_item(&components, Some("minecraft:diamond_pickaxe")).unwrap().hardness, 0.7);
    }

    #[test]
    fn unbreaking_reduction_uses_the_random_chance_threshold() {
        let mut no_reduction = FixedRng(0.0);
        let mut full_reduction = FixedRng(0.99);
        assert_eq!(unbreaking_reduction(10, 3, &mut no_reduction), 0);
        assert_eq!(unbreaking_reduction(10, 3, &mut full_reduction), 10);
    }

    #[test]
    fn shears_take_durability_even_for_instant_blocks() {
        let instant = BlockBreakProfile { hardness: 0.0, ..stone() };
        assert_eq!(destroy_damage(instant, MiningTool { tool_type: tool_type::SHEARS, ..Default::default() }), 1);
        assert_eq!(destroy_damage(instant, iron_pickaxe()), 0);
    }
}
