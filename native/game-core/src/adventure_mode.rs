//! Adventure-mode interaction restrictions adapted from MC-RS.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! Source: crates/mc-rs-server/src/adventure_mode.rs
//! Revision: da93a9501601fcdc284762fde88c24b44604991e (MIT).
//!
//! The host must still enforce these decisions when handling the corresponding
//! Bedrock actions. Item NBT parsing is deliberately kept in the Go adapter.

/// Adventure mode does not permit ordinary block breaking.
pub const CAN_BREAK_BLOCKS: bool = false;
/// Adventure mode does not permit ordinary block placement.
pub const CAN_PLACE_BLOCKS: bool = false;
/// Doors, containers and other interactable blocks remain usable.
pub const CAN_INTERACT: bool = true;
/// Entity attacks remain permitted.
pub const CAN_ATTACK: bool = true;

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct CanDestroyTag {
    pub blocks: Vec<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct CanPlaceOnTag {
    pub blocks: Vec<String>,
}

impl CanDestroyTag {
    pub fn allows(&self, block_identifier: &str) -> bool {
        self.blocks.iter().any(|block| block == block_identifier)
    }
}

impl CanPlaceOnTag {
    pub fn allows(&self, block_identifier: &str) -> bool {
        self.blocks.iter().any(|block| block == block_identifier)
    }
}

/// Resolve the Adventure-mode block-breaking rule for an item with an optional
/// CanDestroy whitelist. A whitelist is required; an empty/missing tag denies.
pub fn can_break_block(tag: Option<&CanDestroyTag>, block_identifier: &str) -> bool {
    tag.is_some_and(|tag| tag.allows(block_identifier))
}

/// Resolve the Adventure-mode placement rule for an item with an optional
/// CanPlaceOn whitelist. A whitelist is required; an empty/missing tag denies.
pub fn can_place_on(tag: Option<&CanPlaceOnTag>, block_identifier: &str) -> bool {
    tag.is_some_and(|tag| tag.allows(block_identifier))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ordinary_adventure_actions_are_restricted_but_interactions_remain() {
        assert!(!CAN_BREAK_BLOCKS);
        assert!(!CAN_PLACE_BLOCKS);
        assert!(CAN_INTERACT);
        assert!(CAN_ATTACK);
    }

    #[test]
    fn destroy_and_place_whitelists_match_exact_identifiers() {
        let destroy = CanDestroyTag { blocks: vec!["minecraft:stone".into()] };
        let place = CanPlaceOnTag { blocks: vec!["minecraft:dirt".into()] };
        assert!(can_break_block(Some(&destroy), "minecraft:stone"));
        assert!(!can_break_block(Some(&destroy), "minecraft:dirt"));
        assert!(!can_break_block(None, "minecraft:stone"));
        assert!(can_place_on(Some(&place), "minecraft:dirt"));
        assert!(!can_place_on(Some(&place), "minecraft:stone"));
        assert!(!can_place_on(None, "minecraft:dirt"));
    }
}
