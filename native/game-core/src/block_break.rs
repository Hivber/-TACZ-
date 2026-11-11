//! Bedrock block-mining calculations adapted from MC-RS.
//!
//! Source: https://github.com/pedrokarim/mc-rs
//! File: crates/mc-rs-server/src/block_break.rs
//! Upstream revision audited for this port: da93a9501601fcdc284762fde88c24b44604991e
//! MC-RS is MIT-licensed. This module ports the tool compatibility, BDS
//! break-time multiplier, environmental penalties and haste/fatigue modifiers.
//! Runtime block profiles and loot tables are not yet imported; callers must
//! supply authoritative profile data from the Dragonfly/Bedrock registry.

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

#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct MiningTool {
    pub tool_type: u8,
    pub harvest_level: u8,
    pub efficiency_correct: f32,
    pub efficiency_incorrect: f32,
}

impl MiningTool {
    pub fn mining_efficiency(self, correct_tool: bool) -> f32 {
        let value = if correct_tool { self.efficiency_correct } else { self.efficiency_incorrect };
        value.max(0.01)
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
    pub fn is_breakable(self) -> bool {
        self.hardness >= 0.0
    }

    pub fn breaks_instantly(self) -> bool {
        self.hardness == 0.0
    }

    pub fn is_tool_compatible(self, tool: MiningTool) -> bool {
        self.is_breakable()
            && (self.tool_type == tool_type::NONE
                || self.harvest_level == 0
                || (self.tool_type & tool.tool_type != 0 && tool.harvest_level >= self.harvest_level))
    }

    /// Mining duration in seconds, following the MC-RS/PMMP and BDS multiplier model.
    pub fn break_time(self, tool: MiningTool) -> f32 {
        if !self.is_breakable() {
            return f32::INFINITY;
        }
        let multiplier = if self.is_tool_compatible(tool) || !self.requires_correct_tool {
            COMPATIBLE_TOOL_MULTIPLIER
        } else {
            INCOMPATIBLE_TOOL_MULTIPLIER
        };
        let correct_tool = self.tool_type & tool.tool_type != 0;
        self.hardness.max(0.0) * multiplier / tool.mining_efficiency(correct_tool)
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
    if !profile.is_breakable() {
        return 0.0;
    }
    let mut break_time_ticks = profile.break_time(tool) * 20.0;
    if !miner.on_ground && !miner.flying {
        break_time_ticks *= 5.0;
    }
    if miner.underwater_without_aqua_affinity {
        break_time_ticks *= 5.0;
    }
    if break_time_ticks <= 0.0 {
        return 1.0;
    }
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

#[cfg(test)]
mod tests {
    use super::*;

    fn stone() -> BlockBreakProfile {
        BlockBreakProfile { hardness: 1.5, tool_type: tool_type::PICKAXE, harvest_level: 1, requires_correct_tool: true }
    }

    fn iron_pickaxe() -> MiningTool {
        MiningTool { tool_type: tool_type::PICKAXE, harvest_level: 2, efficiency_correct: 6.0, efficiency_incorrect: 1.0 }
    }

    #[test]
    fn correct_tool_breaks_stone_faster_than_hand() {
        let hand = MiningTool { efficiency_correct: 1.0, efficiency_incorrect: 1.0, ..Default::default() };
        assert!(stone().break_time(iron_pickaxe()) < stone().break_time(hand));
    }

    #[test]
    fn airborne_and_underwater_penalties_are_applied() {
        let base = MinerState { on_ground: true, ..Default::default() };
        let air = MinerState::default();
        let water = MinerState { on_ground: true, underwater_without_aqua_affinity: true, ..Default::default() };
        let p = stone();
        let tool = iron_pickaxe();
        assert!(break_progress_per_tick(p, tool, base) > break_progress_per_tick(p, tool, air));
        assert!(break_progress_per_tick(p, tool, base) > break_progress_per_tick(p, tool, water));
    }

    #[test]
    fn haste_and_fatigue_modify_progress() {
        let p = stone();
        let tool = iron_pickaxe();
        let base = MinerState { on_ground: true, ..Default::default() };
        let haste = MinerState { haste: 1, ..base };
        let fatigue = MinerState { mining_fatigue: 1, ..base };
        assert!(break_progress_per_tick(p, tool, haste) > break_progress_per_tick(p, tool, base));
        assert!(break_progress_per_tick(p, tool, fatigue) < break_progress_per_tick(p, tool, base));
    }

    #[test]
    fn unbreakable_blocks_never_progress() {
        let p = BlockBreakProfile { hardness: -1.0, ..stone() };
        assert_eq!(break_progress_per_tick(p, iron_pickaxe(), MinerState::default()), 0.0);
    }
}
