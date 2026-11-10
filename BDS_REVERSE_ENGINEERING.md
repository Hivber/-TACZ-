# BDS reverse-engineering research plan

This document defines an evidence-led research workflow for comparing Dragonfly
with the official Minecraft Bedrock Dedicated Server (BDS). It does not claim
that BDS source code is public or that reconstructed declarations are Mojang's
original source.

Target compatibility baseline: Bedrock 1.26.60. Every observation must record
the exact BDS/client build, platform, binary SHA-256, tool version and test
conditions. Do not transfer raw addresses or offsets between builds.

## Public research tools

### Bedrock Lens

Repository: https://github.com/bedrock-mc/lens

Bedrock Lens indexes versioned Bedrock binaries and symbol information, uses
Ghidra/PyGhidra and Ghidra BSim for cross-build function similarity, and stores
artifact hashes and analysis metadata. It is the preferred starting point for
cross-version function research.

A BSim match is a candidate, not proof of equivalent semantics. Confirm each
candidate using call sites, referenced strings/constants, control flow and
runtime behaviour. The repository's packaged corpus and supported platform
must be checked before assuming that a particular 1.26.60 artifact is available.

### LeviLamina

Repository: https://github.com/LiteLDev/LeviLamina
Documentation: https://docs.levimc.org/LeviLamina/developer_guides/architecture/

LeviLamina exposes reconstructed Bedrock declarations under its mc headers and
provides native hooking/mod APIs. These headers are valuable structural
references, but they are not a complete original BDS source tree. Check the
supported BDS build and ABI before relying on a declaration or layout.

### BDS Modding Devkit / mapping

Repository: https://github.com/pmmp/bds-modding-devkit
Mapping: https://github.com/pmmp/bds-mod-mapping

The Devkit documents symbol export, protocol metadata extraction and Frida
packet tracing. The upstream PMMP Devkit repository was archived on 2026-08-08;
treat it as a reference/tooling source, not an actively maintained dependency.
Its README also warns that BDS updates can break mapping mods.

The PocketMine protocol update documentation explains that useful protocol
data has historically come from binary inspection, BDS mods and packet traces:
https://github.com/pmmp/doc.pmmp.io/blob/rtfd/source/developers/internals-docs/updating-minecraft-protocol.rst

## Evidence rules

For every researched mechanism, record:

1. **Artifact identity** — BDS version/build, platform, executable SHA-256 and
   symbol/PDB identity where available.
2. **Static evidence** — function candidates, call graph, fields, constants,
   strings and decompiled control flow, each tied to an artifact.
3. **Runtime evidence** — reproducible input, packet trace or observed state
   transition, with client/server versions and test conditions.
4. **Behavioural specification** — preconditions, state changes, timing,
   side effects, persistence and edge cases.
5. **Dragonfly mapping** — exact Go packages/files, current behaviour, gaps and
   proposed implementation.
6. **Regression tests** — deterministic unit/integration tests and a clear
   expected result.
7. **Validation status** — distinguish static hypothesis, runtime-confirmed,
   implemented, CI-passed and Bedrock-client-tested.

Never copy proprietary BDS code into Dragonfly. Reimplement independently
from observed behaviour and permitted public interfaces. Do not describe a
reconstructed header, decompiler output or guessed struct layout as original
Mojang source.

## First research tracks

### 1. Entity replication and observer lifecycle

Dragonfly starting points:

- `server/entity/movement.go`
- `server/session/world.go`
- `server/session/session.go`
- `server/world/tick.go`
- `server/world/viewer.go`

Questions to answer against BDS traces:

- Which actor state changes are sent as absolute movement, relative movement,
  motion, metadata or dedicated packets?
- What is the exact ordering of actor add, initial metadata/equipment, movement,
  dimension transfer and actor removal?
- When an actor crosses a chunk boundary, how are old and new observers
  reconciled?
- What happens when a viewer enters or leaves an actor's tracking range?
- Which fields are sent periodically, only on change, or after a forced
  resynchronisation?

Required tests: same-chunk movement, cross-chunk movement, observer range
entry/exit, teleport, dimension transfer, spawn/remove ordering and reconnect.

### 2. Chunk delivery and client cache

Dragonfly starting points:

- `server/session/chunk.go`
- `server/session/session.go`
- `server/world/loader.go`
- `server/world/chunk/`

Compare chunk/subchunk payload selection, cache-hit/miss handling, request
windows, ordering, retries, and invalidation after block/biome changes. Measure
bytes and latency in addition to packet counts. Preserve exact payload
ownership when sharing encoded buffers.

### 3. Block state and redstone

Compare runtime block states, neighbour update ordering, scheduled ticks,
redstone propagation, piston resolution, comparator inputs and block-entity
side effects. Record timing in game ticks and test circuit interactions, not
just isolated blocks.

### 4. Entity AI and world simulation

Compare tick scheduling, navigation/path costs, target selection, collision,
despawn/spawn rules and simulation-distance behaviour. Do not parallelise
Dragonfly world transactions merely because BDS uses multiple threads; first
establish ownership boundaries and deterministic ordering.

### 5. Persistence

Compare player/entity/chunk save triggers, serialization fields, migration and
crash recovery. Any asynchronous write design must preserve ordering and
durability guarantees.

## Current Dragonfly baseline

At the time this plan was added, the `dev-settings` branch had passed its
latest CI run for commit `390d15326cdde330f55a9f393326987db31cfb99`.
That verifies the repository's configured checks only; it does not prove
Bedrock 1.26.60 runtime parity or performance on Android/ZeroTermux.

The first code audit found that `MovementComputer.TickMovement` captures
viewers from the entity's starting position, while `tickEntities` reconciles
viewer membership when the entity's stored chunk position changes. This is a
specific lifecycle path to validate with boundary-crossing tests and BDS packet
traces; it is not, by itself, evidence of a confirmed bug.

## Delivery policy

Work in small, testable batches on `dev-settings` only. Read the current branch
head before every commit, build on its current tree, and update the branch with
a non-forced expected-SHA check. A failed CI run must be diagnosed and repaired
before dependent work is treated as validated.

Do not mark a mechanism complete based solely on compilation, a matching
function name, an available item/block identifier, or a successful CI run.
