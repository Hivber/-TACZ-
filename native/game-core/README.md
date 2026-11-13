# Dragonfly Rust gameplay core

The Rust crate is the intended authority for gameplay logic. Dragonfly's Go
runtime remains the Bedrock transport adapter:

- Go/gophertunnel owns RakNet/NetherNet, login/JWT, encryption, compression,
  packet IDs, packet serialization and connection lifecycle.
- Rust owns gameplay state and deterministic gameplay calculations. It receives
  semantic events only, never Bedrock packets or client sockets.
- The current crate contains MC-RS-derived mining, block-attachment,
  game-mode, visibility, survival and world utility helpers; a Chorus-derived
  data-only block component/permutation registry; inventory shadow-state
  primitives; and the persistent IPC prototype.

## Build and test

    cargo fmt --manifest-path native/game-core/Cargo.toml
    cargo test --manifest-path native/game-core/Cargo.toml
    cargo build --release --manifest-path native/game-core/Cargo.toml

The IPC bridge currently mirrors semantic state and is not yet the authoritative
live gameplay runtime. Do not synchronously dial the socket from the Go game
tick; runtime integration must use a persistent connection, bounded queues,
backpressure and explicit failure handling before gameplay is switched to Rust.

## Upstream migration audit

See [UPSTREAM_MIGRATION_AUDIT.md](UPSTREAM_MIGRATION_AUDIT.md) for the inspected
MC-RS and Chorus directory map, transport exclusions, dependency constraints,
version caveats and migration order.

The newly copied `difficulty.rs` and `despawn_rules.rs` are standalone
reference helpers from MC-RS. They are not yet connected to live world/entity
ticks and are not claimed as verified Bedrock 1.26.60 behavior.

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

## Upstream source and version cautions

- Chorus, Apache-2.0: data-only block component and conditional permutation
  concepts adapted from its block component, definition and permutation modules.
- MC-RS, MIT: mining, game-mode, visibility, survival, world utility,
  difficulty and despawn helpers are pinned to
  `da93a9501601fcdc284762fde88c24b44604991e`.
- MC-RS targets protocol 2169 / Bedrock 1.26.45. Its version-specific data
  must not be treated as authoritative 1.26.60 data.
- Chorus advertises protocol v2225 / Bedrock 26.60, but its modular gameplay
  crates still require coordinated dependency and type integration.
- MC-RS's RakNet, protocol, crypto, connection and packet runtime and Chorus's
  Bevy/network transport plugins are intentionally excluded.

This is a tested gameplay-core development slice, not yet the authoritative
runtime for live Dragonfly player sessions. Existing Go gameplay paths remain
in place until semantic event parity tests and a real 1.26.60 client regression
run pass.
