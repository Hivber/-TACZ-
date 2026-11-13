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
