# Dragonfly Rust gameplay core

The Rust crate is the intended authority for gameplay logic. Dragonfly's Go
runtime remains the Bedrock transport adapter:

- Go/gophertunnel owns RakNet/NetherNet, login/JWT, encryption, compression,
  packet IDs, packet serialization and connection lifecycle.
- Rust owns gameplay state and deterministic gameplay calculations. It receives
  semantic events only, never Bedrock packets or client sockets.
- The current crate contains MC-RS-derived mining, block-attachment,
  game-mode, visibility, survival, combat, damage-source, armor and world utility helpers; a Chorus-derived
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

## Imported Chorus block-schema crate

The Chorus `chorus_block_schema` crate is vendored at
`native/chorus/block_schema` with its bundled block-state upgrade schema
resource. The game core re-exports the crate as `block_schema` and can parse
the bundled schema bundle.

Important version boundary: the upstream bundle currently ends at block
version `1.21.60.33`. It is suitable only as historical
world-state migration input where applicable; it is not a complete or
authoritative Bedrock 1.26.60 upgrade table. Newer schemas must be sourced and
validated separately before applying them to live worlds.

## Dropped-item entity mechanics

The Chorus-derived item entity stores the semantic item stack, pickup delay,
age, on-ground state and opaque item data. It implements normal/thrown pickup
delays, 6,000-tick despawn lifetime, compatible stack merging, 3D pickup reach,
and persistence restore. The Bevy ECS component and glam vector dependencies
were removed; Dragonfly/Rust world scheduling remains responsible for ticking
and spawning these item states.

## Tool durability, ore XP and wall state

The item-survival slice adds identifier-based vanilla tool durability tiers,
per-point Unbreaking rolls, tool break detection, ore XP ranges and furnace
smelting XP, plus a wall connection/post state model. Durability uses the
Rust item model's metadata field as the durable damage value in this semantic
adapter; the live Go item conversion must preserve this convention before
these rules can be authoritative. MC-RS's ore XP was originally sampled by
legacy numeric block IDs; the Rust port keys by namespaced identifiers to
avoid treating runtime IDs as stable data. The result remains a source-derived
1.26.45 reference and requires 1.26.60 parity checks.

## Combat and armor baseline

`armor_tier.rs`, `damage_sources.rs`, `armor.rs`, `damage_modifiers.rs` and
`combat.rs` bring over the selective MC-RS/PocketMine-MP combat baseline using
transport-independent inputs. The semantic layer covers armor defense and
toughness, protection enchantment EPF, resistance, absorption hearts, spawn
invulnerability, hurt-time delta handling, death detection and knockback
velocity. It does not yet include Bedrock-specific attribute/effect packet
synchronization, shield blocking, thorns durability, plugin damage events,
weapon attack cooldown attributes, sweeping attacks or server-side entity
physics integration. Treat the equations as a versioned reference until
validated against BDS 26.60.

## Chorus cave carving algorithm

The cave carver now includes its seeded legacy random stream, worm/tunnel
carving, protected bedrock/top layers, and reusable carving mask. It emits
vertical carve intervals independent of any block registry; an eventual
terrain adapter applies those intervals through the Rust chunk storage while
keeping Go packet encoding unchanged.

## Chorus aquifer and fluid-level model

The Chorus aquifer implementation is ported against the local deterministic
positional RNG and Terrain sampler. It now provides aquifer grid caching,
surface-level hints, fluid status selection (air/water/lava), barrier and
pressure calculations, and lava pocket handling for terrain carving. It is a
gameplay/worldgen algorithm layer; no block palette lookup or network packets
are embedded in it. Its output has not yet been validated against BDS 26.60
seeded chunk captures.

## Chorus Bedrock biome and terrain primitives

The shared worldgen layer includes the 56-biome model and Bedrock numeric ID
mapping, vanilla climate parameter tables and biome source selection, cubic
overworld splines, packed integer arrays, reusable scratch buffers, and the
3D terrain density/noise-column sampler. These are the terrain-shape and
biome-sampling foundations used by Chorus's Bedrock edition wrapper. Feature
placement, surface decorators, cave carvers, block-registry mapping and final
chunk conversion still require a coordinated adapter and are not yet live.

## Chorus Bedrock worldgen noise core

The shared Chorus worldgen random and noise engine is now vendored as two
standalone modules: Java-compatible Legacy/Xoroshiro random sources,
128-bit seed upgrading, MD5 name hashing, positional random factories, block
seed functions, Improved Perlin, Simplex, octave NoiseStack, NormalNoise and
BlendedNoise. It carries no packet/session or Bedrock DB dependency. These
are the low-level primitives used by Chorus's Bedrock Edition terrain wrapper;
they are not yet a complete generator and do not replace the live Go/Helix
world generator. Seeded parity must be verified against BDS 26.60 chunk samples
before activation.

## Immutable chunk section store

The Chorus SectionStore layer is adapted as a contiguous metadata/value/packed-
word buffer with Arc-shared immutable snapshots, palette append/copy, direct
block-array conversion, bounds-checked reads and uniform-section fast paths.
The implementation currently builds from the existing Palette model rather
than sharing mutable chunk memory; it can become the immutable snapshot format
for chunk saving or asynchronous generation without pulling in Bevy/packet code.

## Chunk and subchunk gameplay storage

The Chorus Level data model now provides 16x16x16 subchunks with two block
layers, per-cell biome palettes, local-coordinate indexing, and an in-memory
chunk with configurable vertical range, block access, block entities and dirty
tracking. Network chunk/subchunk encoders are not copied; chunk packet framing
and the live Dragonfly world provider remain outside this Rust storage slice.
Block-entity NBT bytes are held opaquely with coordinates split into fields so
no NBT codec dependency is introduced.

## Chorus block/biome palette state

The adapted Palette model adds uniform and indexed 4,096-entry palettes,
palette index packing, value-frequency accounting, stable/lazy removal of dead
entries, width growth/shrink, array conversion and Chorus disk palette writes.
The Bedrock ProtoCodec layer is deliberately omitted, so this does not alter Go
packet encoding or imply a live chunk-storage cutover.

## Chunk palette and biome primitives

The Chorus level slice adds its 32-bit packed BitArray implementation with
Bedrock palette bit widths (1, 2, 3, 4, 5, 6, 8 and 16), widening on value
overflow, indexed access and conversion to/from packed words. This is a
protocol-independent storage primitive suitable for chunk palettes; it is
not yet wired into the live Go chunk sender/storage path. The legacy Bedrock
biome numeric identifier table is also included as data only.

## Chorus common block-state catalog

common_block_states.rs carries the 164 reusable common state definitions
from Chorus crates/block/src/state/common.rs into owned local
BlockStateDefinition values. It preserves upstream identifiers, value ordering
and enum labels without pulling in atomicow, Bevy or Bedrock packet types.
Callers can retrieve definitions by upstream constant name or collect all
definitions sharing a state identifier (some identifiers have multiple ranges).

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
