//! Dragonfly's Rust-authoritative gameplay core.
//!
//! The Go server remains responsible for Bedrock transport and packet codecs.
//! This crate accepts versioned semantic events over a local Unix socket and
//! owns gameplay state. Do not add RakNet, NetherNet, JWT, encryption, or
//! Bedrock packet decoding here.

pub mod block_break;
pub mod block_id;
pub mod ipc;
pub mod player;

pub const CORE_PROTOCOL_VERSION: u16 = 1;
