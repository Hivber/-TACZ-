# Upstream Rust Server Migration Audit

Audit date: 2026-11-13

Target repository: https://github.com/Hivber/-TACZ-  
Target branch: `dev-settings`  
Target base commit inspected: `5634ddcabe9ea9def04e7ba94b73af8d05ebccab`

Upstream snapshots inspected:

- MC-RS: https://github.com/pedrokarim/mc-rs
  - `main` tree: `da93a9501601fcdc284762fde88c24b44604991e`
  - License: MIT
  - Advertised protocol: 2169 / Bedrock 1.26.45
  - Workspace has separate proto, RakNet, crypto, command, NBT and web UI crates, but its gameplay implementation is concentrated in `crates/mc-rs-server`.
  - README explicitly describes the codebase as a selective PocketMine-MP port. Do not assume every algorithm or data table is exact BDS behavior.
- Chorus: https://github.com/bedrock-crustaceans/Chorus
  - `main` tree: `5808efd55dc51f38e7900ad2c72f14d73434b093`
  - License: Apache-2.0
  - README advertises Bedrock 26.60 / protocol v2225, matching this Dragonfly fork's gophertunnel target.
  - Workspace has modular block, block-schema, core, entity, item, level and worldgen crates, plus separate network, RakNet and NetherNet crates.

## Migration boundary

- Keep Dragonfly/gophertunnel as the only Bedrock network and protocol path.
- Do not import MC-RS RakNet, packet codec, crypto, connection/session runtime or packet encoders.
- Do not import Chorus network, RakNet, NetherNet plugins or its client packet dispatch runtime.
- Import gameplay code only where its dependencies can be isolated from transport. Where a gameplay module relies on upstream world, entity, ECS or registry types, port the required data model and adapter together instead of copying a file that cannot compile.
- Keep one authoritative gameplay state. The current Go server still owns live gameplay; the Rust IPC currently receives shadow/mirror state and is not yet the live authority.

## MC-RS classification

### Already present in native/game-core

The target tree already contains MC-RS-derived or adapted modules for mining, block attachment classification, game-mode capabilities, entity visibility decisions, hunger, food values, fall damage, world time and conceptual world borders. The existing migration documentation correctly marks several of these as helpers that are not yet wired into live Dragonfly handlers.

### Copied in this audit batch

- `difficulty.rs`: standalone enum and difficulty helper functions; no upstream packet, socket, ECS or external-crate dependency.
- `despawn_rules.rs`: standalone despawn eligibility/radius helpers; no upstream packet, socket, ECS or external-crate dependency.

Both modules retain upstream attribution and tests. They are reference logic only until their values and behavior are checked against Bedrock 1.26.60 and connected to Rust-owned world/entity ticking.

### Adapt before importing

- `block_behaviors.rs`: transport-independent but explicitly ports selected PocketMine-MP gravity, crop, leaf, snow and ice rules. It requires real block-neighbor, biome, light and random-tick inputs; its simplified temperature table is not suitable as authoritative vanilla data.
- `block_liquid.rs`, `block_entities.rs`, `block_light_update.rs`, `chunk_ticking.rs`: require the source world's block/chunk/entity storage and scheduling model.
- `combat.rs`, `damage_modifiers.rs`, `survival.rs`, `armor.rs`: depend on MC-RS attribute/effect/event types and need adaptation to the target Rust state model. Do not copy them alone.
- `crafting.rs`, container and inventory transaction handlers: depend on MC-RS item registries, recipes, transaction/session state and packet-facing request models. Port semantic gameplay rules and tests, not packet handlers.
- Mob AI, spawning, movement and world generation: large groups of mutually dependent modules; migrate with their state/components and data, not as isolated files.
- MC-RS world-generation and block/drop JSON must be version-audited. MC-RS targets 1.26.45; its sampled block/drop data must not be silently treated as 1.26.60 data.

### Exclude

- `mc-rs-proto`, `mc-rs-raknet`, `mc-rs-crypto`.
- MC-RS connection/session runtime, Bedrock packet encoders/decoders, packet IDs and handlers.
- MC-RS web UI and server process/console integration unless a separate requirement calls for them.

## Chorus classification

### Strong candidates for coordinated migration

- `crates/block` + `crates/block_schema`: block definitions, permutations, components and block registry. This is the best next source group for broad block-mechanics coverage.
- `crates/item`: item registry, item stacks and item entities.
- `crates/entity`: ECS entity model and components.
- `crates/level`: chunks, palettes, subchunks, dimensions and storage.
- `crates/worldgen`: generators and biome/worldgen algorithms.

These are modular source crates, but they are not dependency-free. Their manifests use Bevy ECS/tasks, bedrock-rs types, nbtx, glam and other shared dependencies. Import them as a compatible crate group or adapt their types at a clear boundary; blindly copying one source directory will not compile.

### Shared foundation to inspect during crate import

- `crates/core` supplies shared types and scheduling/configuration support used by the gameplay crates.
- `crates/block_schema` and bundled resources are required by block definitions/upgrades.
- The exact required bedrock-rs features must be narrowed so no Chorus network transport or packet runtime is activated.

### Exclude from the Dragonfly-hosted runtime

- `crates/network`, `crates/raknet`, `crates/nethernet`.
- `src/network` and Chorus server startup/transport wiring.
- Root command/player/world orchestration should be extracted selectively only after checking whether each module is gameplay logic or packet/session glue.

## Recommended order

1. Import Chorus block + block-schema data and registry as a buildable local crate group; keep Dragonfly packet translation in Go.
2. Import Chorus item and entity models, then expose semantic operations through the existing IPC contract.
3. Import Chorus level/chunk storage and worldgen as a coherent world subsystem.
4. Port selected MC-RS algorithms and tests where they add behavior not already implemented by Chorus.
5. Only after these crates build and pass parity tests, move live player/world state authority from Go to Rust.

## Current result

This audit batch adds two self-contained MC-RS helper modules. It does not claim that either upstream server has been transplanted wholesale, that gameplay is already Rust-authoritative, or that 1.26.60 client compatibility has been verified.
