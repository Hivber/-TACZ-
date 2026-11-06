# Java-style command compatibility

These commands are implemented on top of Dragonfly's Bedrock server APIs. They are intended for Bedrock clients connecting to this server; they do not make the server a Java Edition protocol server.

## Commands

### /give
`/give <targets> <item> [count]`

Examples:
- `/give @s minecraft:diamond 16`
- `/give @a apple 8`

Items are resolved from Dragonfly's registered item registry. If an inventory is full, the remaining items are dropped beside the target. The command is operator-only.

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

## Permissions and limitations

All added commands are operator-only. Console sources are treated as trusted. Selector-based commands require a world-bound execution context.

Command blocks are not part of this command compatibility layer. This Dragonfly source currently has no command-block block implementation or command-block actor/NBT execution path. Supporting them requires a separate implementation of the Bedrock command-block block, editor interaction, persisted block-entity data, redstone/chain execution and command source permissions.
