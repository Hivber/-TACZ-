# Server performance tuning

The default configuration is tuned for a balanced Bedrock server, including mobile or lower-core hosts:

- `Players.MaximumChunkRadius = 10`: recommended view-distance cap. A radius of 10 covers about 317 chunks in a circle. The server enforces an absolute safety cap of 16, even if an older config still contains 32.
- `World.ChunkLoadWorkers = 2`: generates or loads two chunks concurrently. Set to 1 to minimise background CPU and memory use, or increase up to 8 on a desktop host with spare cores.
- `World.Generator = "helix"`: retains Bedrock 1.26.60 terrain generation. The Helix Overworld terrain cache, Nether generator and End generator support concurrent use.

Chunk radius is a view-distance limit, not a simulation-distance change. Random ticks, scheduled block updates, entities and redstone remain enabled. World generation only affects new chunks; chunks already stored in the world database are not regenerated.

## Existing configuration files

An existing `config.toml` is not rewritten automatically. The runtime will cap `MaximumChunkRadius` at 16 for safety, but set it to 10 for the recommended balanced profile. The new `ChunkLoadWorkers` key defaults to 2 when absent.

If performance still drops after these changes, record the server log's CPU/memory usage and whether the slowdown occurs while exploring new terrain or while standing in an already generated area. Those two cases have different bottlenecks.
