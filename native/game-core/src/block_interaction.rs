//! Server-authoritative block-breaking state machine.
//!
//! This module produces semantic mining outcomes only. The Go adapter maps
//! crack stages, sounds, block updates and item spawns to gophertunnel packets.

use crate::block_break::{
    break_progress_per_tick, BlockBreakProfile, MinerState, MiningTool,
};
use serde::{Deserialize, Serialize};

#[derive(Clone, Copy, Debug, Eq, PartialEq, Hash, Serialize, Deserialize)]
pub struct BlockPos { pub x: i32, pub y: i32, pub z: i32 }

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct BlockTarget {
    pub position: BlockPos,
    /// Bedrock face values: 0 down, 1 up, 2 north, 3 south, 4 west, 5 east.
    pub face: u8,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum MiningError {
    InvalidFace, InvalidReach, OutOfReach, Unbreakable, NoActiveSession, TargetChanged,
}

#[derive(Clone, Copy, Debug, PartialEq, Serialize, Deserialize)]
#[serde(tag = "status", rename_all = "snake_case")]
pub enum MiningUpdate {
    Started { progress: f32, crack_stage: u8 },
    Progress { progress: f32, crack_stage: u8 },
    Completed,
    Aborted,
}

#[derive(Clone, Copy, Debug)]
struct ActiveMining {
    target: BlockTarget,
    progress: f32,
    last_tick: u64,
}

/// One active survival mining action for a player. The caller supplies the
/// authoritative block profile, held tool, effects, eye position and reach.
#[derive(Clone, Debug, Default)]
pub struct BlockMiningMachine { active: Option<ActiveMining> }

impl BlockMiningMachine {
    pub fn new() -> Self { Self::default() }
    pub fn is_active(&self) -> bool { self.active.is_some() }
    pub fn target(&self) -> Option<BlockTarget> { self.active.map(|active| active.target) }
    pub fn progress(&self) -> Option<f32> { self.active.map(|active| active.progress) }

    pub fn start(
        &mut self,
        target: BlockTarget,
        profile: BlockBreakProfile,
        eye_position: [f32; 3],
        max_reach: f32,
        tick: u64,
    ) -> Result<MiningUpdate, MiningError> {
        self.active = None;
        if target.face > 5 { return Err(MiningError::InvalidFace); }
        validate_reach(eye_position, max_reach)?;
        if !within_reach(target.position, eye_position, max_reach) {
            return Err(MiningError::OutOfReach);
        }
        if !profile.is_breakable() { return Err(MiningError::Unbreakable); }
        if profile.breaks_instantly() { return Ok(MiningUpdate::Completed); }

        self.active = Some(ActiveMining { target, progress: 0.0, last_tick: tick });
        Ok(MiningUpdate::Started { progress: 0.0, crack_stage: 0 })
    }

    /// Add one mining tick. Duplicate or older tick numbers cannot accelerate
    /// mining. Target changes and reach violations cancel the active session.
    pub fn advance(
        &mut self,
        target: BlockTarget,
        profile: BlockBreakProfile,
        tool: MiningTool,
        miner: MinerState,
        eye_position: [f32; 3],
        max_reach: f32,
        tick: u64,
    ) -> Result<MiningUpdate, MiningError> {
        if target.face > 5 {
            self.active = None;
            return Err(MiningError::InvalidFace);
        }
        if let Err(error) = validate_reach(eye_position, max_reach) {
            self.active = None;
            return Err(error);
        }
        let Some(mut active) = self.active else { return Err(MiningError::NoActiveSession); };
        if target.position != active.target.position {
            self.active = None;
            return Err(MiningError::TargetChanged);
        }
        if !within_reach(target.position, eye_position, max_reach) {
            self.active = None;
            return Err(MiningError::OutOfReach);
        }
        if !profile.is_breakable() {
            self.active = None;
            return Err(MiningError::Unbreakable);
        }
        if profile.breaks_instantly() {
            self.active = None;
            return Ok(MiningUpdate::Completed);
        }

        active.target.face = target.face;
        if tick > active.last_tick {
            active.progress = (active.progress + break_progress_per_tick(profile, tool, miner))
                .clamp(0.0, 1.0);
            active.last_tick = tick;
        }
        if active.progress >= 1.0 {
            self.active = None;
            return Ok(MiningUpdate::Completed);
        }
        self.active = Some(active);
        Ok(MiningUpdate::Progress {
            progress: active.progress,
            crack_stage: crack_stage(active.progress),
        })
    }

    pub fn abort(&mut self) -> MiningUpdate {
        self.active = None;
        MiningUpdate::Aborted
    }
}

fn validate_reach(eye_position: [f32; 3], max_reach: f32) -> Result<(), MiningError> {
    if !max_reach.is_finite() || max_reach <= 0.0
        || eye_position.iter().any(|coordinate| !coordinate.is_finite()) {
        return Err(MiningError::InvalidReach);
    }
    Ok(())
}

fn within_reach(position: BlockPos, eye: [f32; 3], reach: f32) -> bool {
    let center = [position.x as f32 + 0.5, position.y as f32 + 0.5, position.z as f32 + 0.5];
    let distance_squared = (center[0] - eye[0]).powi(2)
        + (center[1] - eye[1]).powi(2)
        + (center[2] - eye[2]).powi(2);
    distance_squared <= reach * reach
}

fn crack_stage(progress: f32) -> u8 {
    (progress * 10.0).floor().clamp(0.0, 9.0) as u8
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::block_break::tool_type;

    fn target() -> BlockTarget {
        BlockTarget { position: BlockPos { x: 0, y: 0, z: 0 }, face: 1 }
    }

    fn profile() -> BlockBreakProfile {
        BlockBreakProfile {
            hardness: 1.5, tool_type: tool_type::PICKAXE,
            harvest_level: 1, requires_correct_tool: true,
        }
    }

    fn tool() -> MiningTool {
        MiningTool {
            tool_type: tool_type::PICKAXE, harvest_level: 2,
            efficiency_correct: 8.0, efficiency_incorrect: 1.0,
            ..Default::default()
        }
    }

    fn eye() -> [f32; 3] { [0.5, 1.62, 3.0] }

    #[test]
    fn start_and_tick_produce_monotonic_crack_progress() {
        let mut machine = BlockMiningMachine::new();
        assert_eq!(machine.start(target(), profile(), eye(), 7.0, 10).unwrap(),
            MiningUpdate::Started { progress: 0.0, crack_stage: 0 });
        let miner = MinerState { on_ground: true, ..Default::default() };
        let first = machine.advance(target(), profile(), tool(), miner, eye(), 7.0, 11).unwrap();
        let MiningUpdate::Progress { progress: first_progress, crack_stage: first_stage } = first
            else { panic!("expected mining progress"); };
        assert!(first_progress > 0.0 && first_progress < 1.0);
        assert_eq!(first_stage, crack_stage(first_progress));
        let second = machine.advance(target(), profile(), tool(), miner, eye(), 7.0, 12).unwrap();
        let MiningUpdate::Progress { progress: second_progress, .. } = second
            else { panic!("expected mining progress"); };
        assert!(second_progress > first_progress);
    }

    #[test]
    fn duplicate_tick_does_not_accelerate_mining() {
        let mut machine = BlockMiningMachine::new();
        machine.start(target(), profile(), eye(), 7.0, 20).unwrap();
        let miner = MinerState { on_ground: true, ..Default::default() };
        let first = machine.advance(target(), profile(), tool(), miner, eye(), 7.0, 21).unwrap();
        let second = machine.advance(target(), profile(), tool(), miner, eye(), 7.0, 21).unwrap();
        assert_eq!(first, second);
    }

    #[test]
    fn abort_clears_crack_state() {
        let mut machine = BlockMiningMachine::new();
        machine.start(target(), profile(), eye(), 7.0, 1).unwrap();
        assert!(machine.is_active());
        assert_eq!(machine.abort(), MiningUpdate::Aborted);
        assert!(!machine.is_active());
        assert_eq!(machine.progress(), None);
    }

    #[test]
    fn out_of_reach_and_target_switch_cancel_mining() {
        let mut machine = BlockMiningMachine::new();
        machine.start(target(), profile(), eye(), 7.0, 1).unwrap();
        assert_eq!(machine.advance(target(), profile(), tool(), MinerState::default(),
            [100.0, 100.0, 100.0], 7.0, 2), Err(MiningError::OutOfReach));
        assert!(!machine.is_active());
        machine.start(target(), profile(), eye(), 7.0, 3).unwrap();
        let changed = BlockTarget { position: BlockPos { x: 1, y: 0, z: 0 }, face: 1 };
        assert_eq!(machine.advance(changed, profile(), tool(), MinerState::default(),
            eye(), 7.0, 4), Err(MiningError::TargetChanged));
        assert!(!machine.is_active());
    }

    #[test]
    fn invalid_faces_and_non_finite_reach_are_rejected() {
        let mut machine = BlockMiningMachine::new();
        let bad_face = BlockTarget { face: 6, ..target() };
        assert_eq!(machine.start(bad_face, profile(), eye(), 7.0, 1), Err(MiningError::InvalidFace));
        assert_eq!(machine.start(target(), profile(), eye(), f32::NAN, 1), Err(MiningError::InvalidReach));
    }

    #[test]
    fn unbreakable_blocks_are_rejected_and_instant_blocks_complete() {
        let mut machine = BlockMiningMachine::new();
        let unbreakable = BlockBreakProfile { hardness: -1.0, ..profile() };
        assert_eq!(machine.start(target(), unbreakable, eye(), 7.0, 1), Err(MiningError::Unbreakable));
        let instant = BlockBreakProfile { hardness: 0.0, ..profile() };
        assert_eq!(machine.start(target(), instant, eye(), 7.0, 1), Ok(MiningUpdate::Completed));
        assert!(!machine.is_active());
    }

    #[test]
    fn mining_completes_and_resets_session() {
        let mut machine = BlockMiningMachine::new();
        machine.start(target(), profile(), eye(), 7.0, 0).unwrap();
        let miner = MinerState { on_ground: true, ..Default::default() };
        let mut outcome = MiningUpdate::Started { progress: 0.0, crack_stage: 0 };
        for tick in 1..=100 {
            outcome = machine.advance(target(), profile(), tool(), miner, eye(), 7.0, tick).unwrap();
            if outcome == MiningUpdate::Completed { break; }
        }
        assert_eq!(outcome, MiningUpdate::Completed);
        assert!(!machine.is_active());
    }
}
