# Optional Rust redstone propagation backend

Dragonfly remains Go-first. The default build uses the Go implementation and does not require Rust or cgo.
This crate implements only the side-effect-free, flat-array redstone propagation kernel. It never reads or mutates
World, Tx, blocks, entities, sessions, or plugin state.

## Linux development build

```sh
cargo test --manifest-path native/redstone-rs/Cargo.toml
cargo build --release --manifest-path native/redstone-rs/Cargo.toml
CGO_ENABLED=1 go test -tags dragonfly_rust ./server/world
```

## Android ARM64 / Termux

Install Rust, clang and the Android NDK toolchain. Build the matching Android static library first:

```sh
cargo test --manifest-path native/redstone-rs/Cargo.toml
cargo build --release --target aarch64-linux-android --manifest-path native/redstone-rs/Cargo.toml
CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC=clang go test -tags dragonfly_rust ./server/world
```

The Go build tag is opt-in. A normal `go build` continues to use pure Go. Compare both backends on the same device
and workload with the same Go benchmark (first Go, then Rust):

```sh
go test -run TestPropagateRedstoneGraph -bench BenchmarkPropagateRedstoneGraph -benchmem ./server/world
CGO_ENABLED=1 go test -tags dragonfly_rust -run TestPropagateRedstoneGraph -bench BenchmarkPropagateRedstoneGraph -benchmem ./server/world
```

Also compare the complete server workload and `/perf` results; a faster isolated kernel does not guarantee a higher
server TPS. The FFI wrapper batches an entire graph in one call and copies its flat arrays to avoid per-node crossings.
Do not enable the Rust tag in production unless it wins on the target Android device without increasing total Tick time.

## Current benchmark note

On the GitHub Actions x86_64 runner (Intel Xeon Platinum 8370C, Go 1.26.0), the 4,096-node / 8,190-edge propagation benchmark measured:

- Go: 43.6–47.0 µs/op, 106,496 B/op, 3 allocs/op.
- Rust via cgo: 68.2–75.7 µs/op, 151,552 B/op, 4 allocs/op.

This synthetic result shows the current FFI conversion overhead outweighs the Rust kernel benefit on that host. Keep Go as the production default. The Rust backend is experimental and should only be enabled if target-device benchmarks (especially Android ARM64) show a real end-to-end improvement.
