# Third-party notices

## Helix Bedrock world generator

- Project: https://github.com/helix-bedrock/worldgen
- Source commit: `6359abdbcc3c32eed3a65aaa0058d6b1775d89f7`
- License: GNU General Public License v3.0
- License text: `third_party/helix-worldgen/LICENSE`

The embedded Bedrock 1.26.60 block palette, block type tags, biome ID table,
light attenuation table, and translucency table under
`server/world/generator/helixdata/` are copied from the upstream project's
`1.26.60/overworld/testdata/` directory at that commit. They are data inputs
used to initialise the generator's block and biome registries.

## Helix data

- Project: https://github.com/helix-bedrock/helix-data
- Source commit: `24c4e8453ee3`
- License: GNU General Public License v3.0
- License text: `third_party/helix-data/LICENSE`

The server links against the Helix worldgen module, which depends on this data
module for the Bedrock 1.26.60 vanilla behaviour-pack data. Preserve both
license texts and upstream attribution when redistributing the combined server.


## Chorus block component model

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: c8115402a3bacf2014af2ce72fa5c35729040aa8
- License: Apache License 2.0
- Source paths: crates/block/src/component/, block_definition.rs,
  block_permutation.rs

The data-only component model and conditional permutation component merging in
native/game-core/src/components.rs and native/game-core/src/block_registry.rs
are adapted from the source above. No Chorus Bevy ECS, network plugin, packet
handler or transport runtime is included.

## Chorus block-state schema crate

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: `5808efd55dc51f38e7900ad2c72f14d73434b093`
- License: Apache License 2.0
- Original source: `crates/block_schema/src/lib.rs`
- Bundled resource: `crates/block/resources/block_upgrade_schemas.json`
- Vendored destination: `native/chorus/block_schema/`

The schema model and historical bundled upgrade data are copied into the
repository. The bundled dataset's latest entry is `1.21.60.33`;
it is not represented as a complete Bedrock 1.26.60 schema set. The upstream
Apache-2.0 license text is retained at
`third_party/licenses/Chorus-Apache-2.0.txt`.

## MC-RS gameplay modules

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: da93a9501601fcdc284762fde88c24b44604991e
- License: MIT
- License text: `third_party/licenses/MC-RS-MIT.txt`

The following modules are copied or adapted from MC-RS:
- `native/game-core/src/block_break.rs` (adapted from
  `crates/mc-rs-server/src/block_break.rs`)
- `native/game-core/src/block_attachment.rs` (adapted from
  `crates/mc-rs-server/src/block_attachment.rs`)
- `native/game-core/src/game_mode.rs` (adapted from
  `crates/mc-rs-server/src/gamemode.rs`)
- `native/game-core/src/entity_visibility.rs` (adapted from
  `crates/mc-rs-server/src/entity_culling.rs`)
- `native/game-core/src/hunger.rs`, `food.rs`, `entity_fall.rs`,
  `world_time.rs`, `world_border.rs`
- `native/game-core/src/difficulty.rs` (copied from
  `crates/mc-rs-server/src/difficulty.rs`)
- `native/game-core/src/despawn_rules.rs` (copied from
  `crates/mc-rs-server/src/despawn_rules.rs`)

MC-RS mining compatibility, survival helpers and reference rules are not
automatically exact Bedrock 1.26.60 behavior. MC-RS currently advertises
protocol 2169 / Bedrock 1.26.45; validate version-sensitive rules and data
before enabling them in live gameplay. Modified files retain source
attribution in their file headers.

## Bundled third-party license texts

- Chorus Apache License 2.0: `third_party/licenses/Chorus-Apache-2.0.txt`
- MC-RS MIT License: `third_party/licenses/MC-RS-MIT.txt`

The complete license texts are retained with this repository for redistribution.


## MC-RS combat and armor gameplay logic

The following pure gameplay modules are copied or adapted from the MC-RS
MIT-licensed source at revision \`da93a9501601fcdc284762fde88c24b44604991e\`:
- \`native/game-core/src/armor_tier.rs\` from
  \`crates/mc-rs-server/src/armor_tier.rs\` (selective copy)
- \`native/game-core/src/damage_sources.rs\` from
  \`crates/mc-rs-server/src/damage_sources.rs\` (selective copy)
- \`native/game-core/src/armor.rs\` (adapted armor-slot aggregation; replaces
  MC-RS protocol ItemStack/item-registry access with semantic material/slot inputs)
- \`native/game-core/src/damage_modifiers.rs\` (adapted from
  \`crates/mc-rs-server/src/damage_modifiers.rs\` and the armor reduction
  portion of \`armor.rs\`)
- \`native/game-core/src/combat.rs\` (adapted from
  \`crates/mc-rs-server/src/combat.rs\`; excludes EventManager, packet effects
  and entity registry wiring)

The source implements a selective PocketMine-MP port and does not by itself
establish parity with BDS 26.60. Validate armor, enchantment, effect, hurt-time
and knockback behavior against the target server before treating these
calculations as authoritative.


## MC-RS item durability, ore XP and wall state

Additional selectively ported/adapted gameplay modules from MC-RS revision
\`da93a9501601fcdc284762fde88c24b44604991e\` (MIT):
- \`native/game-core/src/item_durability.rs\` adapts
  \`crates/mc-rs-server/src/durability.rs\` to stable item identifiers and the
  local semantic ItemStack representation; no protocol ItemStack or network
  item IDs were copied.
- \`native/game-core/src/ore_xp.rs\` adapts
  \`crates/mc-rs-server/src/drop_xp.rs\` and resolves values by namespaced block
  and item identifiers instead of copied numeric runtime IDs.
- \`native/game-core/src/wall.rs\` adapts
  \`crates/mc-rs-server/src/block_wall.rs\` for data-only wall connection state.


## Chorus dropped-item entity gameplay

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Source: \`crates/item/src/item_entity.rs\`
- Destination: \`native/game-core/src/item_entity.rs\`

The item entity's pickup delay, despawn age, compatible stack absorption and
pickup reach rules are adapted to the local semantic ItemStack type. The Bevy
ECS component and glam vector dependencies are not included.


## Chorus shared block-state definitions

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Source: \`crates/block/src/state/common.rs\`
- Destination: \`native/game-core/src/common_block_states.rs\`

All 164 shared state definitions are adapted to the owned Rust state model.
State identifiers, value ordering and enum labels are retained; framework
macros and atomicow/Bevy dependencies are excluded.


## Chorus chunk-palette and biome primitives

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Sources: \`crates/level/src/bit_array.rs\`,
  \`crates/level/src/biome/biome_id.rs\`
- Destinations: \`native/game-core/src/bit_array.rs\`,
  \`native/game-core/src/biome_id.rs\`

The bit-array implementation is adapted with checked construction and index
helpers. The biome constants are copied as data; the rest of Chorus Level's
storage and packet codec integration are not included in this slice.


## Chorus block and biome palette implementation

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Source: \`crates/level/src/palette.rs\`
- Destination: \`native/game-core/src/palette.rs\`

Uniform/indexed palette state, palette counts, packed indices and disk
serialization helpers are adapted to standard-library collections and the
local BitArray. The Bedrock ProtoCodec implementation is intentionally omitted.


## Chorus chunk and subchunk storage model

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Sources: \`crates/level/src/sub_chunk.rs\`,
  \`crates/level/src/chunk.rs\`
- Destinations: \`native/game-core/src/sub_chunk.rs\`,
  \`native/game-core/src/chunk.rs\`

Chunk/subchunk palette state, block layers, biome state, block entity coordinate
management and dirty-state tracking are adapted. Bevy, nbtx and Bedrock packet
codec integration are excluded; block-entity payloads remain opaque NBT bytes.


## Chorus immutable section store

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Source: \`crates/level/src/section_store.rs\`
- Destination: \`native/game-core/src/section_store.rs\`

Adapted the immutable, Arc-shared chunk section store using the local BitArray
and Palette types. The local implementation includes bounds-checked APIs and
omits the upstream thread-local scratch-buffer reuse optimization.


## Chorus shared Bedrock world-generation RNG and noise primitives

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Sources: \`crates/worldgen/src/overworld/shared/random.rs\`,
  \`crates/worldgen/src/overworld/shared/noise.rs\`
- Destinations: \`native/game-core/src/chorus_random.rs\`,
  \`native/game-core/src/chorus_noise.rs\`

The modules preserve the source's vanilla seed derivation, Legacy/Xoroshiro
random implementations, positional factories, Perlin/Simplex and octave
noise models. The noise import was redirected to the vendored random module.


## Chorus Bedrock biome, terrain and shared worldgen algorithms

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Sources: \`crates/worldgen/src/overworld/shared/biome.rs\`,
  \`biome_builder.rs\`, \`spline.rs\`, \`packed.rs\`, \`scratch.rs\`,
  \`terrain.rs\`
- Destinations: \`native/game-core/src/chorus_biome.rs\`,
  \`chorus_biome_builder.rs\`, \`chorus_spline.rs\`, \`chorus_packed.rs\`,
  \`chorus_scratch.rs\`, \`chorus_terrain.rs\`

The shared random/noise and Bedrock biome IDs are rewired to their local
vendored modules. The upstream Bedrock block registry, feature placement,
surface decorators, cave carvers, and chunk serializer are excluded from this
slice and remain adapter/integration work.


## Chorus aquifer fluid simulation

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Source: \`crates/worldgen/src/overworld/java/aquifer.rs\`
- Destination: \`native/game-core/src/chorus_aquifer.rs\`

The aquifer fluid model is adapted to the locally ported Terrain and
PositionalFactory implementations. It uses std HashMap in place of the
upstream FxHashMap dependency.


## Chorus overworld cave carver

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Source: \`crates/worldgen/src/overworld/java/carver.rs\`
- Destination: \`native/game-core/src/chorus_carver.rs\`

The carver uses the local Legacy random source, terrain vertical bounds and
scratch buffer. Its carving mask and interval visiting are transport-independent.


## Chorus world-generation block tags

- Project: https://github.com/bedrock-crustaceans/Chorus
- Source revision: \`5808efd55dc51f38e7900ad2c72f14d73434b093\`
- License: Apache License 2.0
- Source: \`crates/worldgen/src/overworld/shared/tags.rs\`
- Destination: \`native/game-core/src/chorus_tags.rs\`

The 67 tag variants and their block identifier memberships are adapted to the
local \`block_id\` constants.


## MC-RS movement, weather and item cooldown gameplay

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: \`da93a9501601fcdc284762fde88c24b44604991e\`
- License: MIT
- Sources: \`movement_calc.rs\`, \`weather_state.rs\`,
  \`item_cooldown.rs\`, \`drop_reasons.rs\`
- Destinations: \`native/game-core/src/movement_calc.rs\`,
  \`weather_state.rs\`, \`item_cooldown.rs\`, \`drop_reasons.rs\`

Weather transitions are adapted to the local deterministic RandomSource instead
of thread_rng to make seeded world simulation reproducible.


## MC-RS entity attributes and potion effects

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: \`da93a9501601fcdc284762fde88c24b44604991e\`
- License: MIT
- Sources: \`crates/mc-rs-server/src/attribute.rs\`,
  \`crates/mc-rs-server/src/effects.rs\`
- Destinations: \`native/game-core/src/attribute.rs\`,
  \`native/game-core/src/effects.rs\`

The attribute map and effect manager are copied as a cohesive pair; MC-RS's
selected PocketMine-MP behavior should be checked against BDS 26.60 before
the semantic values are wired to live Bedrock attribute/effect packets.


## MC-RS projectiles, experience orbs and block physics

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: \`da93a9501601fcdc284762fde88c24b44604991e\`
- License: MIT
- Sources: \`experience_orb.rs\`, \`projectiles.rs\`, \`beacon_effects.rs\`,
  \`slime_block.rs\`, \`honey_block.rs\`; adapted \`anvil_damage.rs\`
- Destinations: matching files under \`native/game-core/src/\`.

The anvil state mapping was modernized to identifiers and its degradation
probability uses the local RandomSource. The other modules are selectively
ported without MC-RS session/entity registries or transport dependencies.


## MC-RS item attributes and horse attributes

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: \`da93a9501601fcdc284762fde88c24b44604991e\`
- License: MIT
- Sources: \`item_attributes.rs\`, \`horse_attributes.rs\`
- Destinations: \`native/game-core/src/item_attributes.rs\`,
  \`native/game-core/src/horse_attributes.rs\`

Horse random generation is adapted to the local RandomSource and no OS/thread
randomness or network code is included.


## MC-RS liquid and item-stack operations

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: \`da93a9501601fcdc284762fde88c24b44604991e\`
- License: MIT
- Sources: \`liquids.rs\`, \`item_stack_merge.rs\`
- Destinations: \`native/game-core/src/liquids.rs\`,
  \`native/game-core/src/stack_ops.rs\`

The liquid property decoder is adapted to stable identifiers and semantic
state values; the network/runtime-ID-dependent chunk palette decoder was
intentionally not included. Stack merging preserves the local ItemStack's
metadata and opaque user data.


## MC-RS furnace, campfire and basic crafting recipes

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: \`da93a9501601fcdc284762fde88c24b44604991e\`
- License: MIT
- Sources: \`furnace_fuel.rs\`, \`furnace_recipes.rs\`,
  \`campfire_recipes.rs\`, \`crafting_recipes.rs\`
- Destinations: matching files under \`native/game-core/src/\`.

These source data tables are ported as pure gameplay recipes. MC-RS packet
CraftingData payload generation and wire recipe identity are excluded.


## MC-RS elytra equipment state

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: da93a9501601fcdc284762fde88c24b44604991e
- License: MIT
- Adapted source: `crates/mc-rs-server/src/elytra.rs`
- Destination: `native/game-core/src/elytra.rs`

The elytra equipment state, durability, gliding lifecycle and firework boost
countdown are adapted into a deterministic host-neutral model using the
shared injected RNG. Bedrock glide velocity, pitch lift, drag, collision and
firework impulse physics are not included in this slice.

## MC-RS Adventure, climbing and composter rules

The following standalone gameplay rules are adapted from the MC-RS MIT-licensed
source at revision `da93a9501601fcdc284762fde88c24b44604991e`:
- `native/game-core/src/adventure_mode.rs` from
  `crates/mc-rs-server/src/adventure_mode.rs`; whitelist decisions only,
  leaving Bedrock NBT parsing in the host adapter.
- `native/game-core/src/climb.rs` from
  `crates/mc-rs-server/src/climb.rs`; semantic climbable-block rules and
  movement constants, with collision and velocity application left to the host.
- `native/game-core/src/composter.rs` from
  `crates/mc-rs-server/src/composter.rs`; item composting probabilities and
  persistent level state, adapted to the repository's injected deterministic
  `RandomSource` rather than thread-local randomness.

These are reusable gameplay calculations/state models; they are not yet wired
as authoritative handlers into the live Go session/world runtime.
