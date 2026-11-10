# Vanilla gameplay parity roadmap

This file tracks server-side gameplay parity for the Bedrock 1.26.60 target.
A visible item or block identifier is not counted as implemented unless its
server-side interactions and persistence are also implemented and tested.

## Existing foundations

- Bedrock 1.26.60 protocol and item/block runtime registries.
- Helix Bedrock / PNX-style terrain generator selection.
- World persistence, dimensions, biome registry, weather, time, difficulty and game rules.
- Redstone network evaluation for existing sources, wire, torches, levers and consumers.
- Many vanilla block, container, furnace, crop, liquid, portal and recipe implementations.
- A set of Bedrock and Java-style commands documented in `JAVA_COMMANDS.md`.
- Basic entity/projectile, potion effect, shield, wind-charge and melee trident support.

## Current known gaps

### Entities and AI
- Only pigs, cows, sheep, chickens, zombies and skeletons currently have living mob implementations.
- The natural spawner currently covers only surface animals and basic night monsters in the Overworld.
- Missing: biome-specific variants and aquatic/cave/Nether/End spawn categories, complete herd tables,
  full pathfinding, breeding, taming, equipment, pickup, riding, sleep, village behaviour, raids,
  patrols, phantoms, bosses and the remaining vanilla entities.
- The first AI pass still uses simplified steering and is not yet vanilla navigation.
- Natural mob spawning applies the Bedrock 24-44 block shell at simulation distance 4,
  and the 24-128 block shell plus 96-block chunk-centre horizontal limit and n-1 chunk
  taxicab boundary at distance 6+. Natural despawning applies the matching chunk boundary.
  The current implementation remains surface-only and uses a small fixed attempt budget;
  per-chunk attempt scheduling, complete population/density caps, caves, aquatic/Nether/End,
  structure spawns and full vanilla biome spawn tables remain incomplete.
- Natural mob despawning uses 3D player distance, simulation-distance immediate limits and
  a persisted 30-second inactivity counter followed by the per-tick random despawn check.

### Redstone and automation
- Existing wire/torch/lever power evaluation is not the complete redstone circuit.
- Existing partial implementations include repeaters, comparators, observers, dispensers/droppers
  (shared ejector implementation), daylight detectors and tripwire hooks. Their complete Bedrock
  timing, update ordering and edge-case parity still requires broader integration tests.
- Missing block implementations include pistons/sticky pistons, target blocks and sculk sensors.
- Missing systems include piston movement resolution, complete dispenser item behaviours, observer
  event fidelity, comparator/container edge cases and cross-component circuit timing/locking.

### Items, combat and interaction
- Several vanilla items are identity-preserving fallbacks or have only partial use behaviour.
- Trident throwing, Loyalty return, Riptide, Channeling and Impaling-specific damage remain incomplete.
- Missing/partial areas include fishing, archaeology, bundles, riding equipment, many spawn eggs,
  mob-specific drops, food interactions, music discs and specialised projectiles.

### World and survival systems
- Nether/End natural spawning and many structure-driven mechanics remain incomplete.
- Missing/partial systems include village/raid logic, patrols, wandering traders, insomnia/phantoms,
  mob caps by category, regional difficulty, entity despawn rules, full breeding/ecology and many
  biome-specific ambient effects.

### Verification
- CI Lint, Go tests and generated-code verification are required for every batch.
- Real Bedrock 1.26.60 client verification and long-duration server testing are still required
  for protocol, AI, redstone timing, persistence and performance claims.

## Delivery order

1. Stabilise mob profiles, spawn categories, groups, despawn and basic combat.
2. Implement the missing redstone timing/automation blocks and their block entities.
3. Complete high-impact item actions and projectile/enchantment interactions.
4. Expand living entity roster and implement species-specific AI and interactions.
5. Add Nether/End, aquatic, cave, structure, raid and village systems.
6. Run cross-system regression tests and in-client compatibility tests.

Progress must be marked complete only after the gameplay behaviour, persistence and tests are present.
