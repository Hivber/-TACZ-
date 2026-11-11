# Rust Game-Core Migration Plan

## Decision

The target server keeps Dragonfly's Go Bedrock network and protocol stack. Rust becomes the sole authority for gameplay state and simulation. MC-RS and Chorus are source projects for gameplay implementation; their network stacks are not part of the target runtime.

The migration is incremental. The existing Dragonfly server remains the compatibility baseline until each Rust-backed subsystem passes tests and a real Bedrock 1.26.60 client regression run.

## Audited source snapshots

| Project | Audited ref | Relevant facts | Intended use |
|---|---|---|---|
| This repository | `dev-settings` at `97653a3a75716c7a9d88f005e281e162f6f9c14a` | Dragonfly fork; Go module pins `github.com/sandertv/gophertunnel v1.63.0`; already contains `native/redstone-rs` | Preserve network/session protocol implementation and existing work |
| MC-RS | `main` at `da93a9501601fcdc284762fde88c24b44604991e` | MIT; README targets protocol 2169 / Bedrock 1.26.45; gameplay is concentrated in `mc-rs-server`, but packet handling and the server runtime are coupled to its own protocol crates | Reuse/adapt gameplay algorithms, data tables, tests and standalone subsystems; do not import its RakNet, crypto, packet codec or connection runtime |
| Chorus | `main` at `c8115402a3bacf2014af2ce72fa5c35729040aa8` | Apache-2.0; README advertises Bedrock 26.60 / protocol v2225; modular Bevy ECS crates for block, entity, item, level and world generation; its app also wires its own network/NetherNet plugins | Primary Rust ECS/game-core foundation and reusable gameplay modules; remove/bypass its network integration in the Dragonfly-hosted build |

### Protocol compatibility checkpoint

The pinned Dragonfly dependency, gophertunnel v1.63.0, declares `CurrentProtocol = 2225` and `CurrentVersion = "1.26.60"` in `minecraft/protocol/info.go`. Chorus's current README also advertises v2225 / 26.60. MC-RS currently advertises protocol 2169 / 1.26.45, so its protocol-specific assumptions and captured data must not be copied as if they were automatically 26.60-compatible.

This confirms the existing Dragonfly protocol dependency is aligned with 1.26.60; it does not by itself prove that every packet handler or gameplay feature in this fork is correct.

## Runtime boundary

### Go: network adapter only

Keep the existing Dragonfly/gophertunnel implementation responsible for:

- RakNet/NetherNet listener and transport as currently configured by this fork.
- Bedrock login, identity/JWT handling, encryption, compression and packet framing.
- Packet ID mapping, packet decoding and packet encoding.
- Sending and receiving all client-facing Bedrock packets.
- Connection lifecycle and transport-level timeouts.

Do not replace gophertunnel with MC-RS RakNet/proto/crypto or Chorus bedrock-rs/bevy-raknet/nethernet. Do not change the advertised protocol version, packet pool, login flow or wire encoding as part of the gameplay migration.

### Rust: authoritative gameplay core

Rust owns the canonical state and rules for:

- Player lifecycle, movement validation, game modes and abilities.
- Blocks, mining progress, block placement, drops and block entities.
- Inventory, item stacks, equipment, durability, use actions and crafting.
- Entities, mobs, AI, combat, health, effects and damage.
- World, chunks, dimensions, lighting, ticking and world generation.
- Containers, redstone, commands, gamerules, permissions and persistence.

There must be one authoritative copy of each player/world/inventory state. Go must not independently simulate gameplay after a subsystem has moved to Rust.

### Local adapter contract

Use a versioned, length-prefixed local IPC channel over a Unix domain socket for the first integration, with batched messages in both directions. This avoids cgo/Android ABI coupling in the initial Termux deployment and keeps local IPC entirely separate from the Bedrock wire protocol.

Inbound direction: Go decodes the existing gophertunnel packet into a versioned semantic event (connection lifecycle, player input, block action, inventory action, chat/command request, resource-pack response, etc.) and batches it to Rust.

Outbound direction: Rust returns semantic game events (state corrections, block changes, entity updates, inventory updates, sounds, level events, forms, disconnect/kick requests, etc.). Go maps these to the existing gophertunnel packet structs and sends them through the existing Dragonfly connection.

The IPC schema must have an explicit protocol version, message length limits, connection/player identifiers, tick sequence, error response and graceful shutdown. Batch high-frequency movement and chunk/block updates. Never send raw Bedrock packet bytes from Rust and never let Rust own the client socket.

## Source selection and adaptation rules

1. Prefer Chorus's modular Bevy ECS, block/entity/item/level/worldgen crates where they provide a suitable 26.60-capable foundation.
2. Port selected MC-RS gameplay logic and data where it is more complete or has useful BDS-derived behavior, especially block breaking/drops, inventory actions, survival mechanics, commands and tests.
3. Do not merge two competing implementations of the same subsystem. Select one canonical implementation and port the other project's useful tests, data or algorithms into it.
4. MC-RS modules that import `mc_rs_proto`, `mc_rs_raknet`, `mc_rs_crypto`, its packet IDs or connection/session types require adaptation. Do not pull those crates into the production runtime.
5. Chorus's `NetworkPlugin`, packet handlers, transport, NetherNet plugin and Bedrock protocol adapter are excluded from the Dragonfly-hosted runtime. Its gameplay systems must be exposed behind an embedded/core plugin that does not start listeners or accept client sockets.
6. Maintain a source manifest for every copied/adapted module: upstream repository, exact commit, original path, local path and license. Retain MIT/Apache-2.0 notices and mark modified files as required by Apache-2.0. Preserve all applicable dependency licenses.
7. Do not copy BDS proprietary source or assets. Only use independently licensed project code and lawfully redistributable data, with provenance recorded.

## Migration order

1. **Boundary and build scaffold:** create the Rust workspace, isolate the Go packet adapter, define IPC schemas and add protocol-version assertions. No gameplay ownership changes yet.
2. **Player/session vertical slice:** connect, spawn, movement, game mode, difficulty, disconnect and reconnect through Go -> Rust -> Go. Preserve UUID/XUID identity handling in Go's login layer.
3. **Block interaction vertical slice:** block state lookup, start/continue/abort/predict-break actions, crack animation events, drops, XP, tool durability and block update packets.
4. **Inventory vertical slice:** stack requests, legacy transactions, hotbar/equipment, offhand, containers, consumption, durability and persistence.
5. **World and entities:** chunk lifecycle, world generation, dimensions, entity replication, AI, combat and death/respawn.
6. **Vanilla parity:** redstone, fluids, block entities, crafting/furnaces, enchantments, effects, commands, gamerules and remaining mechanics.
7. **Performance:** batch IPC, chunk send scheduling, blob cache/delta sync and parallel simulation only after correctness is stable.
8. **Retire Go gameplay:** remove old Go game-state ownership only after feature parity tests pass; keep Dragonfly's network/protocol adapter.

## Required regression gates

- Build/test Go and Rust independently and through the integrated launcher.
- Assert the Go protocol remains 2225 / 1.26.60.
- Real client can join, respawn, reconnect and leave without protocol errors.
- Creative-to-survival mode persists after reconnect.
- Client Settings UI difficulty changes do not disconnect the client and are synchronized to all players.
- All ordinary blocks show crack progression; aborting mining clears cracks; completed mining updates the block and synchronizes drops/XP.
- Food, projectiles, stack actions and tools consume/damage correctly in survival; creative behavior remains correct.
- Player inventory, offhand and equipment survive restart.
- Movement, Elytra, potion effects, entity animation, dimensions and world persistence are checked.
- Performance comparisons record TPS, tick time, memory, chunk throughput and mobile client FPS.
- CI success is not treated as proof of real-device/client compatibility. Record code, CI and device-test status separately.

## Current status

- Source repositories and license files inspected at the audited refs above.
- gophertunnel v1.63.0 protocol constant verified as 2225 / 1.26.60.
- No MC-RS or Chorus gameplay source has yet been copied into this repository.
- No Dragonfly network/protocol code has been replaced.
- Next implementation step: introduce the Rust workspace and versioned local IPC contract, then complete the player/session vertical slice before migrating any existing gameplay ownership.
