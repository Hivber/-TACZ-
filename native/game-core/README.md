# Dragonfly Rust gameplay core

The Rust crate is the intended authority for gameplay logic. Dragonfly's Go
runtime remains the Bedrock transport adapter:

- Go/gophertunnel owns RakNet/NetherNet, login/JWT, encryption, compression,
  packet IDs, packet serialization and connection lifecycle.
- Rust owns gameplay state and deterministic gameplay calculations. It receives
  semantic events only, never Bedrock packets or client sockets.
- The current crate contains player lifecycle state, a Chorus-derived data-only
  block component/permutation registry, MC-RS-derived mining calculations,
  Efficiency/Unbreaking handling, and weighted/silk-touch/Fortune drop resolution.

## Build and test

    cargo fmt --manifest-path native/game-core/Cargo.toml
    cargo test --manifest-path native/game-core/Cargo.toml
    cargo build --release --manifest-path native/game-core/Cargo.toml

The IPC prototype uses a four-byte big-endian length followed by one versioned
JSON request/response, with a 1 MiB frame ceiling. It is not yet a tick-safe
live-session integration. Do not synchronously dial the socket from the Go
game tick; runtime integration must use a persistent connection, bounded
queues, backpressure and explicit failure handling before gameplay is switched
to Rust authority.

## Block components and permutations

BlockDefinition supplies default components and ordered conditional
BlockPermutationDefinition overrides. Every generated permutation receives a
component set assembled from defaults and all matching overrides. Later
components of the same kind replace earlier ones, matching Chorus component
storage semantics. Identical override-match sets share an Arc<BlockComponents>.

The component model includes mineable hardness (including item-specific
hardness), loot, collision boxes, flammability, friction, light, map color,
piston movement, replaceability, solidity and transparency. A permutation
index is local to this crate; it is not a Bedrock runtime ID or palette index.

BlockBreakProfile derives hardness and tool requirements from the active
permutation. The mining model includes correct-tool/harvest-level checks,
Efficiency, Haste, Mining Fatigue, airborne and underwater penalties, plus
tool durability and Unbreaking. resolve_block_drops supports explicit weighted
outcomes, optional silk-touch outcomes and configurable Fortune bonus ranges. The mining state machine handles start,
continue, per-tick progress, crack stages, target/reach validation, abort and completion.
Without a LootComponent, survival defaults to dropping the block itself; an
explicit empty LootComponent means no drops.

## Upstream source and version cautions

- Chorus, Apache-2.0: data-only block component and conditional permutation
  concepts adapted from crates/block/src/component/, block_definition.rs and
  block_permutation.rs at revision
  c8115402a3bacf2014af2ce72fa5c35729040aa8.
- MC-RS, MIT: mining speed, tool compatibility, potion modifiers and durability
  algorithms adapted from crates/mc-rs-server/src/block_break.rs at revision
  da93a9501601fcdc284762fde88c24b44604991e.
- MC-RS's sampled block_break.json and block_drops_bds.json were recorded for
  Bedrock 1.26.52.3. They are deliberately not treated as authoritative 1.26.60
  data. The generic loot resolver requires version-matched block definitions
  before it can represent vanilla drops exhaustively.
- MC-RS's RakNet, protocol, crypto, connection and packet runtime and Chorus's
  Bevy/network plugins are intentionally excluded.

This is a tested gameplay-core vertical slice, not yet the authoritative runtime
for live Dragonfly player sessions. Existing Go gameplay paths remain in place
until semantic event parity tests and a real 1.26.60 client regression run pass.
