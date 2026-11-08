# Server performance tuning

The default configuration is tuned for a phone that may host the server and run the Bedrock client at the same time:

- `Players.MaximumChunkRadius = 8`: mobile-friendly view-distance cap. A radius of 8 covers about 197 chunks in a circle, about 38% fewer than radius 10. The server enforces an absolute safety cap of 16, even if an older config still contains 32.
- `World.ChunkLoadWorkers = 1`: avoids running multiple terrain-generation workers alongside the mobile game client. On a separate host with spare CPU, set this to 2 or increase up to 8.
- `World.Generator = "helix"`: retains Bedrock 1.26.60 terrain generation. The Helix Overworld terrain cache, Nether generator and End generator support concurrent use.

Chunk radius is a view-distance limit, not a simulation-distance change. Random ticks, scheduled block updates, entities and redstone remain enabled. World generation only affects new chunks; chunks already stored in the world database are not regenerated.

## Existing configuration files

An existing `config.toml` is not rewritten automatically. The runtime will cap `MaximumChunkRadius` at 16 for safety, but set it to 8 for the recommended mobile-host profile. The new `ChunkLoadWorkers` key defaults to 1 when absent.

If performance still drops after these changes, record the server log's CPU/memory usage and whether the slowdown occurs while exploring new terrain or while standing in an already generated area. Those two cases have different bottlenecks.


## Live diagnostics

Operators can run `/perf` in-game to inspect the process and the current world:

- Process: Goroutine count, Go heap allocation/in-use memory, cumulative GC cycles and pause time, and GC CPU fraction.
- World: tick invocations per second, total ticks, latest/EMA/P95/maximum tick duration.
- Workload: loaded chunks, in-flight chunk requests, queued chunk loads, worker count, loaded entities, active loaders, simulation distance and random tick speed.

The P95 tick time is calculated from the most recent 100 tick samples. The average is an exponential moving average. Metrics are sampled on demand except for the small fixed-size tick-duration recorder; no profiling server or public network endpoint is opened.

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
