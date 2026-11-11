# Dragonfly Rust gameplay core (initial vertical slice)

This is the first executable Rust gameplay-core module. It is deliberately separate from the Go Bedrock transport:

- Go/gophertunnel remains responsible for RakNet/NetherNet, login/JWT, encryption, compression, packet IDs and packet serialization.
- Rust receives versioned semantic events over a local Unix-domain socket. It never accepts a client connection or decodes Bedrock packets.
- The current slice contains player lifecycle state and a port of MC-RS block-mining progress calculations, including tool compatibility, airborne/underwater penalties, Haste and Mining Fatigue.

## Build and test

```sh
cargo test --manifest-path native/game-core/Cargo.toml
cargo run --release --manifest-path native/game-core/Cargo.toml -- /tmp/dragonfly-game-core.sock
```

The IPC protocol uses a four-byte big-endian length followed by one JSON request/response. Maximum frame size is 1 MiB. Protocol version is explicit and mismatches are rejected. The Go adapter in `server/gamecore` uses the same framing.

## Source attribution

- MC-RS, MIT: mining compatibility and break-progress formula adapted from `crates/mc-rs-server/src/block_break.rs`, audited revision `da93a9501601fcdc284762fde88c24b44604991e`.
- Chorus, Apache-2.0: architecture review only in this slice; Bevy ECS game systems have not yet been vendored or enabled.
- MC-RS's own RakNet, crypto, protocol, connection and packet runtime are intentionally excluded.

This initial slice is not yet wired as the authoritative runtime for live players. The next integration step is to emit join/move/leave semantic events from Dragonfly's existing session lifecycle and then move block-breaking decisions behind the Rust core only after packet/event parity tests pass.
