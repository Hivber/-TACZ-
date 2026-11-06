# Java-style command compatibility

These commands run through Dragonfly's Bedrock server APIs. They are intended for Bedrock clients connecting to this server; they do not make the server a Java Edition protocol server. All compatibility commands below require operator permissions unless explicitly noted.

## Commands

### /give
`/give <targets> <item> [count]`

Examples:
- `/give @s minecraft:diamond 16`
- `/give @a apple 8`

Items are resolved from Dragonfly's registered item registry. If an inventory is full, remaining items are dropped beside the target. The command is operator-only.

### /clear
`/clear`
`/clear <targets> [item] [maxCount]`

The no-argument form clears the caller's normal inventory and hotbar. The target form optionally filters by a registered item identifier and caps the total number removed per target; a maxCount of 0 means unlimited. Ender chests and armour slots are not included.

Examples:
- `/clear`
- `/clear @s`
- `/clear @a minecraft:dirt 32`

### /effect
`/effect give <targets> <effect> [seconds] [amplifier] [hideParticles]`
`/effect clear <targets> [effect]`

Supports the effects implemented by Dragonfly's effect registry, including speed, slowness, haste, strength, regeneration, resistance, invisibility, poison, wither, slow falling and darkness. Effect names may be entered with or without the `minecraft:` namespace. Duration is limited to 1–604800 seconds; amplifier is limited to 0–30. Instant effects apply immediately; duration does not change their instant behaviour.

Examples:
- `/effect give @s speed 120 1`
- `/effect give @a regeneration 30 0 true`
- `/effect clear @s`
- `/effect clear @a poison`

### /enchant
`/enchant <targets> <enchantment> [level]`

Applies an implemented enchantment to the target's main-hand item. The enchantment registry checks item compatibility and the enchantment's supported maximum level. It does not implement Java's full item-component syntax or enchantments that Dragonfly itself has not implemented.

Examples:
- `/enchant @s sharpness 5`
- `/enchant @p unbreaking 3`

### /xp (alias /experience)
`/xp add <targets> <amount> [points|levels]`
`/xp remove <targets> <amount> [points|levels]`
`/xp query <targets>`

The default unit is `points`. Each add/remove operation is capped at 1,000,000,000; the level-setting path is capped at 100,000,000 to stay within the safe range of the current experience manager's integer calculations.

Examples:
- `/xp add @s 100 points`
- `/xp add @a 5 levels`
- `/xp remove @s 20`
- `/xp query @s`

### /item
`/item <targets> <slot> <item> [count]`

Supported slots:
- `weapon.mainhand`
- `weapon.offhand`
- `armor.head`, `armor.chest`, `armor.legs`, `armor.feet`
- `inventory.0` through `inventory.35`

Examples:
- `/item @s weapon.mainhand minecraft:diamond_sword`
- `/item @p armor.head minecraft:diamond_helmet`

This is a practical slot-replacement subset; Java's complete item modifier/component syntax is not implemented.

### /attribute
`/attribute <targets> <attribute> [value]`

Supported player attributes:
- `minecraft:movement_speed`
- `minecraft:flying_speed`
- `minecraft:max_health`
- `minecraft:scale`

Without a value, the command queries the current value. With a value, it changes the corresponding native Dragonfly player setting. Modifiers, named attribute modifiers, and non-player entity attributes are not implemented.

### /bossbar
`/bossbar <targets> <progress> <colour> <text...>`

Progress is between 0 and 1. Colours: `pink`, `blue`, `red`, `green`, `yellow`, `purple`, `white`.

Example:
- `/bossbar @a 0.65 red The Wither is approaching`

Dragonfly currently exposes one active boss bar per player, so this command displays or replaces that bar; it does not implement Java's named multi-bossbar registry.

### /bossbarclear
`/bossbarclear <targets>`

Hides the current boss bar for the selected players.

### /summon
`/summon <entity> [x y z]`

The command spawns entities that this Dragonfly build actually implements:
- `tnt` (primed TNT with a four-second fuse)
- `lightning_bolt`
- `ender_crystal`
- `xp_orb` / `experience_orb` (one experience point)

If coordinates are omitted, the entity spawns at the command source's position. Ordinary mobs such as zombies, skeletons, cows and villagers are not currently implemented by this server, so they cannot be summoned yet.

Examples:
- `/summon lightning_bolt`
- `/summon tnt 120 64 -30`
- `/summon minecraft:ender_crystal`

## Permissions and limitations

All commands in this compatibility list are operator-only. The executable grants operator status to the configured `OwnerXUID` or UUIDs in `operators.json`; the local server console can also use the operator management commands. If `/give` or another admin command is missing from the in-game command list, verify that the account has operator status and reconnect after granting it.

Selectors such as `@a`, `@p`, `@e`, `@s`, and `@r` require a world-bound execution context. The console command source is trusted for permission checks, but commands that need a player target or an entity transaction cannot be run from a world-less console source.

Command blocks are not part of this command compatibility layer. This Dragonfly source still needs a separate command-block implementation covering the block and actor data, editing UI, persistence, redstone triggering, impulse/repeat/chain modes, conditional execution, delays and safe command-source permissions.
