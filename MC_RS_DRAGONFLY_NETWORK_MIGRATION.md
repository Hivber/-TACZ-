# MC-RS gameplay engine + Dragonfly network migration

## Scope and source pin

- Integration branch: `mc-rs-dragonfly-net`.
- Upstream: `https://github.com/pedrokarim/mc-rs`.
- Imported revision: `da93a9501601fcdc284762fde88c24b44604991e`.
- Upstream snapshot: `upstream/mc-rs/` (complete tracked working tree; keep this directory as an unmodified provenance snapshot).
- Upstream license: MIT; preserve its LICENSE and notices.

The upstream snapshot is imported, but the active server is not yet an MC-RS runtime. Do not describe the server as migrated or playable until the Rust engine is authoritative and client-tested.

## Network ownership boundary

Dragonfly remains the only Bedrock network endpoint:

1. UDP/RakNet and NetherNet transport remain in Dragonfly's Go stack.
2. Bedrock login, encryption, compression, packet decoding/encoding, protocol-version compatibility, and session lifecycle remain in Dragonfly/Gophertunnel.
3. Dragonfly's existing `server.Server`, `session.Session`, and `session.Conn` remain the client-facing runtime.
4. Rust receives gameplay-domain inputs only; Rust must not open a client-facing UDP/RakNet listener or run a second Bedrock login pipeline.
5. Rust returns authoritative world/player/inventory/entity outcomes. Go serializes those outcomes using Dragonfly's existing Gophertunnel packet types.

## Findings that constrain the port

- MC-RS is not a standalone gameplay library at the current pinned revision. Its `mc-rs-server` package is a large binary crate with a 4,327-line `src/main.rs`; that file directly imports `mc_rs_raknet::RakNetServer`, RakNet session events, MC-RS protocol packet IDs, and its own connection state machine.
- `mc-rs-server/Cargo.toml` directly depends on `mc-rs-raknet`, `mc-rs-proto`, and `mc-rs-crypto`. Removing these dependencies without extracting gameplay APIs would break compilation and discard logic that is currently interwoven with packet handlers.
- The current Dragonfly tree already contains `server/gamecore` and `native/game-core`. The Unix-domain-socket IPC is length-prefixed JSON, but the current Go bridge sends asynchronous batches and discards response payloads. The current event model only covers a subset of player, mining, and inventory state. It is a mirror/experimental bridge, not an authoritative engine boundary.
- Therefore, simply deleting MC-RS RakNet files or replacing a listener is not a valid network migration. The packet handlers and world state must be separated from MC-RS transport first.

## Required migration sequence

### Phase 1 — Preserve and inventory

- Keep the imported upstream snapshot intact.
- Build a source map of MC-RS gameplay modules, packet-facing handlers, world state, persistence, and transport dependencies.
- Build a Dragonfly map of packet ingress/egress, session lifecycle, player/world ownership, and Gophertunnel protocol handling.

### Phase 2 — Extract the Rust gameplay runtime

- Refactor MC-RS gameplay state and simulation behind a library API, separate from its `main` loop.
- Extract semantic commands/events from MC-RS packet handlers (movement, block use/break/place, inventory transactions, combat, entities, world ticking, dimensions, and persistence).
- Keep MC-RS's protocol/RakNet code in the immutable upstream snapshot for provenance and comparison, but do not link it into the Dragonfly runtime.
- Preserve source attribution and MIT license notices.

### Phase 3 — Replace the experimental mirror with an authoritative bridge

- Define versioned, typed IPC messages and stable IDs for players, worlds, entities, inventories, and block positions.
- Use synchronous request/response or transaction IDs for actions whose acceptance must be authoritative (mining, placement, inventory, combat, teleport, and game-mode changes).
- Use bounded asynchronous batches only for non-authoritative telemetry and coalescible updates. Never silently drop a gameplay mutation.
- Return explicit state deltas/snapshots, errors, and resynchronization requests to Go.
- Define startup, health, timeout, restart, backpressure, and shutdown behavior. A Rust-core failure must not silently cause divergent Go and Rust worlds.

### Phase 4 — Dragonfly adapter and packet serialization

- Translate Gophertunnel packet structs into semantic engine commands.
- Apply authoritative Rust results to Dragonfly's world/session and emit Gophertunnel packets.
- Keep packet IDs, compression, encryption, RakNet/NetherNet, login, and client compatibility exclusively on the Dragonfly side.
- Remove MC-RS networking from the active build only after no gameplay module depends on it.

### Phase 5 — Verification

- Rust unit tests and Go adapter tests for every command/result pair.
- Cross-language IPC contract tests, ordering/backpressure tests, reconnect and crash-recovery tests.
- CI build of the Go server and Rust engine.
- Bedrock 1.26.60 client tests for login, movement, block break/place, drops, inventory, combat, dimensions, save/reload, and reconnect.
- Do not mark a phase complete based only on compilation; distinguish CI success from in-game verification.

## Non-goals

- Do not modify `main` or `dev-settings` from this branch.
- Do not run MC-RS RakNet alongside Dragonfly's network stack.
- Do not switch the server's protocol implementation to MC-RS.
- Do not claim BDS parity or full survival readiness without test evidence.
