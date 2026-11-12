# World-generation source audit

Date: 2026-11-13

## Decision

Use the existing Helix Bedrock 1.26.60 generator as the current terrain source.
Do not use MC-RS worldgen data or Chorus's Java-derived terrain implementation
as the authoritative Bedrock generator.

## Audited sources

- Dragonfly fork: `server/world/generator/helix.go` constructs Helix
  Overworld, Nether and End generators using the Bedrock 1.26.60 package,
  translates its block/biome identifiers into Dragonfly runtime identifiers,
  and copies generated chunks into Dragonfly's chunk format.
- Helix upstream: `helix-bedrock/worldgen`, pinned by this fork to
  `v0.0.0-20261028023607-6359abdbcc3c`, upstream commit
  `6359abdbcc3c32eed3a65aaa0058d6b1775d89f7`. Its README says it derives
  generation behaviour from BDS and compares output with BDS worlds. It
  generates all three dimensions. Its own status still lists refinement work
  and distinguishes exact from close features; some 1.26.60 changes have not
  been fully compared against BDS 1.26.60.
- Chorus: `crates/worldgen/src/overworld/bedrock/mod.rs` defines a Bedrock
  edition, but delegates `fill_noise`, `build_surface`, `apply_carvers`,
  `decorate` and `shapes` to `Java`, constructed by `Java::create`.
  Chorus's `bds_parity.rs` contains ignored tests that compare its generated
  chunks with an existing BDS database. That harness does not make the
  Java-derived generator Bedrock-native.
- MC-RS: its audited worldgen assets and algorithms are not a verified
  Bedrock 1.26.60 generator. Do not use its Java-style biome parameters,
  feature placement or random sequence as a replacement for Bedrock output.

## Current implementation limitations

The Dragonfly adapter currently has smoke tests for non-empty generation in
Overworld, Nether and End and a block-entity coordinate regression test.
Those tests do not compare output with BDS. The adapter also maps generated
block states to Dragonfly's registry: unsupported states may fall back to a
default state or air, and unsupported structure block entities may be
dropped. These mapping losses must be measured before claiming full parity.

## Licence review required

The fork's root `LICENSE` is MIT. The pinned Helix worldgen upstream
repository declares GPL-3.0. This is a material distribution/licensing
compatibility issue. Do not claim that the combined server remains MIT-only
without resolving it. No relicensing is performed here; obtain legal review,
seek a compatible licence, or replace the dependency with an independently
implemented alternative before binary distribution.

## Required next engineering steps

1. Add a reproducible BDS 1.26.60 reference-world corpus for fixed seeds and
   chunk coordinates in Overworld, Nether and End.
2. Add automated block-state, biome-volume, heightmap, block-entity and
   structure comparisons against that corpus.
3. Report exact/defaulted/missing block-state mapping counts and unsupported
   block-entity counts as compatibility metrics.
4. Port generator algorithms to Rust only behind the same reference tests.
   Do not import Chorus network/packet handling, and do not treat Chorus's
   current Bedrock wrapper as a Bedrock-native algorithm.
5. Keep the current Bedrock-oriented Helix generator selected until a
   replacement passes parity tests and its licensing is resolved.
