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

## Implementation status update (2026-11-12)

The following code is now present on dev-settings:

- A Chorus-derived, data-only block component model covers mineable hardness,
  item-specific hardness, loot, collision boxes, flammability, friction,
  internal friction, light dampening/emission, map color, piston movement,
  replaceability, solidity and transparency.
- Block definitions can provide base components and ordered state predicates.
  Matching permutation overrides merge into the active permutation, replacing
  components of the same kind; repeated override combinations share storage.
- The MC-RS-derived mining slice now derives break profiles from active
  permutation components and includes Efficiency, correct-tool/harvest checks,
  Haste, Mining Fatigue, airborne/water penalties and Unbreaking/tool wear.
- The Rust core includes deterministic weighted block-drop resolution,
  silk-touch overrides, configurable Fortune bonus ranges and the vanilla
  fallback rule for blocks without an explicit loot component.
- CI now formats, tests and release-builds native/game-core. A CI pass validates
  this crate on the CI host only; it does not prove Termux compilation or
  1.26.60 client behavior.

Version boundary: MC-RS's block_break.json and block_drops_bds.json sampling is
for Bedrock 1.26.52.3. It has not been copied into the 1.26.60 runtime. The
generic resolver requires a validated 1.26.60 block/loot data source before it
can be considered exhaustive vanilla drop parity.

Still not migrated: live Dragonfly session/packet event hooks, a persistent
bidirectional IPC connection with batching/backpressure, player movement
authority, block action handling and world mutation, inventory/containers,
entities/AI/combat, chunks/world ticking, persistence, and the remaining
vanilla systems. Old Go gameplay paths remain authoritative for live sessions;
no Go gameplay code was removed. Go still owns the complete 1.26.60 network and
protocol path (gophertunnel protocol 2225).


## Block-interaction state machine (2026-11-12)

The Rust core now includes a deterministic block-mining state machine that:
- validates Bedrock block face values and caller-provided interaction reach;
- starts one block-break session per player and binds it to the target position;
- increments mining progress at most once for each server tick, preventing
  duplicate client continuation actions from accelerating the break;
- derives crack stages from authoritative progress and validates the block's
  current break profile on each step;
- aborts on target changes, reach violations, invalid actions or unbreakable
  blocks, and reports semantic completion for the Go adapter to translate into
  normal Dragonfly/gophertunnel effects and block/item packets.

This state machine is unit-tested but is not yet called from Dragonfly's live
packet/session handler. It does not send packets itself and therefore does not
change the Bedrock 1.26.60 network boundary.


## Persistent shadow bridge integration (2026-11-12)

The Go/Rust local IPC is persistent and batched. Go reuses one Unix-domain socket connection, applies frame and batch limits, validates protocol versions, and reconnects after transport failures. The Rust daemon accepts multiple frames per connection, handles clients concurrently, serializes mutations against shared state, validates finite player positions, limits batches, sets socket permissions to 0600, and refuses to unlink a pre-existing non-socket path.

An optional `GameCoreSocket` setting enables a bounded asynchronous Go bridge. Session spawn, accepted movement snapshots and disconnect enqueue semantic player lifecycle events. Events are batched (up to 256 / 10 ms) and flushed at shutdown. Packet/session owners never wait for Rust; queue saturation or daemon failure is logged and does not affect the existing Go gameplay path.

Player inventory mirroring now snapshots the main inventory, off-hand, ender chest and armour slots on spawn and after slot mutations. The fixed slot order is main inventory, off-hand, ender chest, then armour. Empty slots retain their indices. Each non-empty stack carries its item identifier, count, encoded metadata and a versioned `dragonfly-item-nbt-json-v1` payload generated from Dragonfly's disk item codec. This preserves display name/lore, durability, enchantments, repair cost, unbreakable state, item-specific NBT and Dragonfly custom values in the shadow representation. Payloads above the IPC item-data limit or invalid stack counts cause that snapshot to be skipped rather than sending a partial inventory. The Rust core currently stores this payload as opaque bytes; it does not decode it into authoritative item components or apply inventory actions.

**This is shadow-state mirroring, not gameplay authority.** Dragonfly still validates movement and executes block actions, inventory, entities and world ticks in Go. Rust does not yet return authoritative corrections or outbound gameplay effects. The optional bridge is disabled unless configured.


## Scoped inventory/container shadow mirrors (2026-11-12)

The inventory IPC schema uses a stable `inventory_id` instead of treating a
player UUID as the identity of every inventory. Player-carried inventory uses
`player:<uuid>:carried`; block-owned inventories use
`world-container:<dimension-id>:<x>:<y>:<z>`. Bedrock window IDs are
transient transport/UI identifiers and are never used as persistent inventory
keys. Player disconnect removes only that player's carried mirror; world
container mirrors are retained.

When a normal Dragonfly block container is opened, the Go session emits a full
slot-preserving snapshot. Subsequent block-container viewer slot notifications
emit fresh snapshots, including changes made by other viewers. This covers
containers implemented through Dragonfly's `block.Container` interface,
including paired chests and supported block inventories such as barrels,
shulker boxes, hoppers, furnaces, smokers, blast furnaces and brewing stands.
The mirror is still optional and asynchronous.

Ender chests remain part of the player's carried inventory mirror. Crafting,
anvil, smithing, loom, stonecutter, grindstone, enchanting and beacon UI
inventories are transient UI state and are intentionally not mislabeled as
persistent block containers. The current key is position-based; if a block
container is replaced at the same dimension/position, its mirror key is reused
and the next complete snapshot replaces the previous slots.

This is still a **shadow mirror**. Go's inventory transaction handlers,
slot validators, item-stack request logic, crafting/furnace behavior and
persistence remain authoritative. Rust inventory operation results are not
applied to Dragonfly or returned as client packets. No Rust-driven inventory
transaction authority or restart persistence is claimed.


## Cross-inventory transfer primitive (2026-11-12)

The Rust inventory core now exposes an exact-count transfer between two distinct
inventory mirrors, including player-carried and world-container IDs. It validates
source count, destination slot, stack identity, metadata/user-data identity and
maximum stack capacity before mutating either side. Failed transfers leave both
inventories unchanged. The versioned IPC schema carries source/destination
inventory IDs and slots and returns the updated source snapshot. Same-inventory
moves reuse the existing move operation.

This is a tested core primitive and IPC event contract, not yet a live Bedrock
transaction authority. The Dragonfly ItemStackRequest and legacy transaction
handlers still validate and apply client actions on the Go world owner; no Rust
response is yet translated into live inventory mutations or packet responses.


## Atomic paired inventory shadow synchronization (2026-11-13)

Accepted Dragonfly ItemStackRequest transfers and swaps now enqueue a final
paired snapshot for the two affected mirrored inventories. The Go adapter
continues to validate stack network IDs, run inventory event hooks, and apply
the actual slot mutations; Dragonfly remains the live authority. Per-slot
callbacks can enqueue intermediate snapshots during a multi-slot operation,
so the final pair event replaces both Rust mirrors atomically after the
transaction. The pair validates both inventory IDs, slot counts, and every
item payload before replacing either mirror. Transient crafting/UI inventories
are deliberately excluded.

This closes an intermediate-state consistency gap in the Rust shadow mirror.
It does **not** move ItemStackRequest validation, event hooks, transaction
acceptance, item drops, crafting, or inventory ownership into Rust.


## Additional MC-RS gameplay modules (2026-11-13)

Two transport-independent MC-RS gameplay subsystems have now been copied and
adapted into this crate:

- `game_mode.rs`: Survival/Creative/Adventure/Spectator identifiers and
  capability predicates, with safe fallback for unknown identifiers.
  Adventure block permissions remain metadata-dependent; this module does not
  replace item CanDestroy/CanPlaceOn checks.
- `entity_visibility.rs`: horizontal view-distance culling, per-viewer
  visibility transition classification, independent visible-entity sets, and
  cleanup on entity/viewer removal. Invalid coordinates remove stale visibility
  rather than leaving an entity permanently visible.

Both modules are pinned to MC-RS revision
`da93a9501601fcdc284762fde88c24b44604991e`, with original paths and MIT
attribution recorded in source headers. The entity visibility module does not
send packets and is not yet wired to Dragonfly's live entity replication.
The game-mode capability model is not yet the authoritative source for live
player game-mode or permission changes. These are reusable imported gameplay
subsystems, not a claim that the whole MC-RS server has been transplanted.
