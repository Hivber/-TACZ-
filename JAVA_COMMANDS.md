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

### /function
`/function <namespace:path>`

Executes a server-side function file at `functions/<namespace>/<path>.mcfunction`.
A name without a namespace uses `minecraft`. Blank lines and lines beginning
with `#` are ignored. Nested function calls are supported with a recursion
limit of 32 levels and a combined execution limit of 10,000 commands. Function
names are validated to prevent path traversal. Files are limited to 1 MiB.

Example:
- `/function minecraft:example`

### /schedule
`/schedule function <namespace:path> <time> [append|replace]`
`/schedule clear [<namespace:path>|*]`

Schedules a function in the current world's tick loop. Time accepts ticks
(`t`, or no suffix), seconds (`s`) or days (`d`), with a maximum delay of
30 days. `append` is the default; `replace` cancels pending executions of
the same function before scheduling a new one. Scheduled tasks are in-memory
only and are discarded when the server stops.

Examples:
- `/schedule function minecraft:example 200t`
- `/schedule function minecraft:example 10s replace`
- `/schedule clear minecraft:example`
- `/schedule clear *`

### /forceload
`/forceload add <x> <z> [<toX> <toZ>]`
`/forceload remove <x> <z> [<toX> <toZ>]`
`/forceload remove all`
`/forceload query [<x> <z>]`

Keeps selected chunks loaded by holding Dragonfly chunk loaders. Coordinates
are block coordinates and negative values use floor-based chunk conversion.
A single operation is limited to 256 chunks and a world is limited to 2,048
forced chunks. Forced chunks are currently runtime-only and are released when
the server stops; they are not persisted into level.dat.

### /worldborder
`/worldborder get`
`/worldborder set <diameter> [seconds]`
`/worldborder add <distance> [seconds]`
`/worldborder center <x> <z>`

Implements a per-world square horizontal movement boundary. The server clips
normal player movement at the border while still allowing players outside it
to move back in. A size transition interpolates over the optional duration.
Teleports are not clipped. Bedrock does not expose Java's world-border packet,
so this implementation does not draw the Java border overlay or implement
damage and warning settings. Border state is currently in-memory and is reset
when the server restarts.

### /data (read-only subset)
`/data get block <x> <y> <z> [path] [scale]`
`/data get entity <targets> [path] [scale]`

Reads NBT from blocks that implement Dragonfly's NBT interface (such as chests
and supported signs), or from selected entities through their registered
entity-type encoder. Optional paths support compound keys and list indices,
for example `Items[0].id`. An optional numeric scale is applied to a numeric
result. NBT mutation (`merge`, `modify`, `remove`) and Java SNBT parsing are
not implemented.

### /random

`/random value <range>`
`/random roll <range>`

Generates an inclusive signed 32-bit integer range. A single integer selects that exact value; ranges may use `min..max`, `..max`, or `min..`. `value` reports only to the command caller; `roll` also announces the result to players in the same world.

Examples:
- `/random value 1..6`
- `/random roll 1..100`
- `/random value ..0`

Named random sequences and `/random reset` are not implemented yet; those require a persistent per-world sequence store.

## Permissions and limitations

All commands in this compatibility list require operator permissions except `/random value <range>` and `/random roll <range>` without a named sequence, which are available to all players as in Java Edition. The executable grants operator status to the configured `OwnerXUID` or UUIDs in `operators.json`; the local server console can also use the operator management commands. If `/give` or another admin command is missing from the in-game command list, verify that the account has operator status and reconnect after granting it.

Selectors such as `@a`, `@p`, `@e`, `@s`, and `@r` require a world-bound execution context. The console command source is trusted for permission checks, but commands that need a player target or an entity transaction cannot be run from a world-less console source.

Command blocks are implemented as native Bedrock blocks with saved block-actor data and the Bedrock CommandBlockUpdate editing packet. They support impulse, repeating and chain modes, conditional chains, redstone activation, always-active behaviour, tick delay, first-tick execution and command output tracking. Only operators within interaction range can edit them. Commands execute using the server's registered command table, and the maximum chain length is capped at 65,536 even if a world rule is set higher.

Supported commands are those registered by this Dragonfly server. Commands that require a player target or player-only context cannot treat the command block itself as a player. Command block minecarts and Java-specific entity selectors/NBT syntax are not implemented. The `commandblocksenabled` and `commandblockoutput` game rules control execution and output tracking.


## Not yet implemented

The following Java command capabilities are deliberately not registered as
no-op placeholders:

- `/team`: Dragonfly's current scoreboard API does not provide team membership,
  collision rules, friendly-fire settings, or the related team state model.
- `/data merge`, `/data modify`, and `/data remove`: the current `/data`
  implementation is read-only and does not parse Java SNBT.
- `/random sequence` and `/random reset`: named deterministic sequences
  require per-world sequence state and persistence.
- `/worldborder damage` and `/worldborder warning`: the current border
  enforces movement bounds but has no damage/warning subsystem or Bedrock UI
  equivalent to Java's border overlay.
