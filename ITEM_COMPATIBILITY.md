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

- **Shield:** an off-hand shield blocks frontal melee attacks and ordinary
  projectile damage while the player is sneaking. The shield loses durability
  when it blocks a hit. Environmental damage is not blocked. This is the first
  server-side blocking implementation; mounted activation, axe disabling,
  explosion-specific exceptions, and all projectile edge cases are not yet
  implemented.
- **Trident:** melee attacks deal 9 damage and the item has 251 durability,
  losing durability on attacks. Throwing, Loyalty return, Riptide propulsion,
  Channeling lightning and Impaling-specific damage are not yet implemented.

## Remaining fallback items

The initial creative compatibility set still includes items such as fishing
rod, saddle, filled map, brush, wind charge, bundle, wooden spear, breeze rod,
trial keys, ominous trial key and ominous bottle. These remain identity-preserving
fallbacks until their required server systems are implemented. For example,
fishing requires a hook entity and loot cycle, archaeology requires brushable
blocks and timed interaction state, and bundles require server-backed storage
and item-stack UI support.

The creative catalogue is sourced from the checked-in Bedrock item data. New
vanilla identifiers can therefore remain visible before their full behaviour is
implemented, but visibility must not be confused with gameplay support.
