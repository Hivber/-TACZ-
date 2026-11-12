//! Vanilla item stack limits copied from Chorus.
//! Upstream: bedrock-crustaceans/Chorus @ c8115402a3bacf2014af2ce72fa5c35729040aa8.
//! Source: crates/item/src/stack_size.rs. Apache-2.0.
//! Gameplay rules only; no Bedrock protocol or runtime IDs.

pub fn max_stack_size(identifier: &str) -> u16 {
    let name = identifier.strip_prefix("minecraft:").unwrap_or(identifier);
    const SINGLE_SUFFIXES: &[&str] = &[
        "_sword", "_pickaxe", "_axe", "_shovel", "_hoe", "_helmet",
        "_chestplate", "_leggings", "_boots", "_horse_armor", "_boat",
        "_chest_boat", "_raft", "_chest_raft", "_minecart", "_bed",
        "_shulker_box", "_bundle", "_banner_pattern",
    ];
    const SINGLE_NAMES: &[&str] = &[
        "bow", "crossbow", "trident", "fishing_rod", "shield", "elytra",
        "shears", "flint_and_steel", "carrot_on_a_stick",
        "warped_fungus_on_a_stick", "brush", "mace", "spyglass", "saddle",
        "totem_of_undying", "potion", "splash_potion", "lingering_potion",
        "written_book", "writable_book", "enchanted_book", "filled_map",
        "bundle", "mushroom_stew", "rabbit_stew", "beetroot_soup",
        "suspicious_stew", "cake", "minecart", "shulker_box", "goat_horn",
    ];
    const SIXTEEN_NAMES: &[&str] = &[
        "bucket", "ender_pearl", "snowball", "egg", "honey_bottle", "armor_stand",
    ];
    const SIXTEEN_SUFFIXES: &[&str] = &["_sign", "_hanging_sign", "_banner"];
    if SINGLE_NAMES.contains(&name) || name.starts_with("music_disc_")
        || SINGLE_SUFFIXES.iter().any(|suffix| name.ends_with(suffix))
        || (name.ends_with("_bucket") && name != "bucket") {
        1
    } else if SIXTEEN_NAMES.contains(&name)
        || SIXTEEN_SUFFIXES.iter().any(|suffix| name.ends_with(suffix)) {
        16
    } else { 64 }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn representative_stack_limits_match_vanilla_rules() {
        for (name, expected) in [
            ("minecraft:diamond_sword", 1), ("minecraft:shield", 1),
            ("minecraft:water_bucket", 1), ("minecraft:ender_pearl", 16),
            ("minecraft:oak_sign", 16), ("minecraft:cobblestone", 64),
        ] { assert_eq!(max_stack_size(name), expected, "{name}"); }
    }
}
