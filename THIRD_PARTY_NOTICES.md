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
