# World generation integration status

## Current default

The server now defaults to the upstream Helix Bedrock 1.26.60 vanilla terrain
generator. The generator is selected through the [World] section of
`config.toml`:

```toml
[World]
Generator = "helix"
Seed = 0
Folder = "world_helix"
SaveData = true
```

Supported values:

- `helix`: upstream Bedrock 1.26.60 overworld, nether, and end generation.
- `pnx`: the Dragonfly-native PNX-style generator retained as an alternative.
- `flat`: the legacy flat generator.

## Integrated path

- The Helix 1.26.60 upstream module and its compatible block palette, block
  type, biome, light, and translucency data are included in the source tree.
- The adapter constructs all three vanilla dimension generators from the
  configured seed.
- Generated Bedrock block states and biome IDs are translated to Dragonfly
  runtime IDs before a chunk is inserted into a world.
- Waterlogged blocks are written to Dragonfly's secondary block layer.
- Supported structure block entities are converted to Dragonfly chunk NBT.
  Coordinates are translated from world space to chunk-local X/Z when reading
  the block state, while the saved block entity position remains in world space.
- The normal Dragonfly chunk loading path recognises generators that provide
  block entity data, so generated structures enter the same persistence path as
  other chunks.

## Compatibility boundaries

Dragonfly does not currently expose every Bedrock block state or block entity.
The adapter maps exact block states where possible, falls back to a known
default state for supported blocks, and maps unavailable block states to air.
The startup log reports exact, defaulted, and missing state counts. Structure
block entities are retained only when the resulting Dragonfly block implements
the NBT interface; unsupported ones are skipped and counted.

The generator is responsible for new chunks only. Chunks already stored in a
world database are loaded as saved and are not regenerated when the generator
setting changes. Use a new world folder when switching from PNX or flat to
Helix to avoid joining different terrain generators at existing chunk edges.
The default folder is therefore `world_helix`; the previous `world` folder is
left untouched.

## Verification

The source includes smoke tests that instantiate Helix and generate chunks in
the overworld, nether, and end, plus a regression test for block entity
coordinates on non-origin chunks. GitHub Actions runs formatting/lint checks,
Go tests, and generated-code verification. Actual Bedrock 1.26.60 client
validation is still required for terrain appearance, structure block entities,
dimension portals, and long-distance chunk seams.


## Source audit (2026-11-13)

The selected `helix` generator is **not the PNX/Java-style generator**. The pinned
Go dependency is `github.com/helix-bedrock/worldgen` at
`v0.0.0-20261028023607-6359abdbcc3c` (upstream commit
`6359abdbcc3c32eed3a65aaa0058d6b1775d89f7`). Its upstream project explicitly
describes itself as Bedrock world generation derived from BDS behaviour and
compared against BDS-generated chunks. It contains separate versioned
`1.26.60` Overworld, Nether and End packages.

**Important Chorus finding:** Chorus has both `overworld/java` and
`overworld/bedrock` modules, but its current `Bedrock` edition is not an
independent Bedrock terrain implementation. `overworld/bedrock/mod.rs`
constructs `Java::create(...)` and delegates noise filling, surface building,
carvers, decoration and shapes to that Java implementation. Its
`bds_parity.rs` is an ignored comparison harness requiring a real BDS world
database; it is not a Bedrock generator. Therefore, do not replace Helix with
Chorus's `Bedrock` wrapper or import Chorus's Java worldgen into the vanilla
Bedrock path.

**Version-parity qualification:** Helix's pinned upstream README says all
three dimensions are generated and most parts match BDS, while a refinement
pass remains. Its status document says the 1.26.60 folder was carried forward
from BDS 1.26.52, with selected 1.26.60 changes checked against BDS 1.26.60;
it does not claim every 1.26.60 feature has been compared block-for-block.
This fork's existing smoke tests prove only that chunks generate and contain
blocks. They do not prove seed parity with BDS.

**Licensing blocker:** this repository's top-level license is MIT, but the
pinned `helix-bedrock/worldgen` module declares GPL-3.0. Do not assume the
combined distributed server can continue to be represented as MIT-only.
Before distributing binaries, resolve the dependency's copyleft obligations
with appropriate legal review, obtain a compatible alternative licence, or
replace it with an independently implemented compatible generator. No
license change is made by this audit.

## World-generation migration gates

1. Keep Helix as the current Bedrock-oriented generator; do not silently
   switch the default to PNX or Chorus Java-derived generation.
2. Capture deterministic reference chunks from BDS 1.26.60 for fixed seeds
   and coordinates, including all three dimensions, block states, biome
   volumes, block entities and structures.
3. Compare this fork's generated chunks against those references and report
   block/biome/height/structure parity separately. Smoke tests are not parity
   tests.
4. Only port or replace the generator after a Rust implementation passes the
   same reference corpus. Keep Dragonfly's packet/network path unchanged.
5. Treat the GPL-3.0 dependency as a release blocker until its distribution
   implications are resolved.
