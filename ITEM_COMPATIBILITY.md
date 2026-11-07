# Vanilla item coverage

## Compatibility fallback

The vanilla item runtime table contains more identifiers than Dragonfly currently
has dedicated gameplay implementations for. Known identifiers without a
dedicated implementation resolve to an inert fallback so that their identity is
preserved in creative inventory, commands, inventories, persistence and network
packets.

A fallback item is deliberately not considered a completed gameplay feature.
It does not automatically implement use, combat, equipment, durability,
projectile, riding or container behaviour.

## Dedicated gameplay implementations added

- **Shield:** a shield in either hand raises after five crouching ticks,
  synchronises Bedrock blocking, transition and blocked-hit metadata, blocks
  frontal melee and projectile damage, and knocks back melee attackers. Hits
  below 3 damage do not consume durability; larger hits consume
  `int(damage)+1` durability, following the current PowerNukkitX reference.
  Axe disabling, projectile deflection, banner patterns, mounted activation
  and additional explosion/projectile exceptions remain unimplemented.
- **Trident:** melee attacks deal 9 damage and the item has 251 durability,
  losing durability on attacks. Throwing, Loyalty return, Riptide propulsion,
  Channeling lightning and Impaling-specific damage are not yet implemented.
- **Wind charge:** can be thrown as a server-side projectile, deals 1 point of
  direct hit damage, and creates a 2-block wind burst. The burst halves the
  target's existing velocity, adds a radial impulse and 0.6 upward velocity,
  then resets fall distance. It can toggle registered doors, trapdoors, fence
  gates and levers, ignite candles, and break decorated pots through Dragonfly's
  projectile-hit interface. It has a 64-item stack limit and a 10-tick cooldown.
  Buttons, bells, chorus flowers, projectile deflection and liquid attenuation
  remain unavailable or unimplemented in this branch.

## Remaining fallback items

The initial creative compatibility set still includes items such as fishing
rod, saddle, filled map, brush, bundle, wooden spear, breeze rod,
trial keys, ominous trial key and ominous bottle. These remain identity-preserving
fallbacks until their required server systems are implemented. For example,
fishing requires a hook entity and loot cycle, archaeology requires brushable
blocks and timed interaction state, and bundles require server-backed storage
and item-stack UI support.

The creative catalogue is sourced from the checked-in Bedrock item data. New
vanilla identifiers can therefore remain visible before their full behaviour is
implemented, but visibility must not be confused with gameplay support.




## Initial mob runtime and spawning

The first living-entity set is registered as real server entities: pigs, cows,
sheep, chickens, zombies and skeletons. They have health, damage handling,
basic movement and wandering, hostile melee targeting, death and basic item
drops.

The Overworld performs bounded natural-spawn attempts every 200 world ticks.
The first pass respects `doMobSpawning`, world difficulty, surface light,
loaded chunks and per-player mob caps. It currently spawns common animals on
grass in bright conditions and zombies/skeletons in sufficiently dark night
conditions.

Dedicated spawn eggs are enabled for those six mob types. Using one on a block
spawns the corresponding entity and consumes one egg. Thrown ordinary eggs
hatch chickens with the vanilla 1/8 chance; on a successful hatch, there is a
1/32 chance to spawn four chicks.

This is not full vanilla mob parity yet. Biome-specific spawn tables, group
weights, despawning, breeding, taming, full pathfinding, ranged skeleton AI,
creeper behaviour, Nether/End spawns and the remaining vanilla mob types
remain future work.


## Mob natural-spawning improvements

Natural spawning now uses profile-based category, weight, group bounds, biome tags
and light ranges instead of selecting a random mob from one hard-coded list.
The first profiles cover surface animals and basic night monsters. Spawns occur
only in already-loaded chunks, outside the 24-block player exclusion radius,
and obey category caps near active players.

Zombies and skeletons now acquire targets at species-appropriate ranges;
skeletons fire server-side arrows at medium range. These two hostile species
burn in direct daylight, can be extinguished by rain or liquid, and receive
difficulty-scaled melee damage. Naturally spawned mobs have hard/soft distance
despawn checks; custom-named mobs are retained.

This remains a deliberately limited first AI/spawn pass: the profiles do not
yet model the full Bedrock spawn tables, all mob categories, navigation,
equipment, breeding, taming, or species-specific survival behaviours.
