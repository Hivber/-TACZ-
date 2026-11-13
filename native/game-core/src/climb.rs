//! Climbable-block rules adapted from MC-RS.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! Source: crates/mc-rs-server/src/climb.rs
//! Revision: da93a9501601fcdc284762fde88c24b44604991e (MIT).
//!
//! These are semantic movement constants; collision detection and velocity
//! application remain responsibilities of the world/entity adapter.

pub const CLIMB_SPEED: f32 = 0.2;
pub const SCAFFOLDING_DESCEND_SPEED: f32 = 0.1;
pub const CLIMB_FALL_DAMAGE_IMMUNITY_DIST: f32 = 0.0;

pub fn is_climbable(block_identifier: &str) -> bool {
    matches!(
        block_identifier,
        "minecraft:ladder"
            | "minecraft:vine"
            | "minecraft:scaffolding"
            | "minecraft:twisting_vines"
            | "minecraft:twisting_vines_plant"
            | "minecraft:weeping_vines"
            | "minecraft:weeping_vines_plant"
            | "minecraft:cave_vines"
            | "minecraft:cave_vines_plant"
    )
}

pub fn vertical_climb_speed(block_identifier: &str, sneaking: bool) -> Option<f32> {
    if !is_climbable(block_identifier) {
        return None;
    }
    if block_identifier == "minecraft:scaffolding" && sneaking {
        Some(-SCAFFOLDING_DESCEND_SPEED)
    } else {
        Some(CLIMB_SPEED)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn vanilla_climbable_blocks_are_recognized() {
        for block in [
            "minecraft:ladder",
            "minecraft:vine",
            "minecraft:scaffolding",
            "minecraft:twisting_vines",
            "minecraft:twisting_vines_plant",
            "minecraft:weeping_vines",
            "minecraft:weeping_vines_plant",
            "minecraft:cave_vines",
            "minecraft:cave_vines_plant",
        ] {
            assert!(is_climbable(block), "{block}");
        }
        assert!(!is_climbable("minecraft:stone"));
    }

    #[test]
    fn scaffolding_sneak_descends_and_other_climbables_ascend() {
        assert_eq!(vertical_climb_speed("minecraft:scaffolding", true), Some(-0.1));
        assert_eq!(vertical_climb_speed("minecraft:scaffolding", false), Some(0.2));
        assert_eq!(vertical_climb_speed("minecraft:ladder", true), Some(0.2));
        assert_eq!(vertical_climb_speed("minecraft:stone", false), None);
    }
}
