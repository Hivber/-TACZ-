# Vanilla item coverage

## Compatibility fallback

The server's vanilla item runtime table contains more identifiers than Dragonfly
currently has dedicated gameplay implementations for. Previously, creative
catalog entries for those identifiers were silently discarded.

Known vanilla identifiers without a dedicated implementation now resolve to
an inert fallback item. This allows them to be listed in creative inventory,
given through commands, carried, moved, saved and dropped without losing their
vanilla identifier.

The fallback intentionally does not implement item use, combat, equipment,
durability, projectile, riding or container behaviour. Those behaviours must
be implemented individually before an item can be considered fully functional.
Unknown identifiers that are not present in the vanilla runtime table remain
rejected.

Initial compatibility coverage includes shield, fishing rod, saddle, filled
map, trident, brush, wind charge, bundle, wooden spear, breeze rod, trial key,
ominous trial key, ominous bottle and glow berries. The full creative catalogue
is sourced from the checked-in Bedrock item data and will automatically benefit
when a listed identifier is present in the runtime table.
