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


