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
- Source paths: crates/block/src/component/, crates/block/src/block_definition.rs,
  crates/block/src/block_permutation.rs

The data-only component model and conditional permutation component merging in
native/game-core/src/components.rs and native/game-core/src/block_registry.rs
are adapted from the source above. No Chorus Bevy ECS, network plugin, packet
handler or transport runtime is included.

## MC-RS mining rules

- Project: https://github.com/pedrokarim/mc-rs
- Source revision: da93a9501601fcdc284762fde88c24b44604991e
- License: MIT
- Source path: crates/mc-rs-server/src/block_break.rs

Mining compatibility, break-time modifiers, Efficiency enchantment and tool
durability calculations in native/game-core/src/block_break.rs are adapted
from this source. MC-RS's 1.26.52.3 sampled block/drop JSON is not imported as
1.26.60 game data.
