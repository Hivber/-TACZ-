//! Dragonfly's Rust-authoritative gameplay core.
//!
//! The Go server remains responsible for Bedrock transport and packet codecs.
//! This crate owns data-driven gameplay state and calculations. Do not add
//! RakNet, NetherNet, JWT, encryption, compression, or Bedrock packet decoding.

pub mod block_break;
pub mod block_drops;
pub mod block_id;
pub mod block_registry;
pub mod block_state;
pub mod components;
pub mod ipc;
pub mod player;
pub mod random;

pub const CORE_PROTOCOL_VERSION: u16 = 1;
