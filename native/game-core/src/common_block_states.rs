//! Shared vanilla block-state catalog adapted from Chorus state/common.rs.
//! Upstream: https://github.com/bedrock-crustaceans/Chorus
//! Revision: 5808efd55dc51f38e7900ad2c72f14d73434b093; Apache-2.0.
//! Bevy/atomicow static definitions are converted into the local owned state model.

use crate::block_state::BlockStateDefinition;

/// Returns the full upstream common-state catalog, indexed by upstream constant name.
pub fn all() -> Vec<(&'static str, BlockStateDefinition)> {
    vec![
        ("ACTIVE", BlockStateDefinition::boolean("active", false)),
        ("AGE_16", BlockStateDefinition::integer("age", 0, 15).expect("valid Chorus common state")),
        ("AGE_3", BlockStateDefinition::integer("age", 0, 2).expect("valid Chorus common state")),
        ("AGE_4", BlockStateDefinition::integer("age", 0, 3).expect("valid Chorus common state")),
        ("AGE_6", BlockStateDefinition::integer("age", 0, 5).expect("valid Chorus common state")),
        ("AGE_BIT", BlockStateDefinition::boolean("age_bit", false)),
        ("ALLOW_UNDERWATER_BIT", BlockStateDefinition::boolean("allow_underwater_bit", false)),
        ("ATTACHED_BIT", BlockStateDefinition::boolean("attached_bit", false)),
        ("ATTACHMENT", BlockStateDefinition::enumeration("attachment", ["hanging", "multiple", "side", "standing"]).expect("valid Chorus common state")),
        ("BAMBOO_LEAF_SIZE", BlockStateDefinition::enumeration("bamboo_leaf_size", ["no_leaves", "small_leaves", "large_leaves"]).expect("valid Chorus common state")),
        ("BAMBOO_STALK_THICKNESS", BlockStateDefinition::enumeration("bamboo_stalk_thickness", ["thick", "thin"]).expect("valid Chorus common state")),
        ("BIG_DRIPLEAF_HEAD", BlockStateDefinition::boolean("big_dripleaf_head", false)),
        ("BIG_DRIPLEAF_TILT", BlockStateDefinition::enumeration("big_dripleaf_tilt", ["full_tilt", "none", "partial_tilt", "unstable"]).expect("valid Chorus common state")),
        ("BITE_COUNTER", BlockStateDefinition::integer("bite_counter", 0, 6).expect("valid Chorus common state")),
        ("BLOCK_LIGHT_LEVEL", BlockStateDefinition::integer("block_light_level", 0, 15).expect("valid Chorus common state")),
        ("BLOOM", BlockStateDefinition::boolean("bloom", false)),
        ("BOOKS_STORED", BlockStateDefinition::integer("books_stored", 0, 63).expect("valid Chorus common state")),
        ("BREWING_STAND_SLOT_A_BIT", BlockStateDefinition::boolean("brewing_stand_slot_a_bit", false)),
        ("BREWING_STAND_SLOT_B_BIT", BlockStateDefinition::boolean("brewing_stand_slot_b_bit", false)),
        ("BREWING_STAND_SLOT_C_BIT", BlockStateDefinition::boolean("brewing_stand_slot_c_bit", false)),
        ("BRUSHED_PROGRESS", BlockStateDefinition::integer("brushed_progress", 0, 3).expect("valid Chorus common state")),
        ("BUTTON_PRESSED_BIT", BlockStateDefinition::boolean("button_pressed_bit", false)),
        ("CANDLES", BlockStateDefinition::integer("candles", 0, 3).expect("valid Chorus common state")),
        ("CAN_SUMMON", BlockStateDefinition::boolean("can_summon", false)),
        ("CAULDRON_LIQUID", BlockStateDefinition::enumeration("cauldron_liquid", ["water", "lava", "powder_snow"]).expect("valid Chorus common state")),
        ("CHEMISTRY_TABLE_TYPE", BlockStateDefinition::enumeration("chemistry_table_type", ["compound_creator", "element_constructor", "lab_table", "material_reducer"]).expect("valid Chorus common state")),
        ("CHISEL_TYPE", BlockStateDefinition::enumeration("chisel_type", ["default", "chiseled", "lines", "smooth"]).expect("valid Chorus common state")),
        ("CLUSTER_COUNT", BlockStateDefinition::integer("cluster_count", 0, 3).expect("valid Chorus common state")),
        ("COLOR", BlockStateDefinition::enumeration("color", ["black", "blue", "brown", "cyan", "gray", "green", "light_blue", "lime", "magenta", "orange", "pink", "purple", "red", "silver", "white", "yellow"]).expect("valid Chorus common state")),
        ("COLOR_BIT", BlockStateDefinition::boolean("color_bit", false)),
        ("COMPOSTER_FILL_LEVEL", BlockStateDefinition::integer("composter_fill_level", 0, 8).expect("valid Chorus common state")),
        ("CONDITIONAL_BIT", BlockStateDefinition::boolean("conditional_bit", false)),
        ("CORAL_COLOR", BlockStateDefinition::enumeration("coral_color", ["blue", "pink", "purple", "red", "yellow"]).expect("valid Chorus common state")),
        ("CORAL_DIRECTION", BlockStateDefinition::integer("coral_direction", 0, 3).expect("valid Chorus common state")),
        ("CORAL_FAN_DIRECTION", BlockStateDefinition::integer("coral_fan_direction", 0, 1).expect("valid Chorus common state")),
        ("CORAL_HANG_TYPE_BIT", BlockStateDefinition::boolean("coral_hang_type_bit", false)),
        ("COVERED_BIT", BlockStateDefinition::boolean("covered_bit", false)),
        ("CRACKED_STATE", BlockStateDefinition::enumeration("cracked_state", ["no_cracks", "cracked", "max_cracked"]).expect("valid Chorus common state")),
        ("CRAFTING", BlockStateDefinition::boolean("crafting", false)),
        ("CREAKING_HEART_STATE", BlockStateDefinition::enumeration("creaking_heart_state", ["uprooted", "dormant", "awake"]).expect("valid Chorus common state")),
        ("DAMAGE", BlockStateDefinition::enumeration("damage", ["undamaged", "slightly_damaged", "very_damaged", "broken"]).expect("valid Chorus common state")),
        ("DEAD_BIT", BlockStateDefinition::boolean("dead_bit", false)),
        ("DEPRECATED", BlockStateDefinition::integer("deprecated", 0, 3).expect("valid Chorus common state")),
        ("DIRECTION", BlockStateDefinition::integer("direction", 0, 3).expect("valid Chorus common state")),
        ("DIRT_TYPE", BlockStateDefinition::enumeration("dirt_type", ["normal", "coarse"]).expect("valid Chorus common state")),
        ("DISARMED_BIT", BlockStateDefinition::boolean("disarmed_bit", false)),
        ("DOOR_HINGE_BIT", BlockStateDefinition::boolean("door_hinge_bit", false)),
        ("DOUBLE_PLANT_TYPE", BlockStateDefinition::enumeration("double_plant_type", ["sunflower", "syringa", "grass", "fern", "rose", "paeonia", "pitcher_plant"]).expect("valid Chorus common state")),
        ("DRAG_DOWN", BlockStateDefinition::boolean("drag_down", false)),
        ("DRIPSTONE_THICKNESS", BlockStateDefinition::enumeration("dripstone_thickness", ["base", "frustum", "merge", "middle", "tip"]).expect("valid Chorus common state")),
        ("END_PORTAL_EYE_BIT", BlockStateDefinition::boolean("end_portal_eye_bit", false)),
        ("EXPLODE_BIT", BlockStateDefinition::boolean("explode_bit", false)),
        ("EXTINGUISHED", BlockStateDefinition::boolean("extinguished", false)),
        ("FACING_DIRECTION", BlockStateDefinition::integer("facing_direction", 0, 5).expect("valid Chorus common state")),
        ("FILL_LEVEL", BlockStateDefinition::integer("fill_level", 0, 6).expect("valid Chorus common state")),
        ("FLOWER_TYPE", BlockStateDefinition::enumeration("flower_type", ["poppy", "orchid", "allium", "houstonia", "tulip_red", "tulip_orange", "tulip_white", "tulip_pink", "oxeye", "cornflower", "lily_of_the_valley"]).expect("valid Chorus common state")),
        ("GROUND_SIGN_DIRECTION", BlockStateDefinition::integer("ground_sign_direction", 0, 15).expect("valid Chorus common state")),
        ("GROWING_PLANT_AGE", BlockStateDefinition::integer("growing_plant_age", 0, 25).expect("valid Chorus common state")),
        ("GROWTH_2", BlockStateDefinition::integer("growth", 0, 1).expect("valid Chorus common state")),
        ("GROWTH_8", BlockStateDefinition::integer("growth", 0, 7).expect("valid Chorus common state")),
        ("HANGING", BlockStateDefinition::boolean("hanging", false)),
        ("HEAD_PIECE_BIT", BlockStateDefinition::boolean("head_piece_bit", false)),
        ("HEIGHT", BlockStateDefinition::integer("height", 0, 7).expect("valid Chorus common state")),
        ("HONEY_LEVEL", BlockStateDefinition::integer("honey_level", 0, 5).expect("valid Chorus common state")),
        ("HUGE_MUSHROOM_BITS", BlockStateDefinition::integer("huge_mushroom_bits", 0, 15).expect("valid Chorus common state")),
        ("INFINIBURN_BIT", BlockStateDefinition::boolean("infiniburn_bit", false)),
        ("IN_WALL_BIT", BlockStateDefinition::boolean("in_wall_bit", false)),
        ("ITEM_FRAME_MAP_BIT", BlockStateDefinition::boolean("item_frame_map_bit", false)),
        ("ITEM_FRAME_PHOTO_BIT", BlockStateDefinition::boolean("item_frame_photo_bit", false)),
        ("KELP_AGE", BlockStateDefinition::integer("kelp_age", 0, 25).expect("valid Chorus common state")),
        ("LEVER_DIRECTION", BlockStateDefinition::enumeration("lever_direction", ["down_east_west", "down_north_south", "east", "north", "south", "up_east_west", "up_north_south", "west"]).expect("valid Chorus common state")),
        ("LIQUID_DEPTH", BlockStateDefinition::integer("liquid_depth", 0, 15).expect("valid Chorus common state")),
        ("LIT", BlockStateDefinition::boolean("lit", false)),
        ("MINECRAFT_BLOCK_FACE", BlockStateDefinition::enumeration("minecraft:block_face", ["down", "up", "north", "south", "west", "east"]).expect("valid Chorus common state")),
        ("MINECRAFT_CARDINAL_DIRECTION", BlockStateDefinition::enumeration("minecraft:cardinal_direction", ["south", "west", "north", "east"]).expect("valid Chorus common state")),
        ("MINECRAFT_FACING_DIRECTION", BlockStateDefinition::enumeration("minecraft:facing_direction", ["down", "up", "north", "south", "west", "east"]).expect("valid Chorus common state")),
        ("MINECRAFT_VERTICAL_HALF", BlockStateDefinition::enumeration("minecraft:vertical_half", ["bottom", "top"]).expect("valid Chorus common state")),
        ("MOISTURIZED_AMOUNT", BlockStateDefinition::integer("moisturized_amount", 0, 7).expect("valid Chorus common state")),
        ("MONSTER_EGG_STONE_TYPE", BlockStateDefinition::enumeration("monster_egg_stone_type", ["chiseled_stone_brick", "cobblestone", "cracked_stone_brick", "mossy_stone_brick", "stone", "stone_brick"]).expect("valid Chorus common state")),
        ("MULTI_FACE_DIRECTION_BITS", BlockStateDefinition::integer("multi_face_direction_bits", 0, 63).expect("valid Chorus common state")),
        ("NATURAL", BlockStateDefinition::boolean("natural", false)),
        ("NETHER_REACTOR_STATE", BlockStateDefinition::enumeration("nether_reactor_state", ["ready", "initialized", "finished"]).expect("valid Chorus common state")),
        ("NEW_LEAF_TYPE", BlockStateDefinition::enumeration("new_leaf_type", ["acacia", "dark_oak"]).expect("valid Chorus common state")),
        ("NEW_LOG_TYPE", BlockStateDefinition::enumeration("new_log_type", ["acacia", "dark_oak"]).expect("valid Chorus common state")),
        ("OCCUPIED_BIT", BlockStateDefinition::boolean("occupied_bit", false)),
        ("OLD_LEAF_TYPE", BlockStateDefinition::enumeration("old_leaf_type", ["oak", "spruce", "birch", "jungle"]).expect("valid Chorus common state")),
        ("OLD_LOG_TYPE", BlockStateDefinition::enumeration("old_log_type", ["oak", "spruce", "birch", "jungle"]).expect("valid Chorus common state")),
        ("OMINOUS", BlockStateDefinition::boolean("ominous", false)),
        ("OPEN_BIT", BlockStateDefinition::boolean("open_bit", false)),
        ("ORIENTATION", BlockStateDefinition::enumeration("orientation", ["down_east", "down_north", "down_south", "down_west", "east_up", "north_up", "south_up", "up_east", "up_north", "up_south", "up_west", "west_up"]).expect("valid Chorus common state")),
        ("OUTPUT_LIT_BIT", BlockStateDefinition::boolean("output_lit_bit", false)),
        ("OUTPUT_SUBTRACT_BIT", BlockStateDefinition::boolean("output_subtract_bit", false)),
        ("OXIDIZATION_LEVEL", BlockStateDefinition::enumeration("oxidization_level", ["unaffected", "exposed", "weathered", "oxidized"]).expect("valid Chorus common state")),
        ("PALE_MOSS_CARPET_SIDE_EAST", BlockStateDefinition::enumeration("pale_moss_carpet_side_east", ["none", "short", "tall"]).expect("valid Chorus common state")),
        ("PALE_MOSS_CARPET_SIDE_NORTH", BlockStateDefinition::enumeration("pale_moss_carpet_side_north", ["none", "short", "tall"]).expect("valid Chorus common state")),
        ("PALE_MOSS_CARPET_SIDE_SOUTH", BlockStateDefinition::enumeration("pale_moss_carpet_side_south", ["none", "short", "tall"]).expect("valid Chorus common state")),
        ("PALE_MOSS_CARPET_SIDE_WEST", BlockStateDefinition::enumeration("pale_moss_carpet_side_west", ["none", "short", "tall"]).expect("valid Chorus common state")),
        ("PERSISTENT_BIT", BlockStateDefinition::boolean("persistent_bit", false)),
        ("PILLAR_AXIS", BlockStateDefinition::enumeration("pillar_axis", ["y", "z", "x"]).expect("valid Chorus common state")),
        ("PORTAL_AXIS", BlockStateDefinition::enumeration("portal_axis", ["unknown", "x", "z"]).expect("valid Chorus common state")),
        ("POTENT_SULFUR_STATE", BlockStateDefinition::enumeration("potent_sulfur_state", ["dry", "wet", "dormant", "erupting", "continuous"]).expect("valid Chorus common state")),
        ("POWERED_BIT", BlockStateDefinition::boolean("powered_bit", false)),
        ("PRISMARINE_BLOCK_TYPE", BlockStateDefinition::enumeration("prismarine_block_type", ["bricks", "dark", "default"]).expect("valid Chorus common state")),
        ("PROPAGULE_STAGE", BlockStateDefinition::integer("propagule_stage", 0, 4).expect("valid Chorus common state")),
        ("RAIL_DATA_BIT", BlockStateDefinition::boolean("rail_data_bit", false)),
        ("RAIL_DIRECTION_10", BlockStateDefinition::integer("rail_direction", 0, 9).expect("valid Chorus common state")),
        ("RAIL_DIRECTION_6", BlockStateDefinition::integer("rail_direction", 0, 5).expect("valid Chorus common state")),
        ("REDSTONE_SIGNAL", BlockStateDefinition::integer("redstone_signal", 0, 15).expect("valid Chorus common state")),
        ("REDSTONE_EAST", BlockStateDefinition::enumeration("redstone_east", ["none", "side", "up"]).expect("valid Chorus common state")),
        ("REDSTONE_NORTH", BlockStateDefinition::enumeration("redstone_north", ["none", "side", "up"]).expect("valid Chorus common state")),
        ("REDSTONE_SOUTH", BlockStateDefinition::enumeration("redstone_south", ["none", "side", "up"]).expect("valid Chorus common state")),
        ("REDSTONE_WEST", BlockStateDefinition::enumeration("redstone_west", ["none", "side", "up"]).expect("valid Chorus common state")),
        ("REHYDRATION_LEVEL", BlockStateDefinition::integer("rehydration_level", 0, 3).expect("valid Chorus common state")),
        ("REPEATER_DELAY", BlockStateDefinition::integer("repeater_delay", 0, 3).expect("valid Chorus common state")),
        ("RESPAWN_ANCHOR_CHARGE", BlockStateDefinition::integer("respawn_anchor_charge", 0, 4).expect("valid Chorus common state")),
        ("ROTATION", BlockStateDefinition::integer("rotation", 0, 3).expect("valid Chorus common state")),
        ("SAND_STONE_TYPE", BlockStateDefinition::enumeration("sand_stone_type", ["default", "heiroglyphs", "cut", "smooth"]).expect("valid Chorus common state")),
        ("SAND_TYPE", BlockStateDefinition::enumeration("sand_type", ["normal", "red"]).expect("valid Chorus common state")),
        ("SCULK_SENSOR_PHASE", BlockStateDefinition::integer("sculk_sensor_phase", 0, 2).expect("valid Chorus common state")),
        ("SEA_GRASS_TYPE", BlockStateDefinition::enumeration("sea_grass_type", ["default", "double_top", "double_bot"]).expect("valid Chorus common state")),
        ("SPONGE_TYPE", BlockStateDefinition::enumeration("sponge_type", ["dry", "wet"]).expect("valid Chorus common state")),
        ("STABILITY", BlockStateDefinition::integer("stability", 0, 7).expect("valid Chorus common state")),
        ("STABILITY_CHECK", BlockStateDefinition::boolean("stability_check", false)),
        ("STONE_BRICK_TYPE", BlockStateDefinition::enumeration("stone_brick_type", ["chiseled", "cracked", "default", "mossy", "smooth"]).expect("valid Chorus common state")),
        ("STONE_SLAB_TYPE", BlockStateDefinition::enumeration("stone_slab_type", ["smooth_stone", "sandstone", "wood", "cobblestone", "brick", "stone_brick", "quarts", "nether_brick"]).expect("valid Chorus common state")),
        ("STONE_SLAB_TYPE_2", BlockStateDefinition::enumeration("stone_slab_type_2", ["red_sandstone", "purpur", "prismarine_rough", "prismarine_dark", "prismarine_brick", "mossy_cobblestone", "smooth_sandstone", "red_nether_brick"]).expect("valid Chorus common state")),
        ("STONE_SLAB_TYPE_3", BlockStateDefinition::enumeration("stone_slab_type_3", ["end_stone_brick", "smooth_red_sandstone", "polished_andesite", "andesite", "diorite", "polished_diorite", "granite", "polished_granite"]).expect("valid Chorus common state")),
        ("STONE_SLAB_TYPE_4", BlockStateDefinition::enumeration("stone_slab_type_4", ["mossy_stone_brick", "smooth_quartz", "stone", "cut_sandstone", "cut_red_sandstone"]).expect("valid Chorus common state")),
        ("STONE_TYPE", BlockStateDefinition::enumeration("stone_type", ["andesite", "andesite_smooth", "diorite", "diorite_smooth", "granite", "granite_smooth", "stone"]).expect("valid Chorus common state")),
        ("STRIPPED_BIT", BlockStateDefinition::boolean("stripped_bit", false)),
        ("STRUCTURE_BLOCK_TYPE", BlockStateDefinition::enumeration("structure_block_type", ["invalid", "data", "save", "load", "corner", "export"]).expect("valid Chorus common state")),
        ("STRUCTURE_VOID_TYPE", BlockStateDefinition::enumeration("structure_void_type", ["air", "void"]).expect("valid Chorus common state")),
        ("SUSPENDED_BIT", BlockStateDefinition::boolean("suspended_bit", false)),
        ("TALL_GRASS_TYPE", BlockStateDefinition::enumeration("tall_grass_type", ["default", "tall", "fern", "snow"]).expect("valid Chorus common state")),
        ("TIP", BlockStateDefinition::boolean("tip", false)),
        ("TOGGLE_BIT", BlockStateDefinition::boolean("toggle_bit", false)),
        ("TORCH_FACING_DIRECTION", BlockStateDefinition::enumeration("torch_facing_direction", ["unknown", "west", "east", "north", "south", "top"]).expect("valid Chorus common state")),
        ("TRIAL_SPAWNER_STATE", BlockStateDefinition::integer("trial_spawner_state", 0, 5).expect("valid Chorus common state")),
        ("TRIGGERED_BIT", BlockStateDefinition::boolean("triggered_bit", false)),
        ("TURTLE_EGG_COUNT", BlockStateDefinition::enumeration("turtle_egg_count", ["one_egg", "two_egg", "three_egg", "four_egg"]).expect("valid Chorus common state")),
        ("TWISTING_VINES_AGE", BlockStateDefinition::integer("twisting_vines_age", 0, 25).expect("valid Chorus common state")),
        ("UPDATE_BIT", BlockStateDefinition::boolean("update_bit", false)),
        ("UPPER_BLOCK_BIT", BlockStateDefinition::boolean("upper_block_bit", false)),
        ("UPSIDE_DOWN_BIT", BlockStateDefinition::boolean("upside_down_bit", false)),
        ("VAULT_STATE", BlockStateDefinition::enumeration("vault_state", ["inactive", "active", "unlocking", "ejecting"]).expect("valid Chorus common state")),
        ("VINE_DIRECTION_BITS", BlockStateDefinition::integer("vine_direction_bits", 0, 15).expect("valid Chorus common state")),
        ("WALL_BLOCK_TYPE", BlockStateDefinition::enumeration("wall_block_type", ["andesite", "brick", "cobblestone", "diorite", "end_brick", "granite", "mossy_cobblestone", "mossy_stone_brick", "nether_brick", "prismarine", "red_nether_brick", "red_sandstone", "sandstone", "stone_brick"]).expect("valid Chorus common state")),
        ("WALL_CONNECTION_TYPE_EAST", BlockStateDefinition::enumeration("wall_connection_type_east", ["none", "short", "tall"]).expect("valid Chorus common state")),
        ("WALL_CONNECTION_TYPE_NORTH", BlockStateDefinition::enumeration("wall_connection_type_north", ["none", "short", "tall"]).expect("valid Chorus common state")),
        ("WALL_CONNECTION_TYPE_SOUTH", BlockStateDefinition::enumeration("wall_connection_type_south", ["none", "short", "tall"]).expect("valid Chorus common state")),
        ("WALL_CONNECTION_TYPE_WEST", BlockStateDefinition::enumeration("wall_connection_type_west", ["none", "short", "tall"]).expect("valid Chorus common state")),
        ("WALL_POST_BIT", BlockStateDefinition::boolean("wall_post_bit", false)),
        ("WEEPING_VINES_AGE", BlockStateDefinition::integer("weeping_vines_age", 0, 25).expect("valid Chorus common state")),
        ("WEIRDO_DIRECTION", BlockStateDefinition::integer("weirdo_direction", 0, 3).expect("valid Chorus common state")),
        ("WOOD_TYPE", BlockStateDefinition::enumeration("wood_type", ["oak", "spruce", "birch", "jungle", "acacia", "dark_oak", "cherry", "pale_oak", "mangrove"]).expect("valid Chorus common state")),
        ("POWERED_SHELF_TYPE", BlockStateDefinition::integer("powered_shelf_type", 0, 3).expect("valid Chorus common state")),
        ("DIRECTION_16", BlockStateDefinition::integer("direction", 0, 15).expect("valid Chorus common state")),
        ("MINECRAFT_CORNER", BlockStateDefinition::enumeration("minecraft:corner", ["none", "inner_left", "inner_right", "outer_left", "outer_right"]).expect("valid Chorus common state")),
        ("MINECRAFT_CONNECTION_DOWN", BlockStateDefinition::boolean("minecraft:connection_down", false)),
        ("MINECRAFT_CONNECTION_EAST", BlockStateDefinition::boolean("minecraft:connection_east", false)),
        ("MINECRAFT_CONNECTION_NORTH", BlockStateDefinition::boolean("minecraft:connection_north", false)),
        ("MINECRAFT_CONNECTION_SOUTH", BlockStateDefinition::boolean("minecraft:connection_south", false)),
        ("MINECRAFT_CONNECTION_UP", BlockStateDefinition::boolean("minecraft:connection_up", false)),
        ("MINECRAFT_CONNECTION_WEST", BlockStateDefinition::boolean("minecraft:connection_west", false)),
    ]
}

/// Clone a common state by its upstream constant name.
pub fn get(constant_name: &str) -> Option<BlockStateDefinition> {
    all().into_iter().find_map(|(name, definition)| (name == constant_name).then_some(definition))
}

/// Return every shared definition with this Bedrock state identifier.
pub fn by_identifier(identifier: &str) -> Vec<BlockStateDefinition> {
    all().into_iter().filter_map(|(_, definition)| (definition.identifier() == identifier).then_some(definition)).collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test] fn imports_complete_shared_state_catalog() {
        assert_eq!(all().len(), 164);
        assert!(get("ACTIVE").is_some());
        assert!(get("WALL_CONNECTION_TYPE_NORTH").is_some());
    }
    #[test] fn wall_and_facing_state_values_match_chorus_order() {
        let wall = get("WALL_CONNECTION_TYPE_NORTH").unwrap();
        assert_eq!(wall.values().len(), 3);
        assert_eq!(wall.default_value(), &"none".into());
        let facing = get("FACING_DIRECTION").unwrap();
        assert_eq!(facing.values().len(), 6);
        assert_eq!(facing.values()[0], 0.into());
        assert_eq!(facing.values()[5], 5.into());
    }
    #[test] fn identifier_lookup_preserves_alternate_direction_ranges() {
        let directions = by_identifier("direction");
        assert!(directions.iter().any(|state| state.values().len() == 4));
        assert!(directions.iter().any(|state| state.values().len() == 16));
    }
}