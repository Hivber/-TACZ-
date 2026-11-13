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
