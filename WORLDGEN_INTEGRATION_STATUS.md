# Helix Bedrock worldgen integration status

The user approved GPL-3.0-compatible integration of `github.com/helix-bedrock/worldgen`.

**This archive does not yet contain that integration.** The active world generator remains the existing Dragonfly-native PNX-inspired generator (`Generator = "pnx"`) or the legacy flat generator (`Generator = "flat"). No Helix source or Helix dependency has been added in this interim build.

The required next work is a tested adapter between Helix's 1.26.60 block-state palette, biome IDs, structure block entities/entity spawns and Dragonfly's chunk/column model. A simple block-copy adapter would silently lose structure data, so it is not treated as a finished integration.

Upstream reference: `github.com/helix-bedrock/worldgen`, commit `6359abdbcc3c32eed3a65aaa0058d6b1775d89f7` (2026-10-28). Its module declares Go 1.26.0 and GPL-3.0. Any later archive that actually incorporates it must include the GPL-3.0 license, preserve upstream notices, document modifications, and provide the corresponding source when distributed.
