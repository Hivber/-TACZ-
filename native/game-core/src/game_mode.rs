//! Game-mode capability rules adapted from MC-RS.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! Revision: da93a9501601fcdc284762fde88c24b44604991e
//! Original: crates/mc-rs-server/src/gamemode.rs
//! License: MIT; see ../../third_party/licenses/MC-RS-MIT.txt.

use serde::{Deserialize, Serialize};

/// Bedrock game-mode identifiers used by Dragonfly/gophertunnel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[repr(u8)]
#[serde(rename_all = "snake_case")]
pub enum GameMode {
    #[default]
    Survival = 0,
    Creative = 1,
    Adventure = 2,
    Spectator = 3,
}

impl GameMode {
    /// Converts the protocol/game-state identifier. Unknown values safely fall
    /// back to Survival rather than accidentally granting creative abilities.
    pub const fn from_id(value: u8) -> Self {
        match value {
            1 => Self::Creative,
            2 => Self::Adventure,
            3 => Self::Spectator,
            _ => Self::Survival,
        }
    }

    pub const fn id(self) -> u8 {
        self as u8
    }

    pub const fn identifier(self) -> &'static str {
        match self {
            Self::Survival => "survival",
            Self::Creative => "creative",
            Self::Adventure => "adventure",
            Self::Spectator => "spectator",
        }
    }

    pub const fn is_creative(self) -> bool {
        matches!(self, Self::Creative)
    }

    pub const fn is_spectator(self) -> bool {
        matches!(self, Self::Spectator)
    }

    pub const fn allows_flight(self) -> bool {
        matches!(self, Self::Creative | Self::Spectator)
    }

    pub const fn takes_damage(self) -> bool {
        matches!(self, Self::Survival | Self::Adventure)
    }

    /// Adventure-mode block permissions still require item CanDestroy/CanPlaceOn
    /// metadata and are intentionally not granted by this coarse capability.
    pub const fn can_break_blocks(self) -> bool {
        matches!(self, Self::Survival | Self::Creative)
    }

    pub const fn can_place_blocks(self) -> bool {
        matches!(self, Self::Survival | Self::Creative)
    }

    pub const fn can_interact_with_entities(self) -> bool {
        !matches!(self, Self::Spectator)
    }

    pub const fn can_pick_up_items(self) -> bool {
        !matches!(self, Self::Spectator)
    }
}

impl From<u8> for GameMode {
    fn from(value: u8) -> Self {
        Self::from_id(value)
    }
}

impl From<GameMode> for u8 {
    fn from(value: GameMode) -> Self {
        value.id()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn identifiers_round_trip() {
        for mode in [GameMode::Survival, GameMode::Creative, GameMode::Adventure, GameMode::Spectator] {
            assert_eq!(GameMode::from_id(mode.id()), mode);
        }
        assert_eq!(GameMode::from_id(255), GameMode::Survival);
    }

    #[test]
    fn creative_and_spectator_flight_capabilities_are_distinct() {
        assert!(GameMode::Creative.allows_flight());
        assert!(GameMode::Spectator.allows_flight());
        assert!(!GameMode::Survival.allows_flight());
        assert!(!GameMode::Adventure.allows_flight());
    }

    #[test]
    fn spectator_is_non_interactive_and_immune_to_damage() {
        assert!(!GameMode::Spectator.takes_damage());
        assert!(!GameMode::Spectator.can_interact_with_entities());
        assert!(!GameMode::Spectator.can_pick_up_items());
    }

    #[test]
    fn adventure_requires_explicit_block_permissions() {
        assert!(!GameMode::Adventure.can_break_blocks());
        assert!(!GameMode::Adventure.can_place_blocks());
        assert!(GameMode::Survival.can_break_blocks());
    }
}
