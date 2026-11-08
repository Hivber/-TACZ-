# Server performance tuning

The default configuration is tuned for a balanced Bedrock server, including mobile or lower-core hosts:

- `Players.MaximumChunkRadius = 10`: recommended view-distance cap. A radius of 10 covers about 317 chunks in a circle. The server enforces an absolute safety cap of 16, even if an older config still contains 32.
- `World.ChunkLoadWorkers = 2`: generates or loads two chunks concurrently. Set to 1 to minimise background CPU/memory use, or increase up to 8 on a desktop host with spare cores.
- `World.Generator = "helix"`: retains Bedrock 1.26.60 terrain generation. The Helix Overworld terrain cache, Nether generator and End generator support concurrent use.

Chunk radius is a view-distance limit, not a simulation-distance change. Random ticks, scheduled block updates, entities and redstone remain enabled. World generation only affects new chunks; chunks already stored in the world database are not regenerated.

## Existing configuration files

An existing `config.toml` is not rewritten automatically. The runtime will cap `MaximumChunkRadius` at 16 for safety, but set it to 10 for the recommended balanced profile. The new `ChunkLoadWorkers` key defaults to 2 when absent.

If performance still drops after these changes, record the server log's CPU/memory usage and whether the slowdown occurs while exploring new terrain or while standing in an already generated area. Those two cases have different bottlenecks.


## Live diagnostics

Operators can run `/perf` in-game to inspect the process and the current world:

- Process: Goroutine count, Go heap allocation/in-use memory, cumulative GC cycles and pause time, and GC CPU fraction.
- World: tick invocations per second, total ticks, latest/EMA/P95/maximum tick duration.
- Workload: loaded chunks, in-flight chunk requests, queued chunk loads, worker count, loaded entities, active loaders, simulation distance and random tick speed.

The P95 tick time is calculated from the most recent 100 tick samples. The average is an exponential moving average. Metrics are sampled on demand except for the small fixed-size tick-duration recorder. Profiling remains disabled unless explicitly enabled.

This first diagnostic pass does not yet measure OS process CPU percentage, network bytes, per-stage tick costs, or end-to-end chunk generation latency. Use Go CPU/heap profiles and repeatable load tests before selecting deeper optimisations. A high tick duration suggests world logic; a growing chunk queue suggests generation or storage cannot keep up; high heap/GC values suggest allocation pressure. These are investigation clues, not automatic proof of a specific bottleneck.


## Tick-stage profiling

The `/perf` output also shows exponential-moving-average and recent-window P95 durations for entity updates, scheduled block updates, random/block-entity ticks, neighbour updates, redstone, lightning and remaining unclassified work.

The overall tick line distinguishes the maximum among the latest 100 ticks from the lifetime maximum. A historical spike remains visible without being confused with current performance. These timings do not change simulation order or disable game mechanics.


## Queue and redstone workload diagnostics

The latest `/perf` output includes world-transaction queue wait, processed/pending neighbour updates, redstone dirty positions, compiled graph nodes/edges and the number of dirty-position candidates scanned while selecting redstone update contexts.

Neighbour updates now use two reusable buffers. Work queued while neighbour callbacks are running is still deferred to the next tick, preserving the previous processing boundary while avoiding a per-tick slice clone.


## Runtime directories

The executable creates `plugins/` and `mods/` in its working directory at startup. They are empty folders for future organisation only: the current server does not scan, import, load, or execute files from either directory.


Redstone source and consumer updates now resolve their nearest dirty context only when an update event or block-state change will actually be emitted. Unchanged power therefore avoids the nearest-context scan entirely.


## Optional Go pprof endpoint

Set `[Features] Pprof = true` and `PprofAddress = "127.0.0.1:6060"` in `config.toml` to enable Go runtime profiling. The endpoint is deliberately restricted to loopback addresses (`localhost`, `127.0.0.1` or `[::1]`); wildcard, LAN and public bind addresses are rejected. `Pprof = false` keeps profiling disabled.

Useful commands from the same device:

```sh
go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=30
go tool pprof http://127.0.0.1:6060/debug/pprof/heap
go tool pprof http://127.0.0.1:6060/debug/pprof/goroutine
go tool pprof http://127.0.0.1:6060/debug/pprof/mutex
```

Do not expose this endpoint through port forwarding or a public reverse proxy. CPU profiling runs for the requested duration and can add overhead; collect CPU, heap, block and mutex profiles separately when diagnosing a live server.


## Allocation reuse

Network chunk block-entity NBT encoding reuses bounded byte buffers. Encoded payloads are copied before a buffer is returned to the pool, so queued packets never depend on pooled memory. This reduces temporary buffer allocation without changing packet contents. Further object pooling is intentionally gated on allocation benchmarks and explicit buffer-ownership checks.


## Vanilla features and optional extensions

The server follows a Bedrock Dedicated Server default policy: implemented vanilla Bedrock gameplay, protocol behaviour and vanilla commands stay enabled by default. They are not disabled by the optional feature switches.

The [Features] section in config.toml contains only non-vanilla additions:

- JavaCompatibilityCommands = false: enables Java-only command extensions such as /item, /attribute, /bossbar, /forceload, /worldborder, /random, and the NBT /data extension.
- PerformanceDiagnostics = false: enables the extra /perf command.
- Pprof = false: enables the Go runtime profiling HTTP endpoint. When enabled, PprofAddress must be a loopback address such as 127.0.0.1:6060.

Vanilla Bedrock commands already implemented by the server, including /give, /function, /schedule, /gamerule and /daylock, are not gated by these switches. New features should be classified against the Bedrock Dedicated Server command/gameplay surface before being assigned a default.


## Per-session network section cache

Network sub-chunk and biome encodings are cached per session using mutation revisions from the underlying paletted storage. Block palette writes, direct storage writes and biome section replacements invalidate the matching encoded section. The cache is bounded to 128 entries per session and uses FIFO eviction; it is cleared on world switches. This does not alter gameplay or the Bedrock wire format.

The `/perf` output includes `encode_cache_hits` and `encode_cache_misses` when `[Features] PerformanceDiagnostics = true`. These counters help determine whether repeated sub-chunk requests are reusing encoded data.


## Inbound packet processing queue

The connection reader now places decoded packets into a bounded 128-packet queue. A single ordered worker applies them through the world owner, preserving per-session packet order. If the queue fills, the reader stops consuming the socket until capacity is available, providing backpressure rather than allowing unbounded memory growth. On disconnect, queued packets are drained before the player is closed. The `/perf` diagnostics report `inbound_queue` occupancy when enabled.
