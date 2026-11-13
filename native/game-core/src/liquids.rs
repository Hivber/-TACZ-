//! Selectively ported from MC-RS at da93a9501601fcdc284762fde88c24b44604991e.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).
//! Liquid state timing and geometry helpers using stable namespaced IDs.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LiquidKind { Water, Lava }

impl LiquidKind {
    pub const fn flow_tick_rate(self, in_nether: bool) -> u32 {
        match self { Self::Water => 5, Self::Lava => if in_nether { 10 } else { 30 } }
    }
    pub const fn flow_distance(self) -> u32 {
        match self { Self::Water => 8, Self::Lava => 4 }
    }
    pub const fn contact_damage(self) -> f32 {
        match self { Self::Water => 0.0, Self::Lava => 4.0 }
    }
    pub const fn fire_ticks_on_exit(self) -> u32 {
        match self { Self::Water => 0, Self::Lava => 15 * 20 }
    }
    pub const fn can_be_source(self, level: u8) -> bool { level == 0 }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LiquidLevel(pub u8);

impl LiquidLevel {
    pub const fn is_source(self) -> bool { self.0 == 0 }
    pub const fn is_flowing(self) -> bool { !self.is_source() }

    pub fn next_for(self, kind: LiquidKind) -> Option<Self> {
        let next = self.0.saturating_add(1);
        if next >= kind.flow_distance() as u8 { None } else { Some(Self(next)) }
    }
}

/// The single liquid_depth Bedrock property has a falling bit plus 3 decay
/// bits. This function accepts the semantic block identifier/state, not a
/// runtime palette ID whose value is version-specific.
pub fn water_height_percent(block_identifier: &str, liquid_depth: u8) -> Option<f32> {
    if !matches!(block_identifier, "minecraft:water" | "minecraft:flowing_water") { return None; }
    if liquid_depth > 15 { return None; }
    let falling = liquid_depth & 0b1000 != 0;
    let decay = if falling { 0 } else { liquid_depth & 0b0111 };
    Some((decay as f32 + 1.0) / 9.0)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test] fn water_flows_faster_than_lava() {
        assert!(LiquidKind::Water.flow_tick_rate(false) < LiquidKind::Lava.flow_tick_rate(false));
    }
    #[test] fn lava_contact_and_exit_effects_are_separate_from_water() {
        assert_eq!(LiquidKind::Lava.contact_damage(), 4.0);
        assert_eq!(LiquidKind::Lava.fire_ticks_on_exit(), 300);
        assert_eq!(LiquidKind::Water.contact_damage(), 0.0);
        assert_eq!(LiquidKind::Water.fire_ticks_on_exit(), 0);
    }
    #[test] fn water_depth_uses_falling_bit_and_three_decay_bits() {
        assert_eq!(water_height_percent("minecraft:water", 0), Some(1.0/9.0));
        assert_eq!(water_height_percent("minecraft:water", 7), Some(8.0/9.0));
        assert_eq!(water_height_percent("minecraft:flowing_water", 15), Some(1.0/9.0));
        assert_eq!(water_height_percent("minecraft:lava", 0), None);
        assert_eq!(water_height_percent("minecraft:water", 16), None);
    }
    #[test] fn liquid_levels_stop_at_type_specific_flow_distance() {
        assert_eq!(LiquidLevel(3).next_for(LiquidKind::Water), Some(LiquidLevel(4)));
        assert_eq!(LiquidLevel(3).next_for(LiquidKind::Lava), None);
    }
}
