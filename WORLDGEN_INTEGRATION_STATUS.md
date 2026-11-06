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
