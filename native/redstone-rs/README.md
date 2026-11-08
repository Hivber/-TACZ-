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
go test -run '^
 -bench 'BenchmarkPropagateRedstoneGraph|BenchmarkRedstoneDirtyContext' -benchmem ./server/world
CGO_ENABLED=1 go test -tags dragonfly_rust -run '^
 -bench 'BenchmarkPropagateRedstoneGraph|BenchmarkRedstoneDirtyContext' -benchmem ./server/world
```

Also compare the complete server workload and `/perf` results; a faster isolated kernel does not guarantee a higher
server TPS. The FFI wrapper batches an entire graph in one call and copies its flat arrays to avoid per-node crossings.
Do not enable the Rust tag in production unless it wins on the target Android device without increasing total Tick time.
