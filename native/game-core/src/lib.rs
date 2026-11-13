//! Dragonfly's Rust-authoritative gameplay core.
//!
//! The Go server remains responsible for Bedrock transport and packet codecs.
//! This crate owns data-driven gameplay state and calculations. Do not add
//! RakNet, NetherNet, JWT, encryption, compression, or Bedrock packet decoding.

pub mod block_break;
pub mod armor_tier;
pub mod armor;
pub mod damage_sources;
pub mod damage_modifiers;
pub mod combat;
pub mod block_attachment;
pub mod block_drops;
pub mod entity_visibility;
pub mod game_mode;
pub mod entity_fall;
pub mod food;
pub mod hunger;
pub mod world_border;
pub mod world_time;
pub use chorus_block_schema as block_schema;
pub mod difficulty;
pub mod despawn_rules;
pub mod block_interaction;
pub mod block_id;
pub mod block_registry;
pub mod block_state;
pub mod components;
pub mod inventory;
pub mod ipc;
pub mod item_stack;
pub mod player;
pub mod stack_size;
pub mod random;

pub const CORE_PROTOCOL_VERSION: u16 = 2;
