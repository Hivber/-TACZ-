//! Data-only block components adapted from Chorus.
//! Upstream: bedrock-crustaceans/Chorus @ c8115402a3bacf2014af2ce72fa5c35729040aa8.
//! Apache-2.0. Explicit enum storage removes Bevy, atomicow and runtime dependencies.

use std::collections::BTreeMap;

#[derive(Clone, Copy, Debug, Eq, PartialEq, Ord, PartialOrd, Hash)]
pub enum BlockComponentKind {
    Mineable, Loot, CollisionBox, Flammable, Friction, InternalFriction,
    LightDampening, LightEmission, MapColor, Moveable, Replaceable, Solid, Transparent,
}

#[derive(Clone, Debug, PartialEq)]
pub struct MineableComponent {
    pub hardness: f32,
    pub item_specific_hardness: BTreeMap<String, f32>,
    pub tool_type: u8,
    pub harvest_level: u8,
    pub requires_correct_tool: bool,
}

impl MineableComponent {
    pub fn hardness(hardness: f32) -> Self {
        Self {
            hardness, item_specific_hardness: BTreeMap::new(),
            tool_type: 0, harvest_level: 0, requires_correct_tool: false,
        }
    }

    pub fn with_tool_requirement(mut self, tool_type: u8, harvest_level: u8) -> Self {
        self.tool_type = tool_type;
        self.harvest_level = harvest_level;
        self.requires_correct_tool = true;
        self
    }

    pub fn with_item_hardness(mut self, item: impl Into<String>, hardness: f32) -> Self {
        self.item_specific_hardness.insert(item.into(), hardness);
        self
    }

    pub fn hardness_for_item(&self, item: Option<&str>) -> f32 {
        item.and_then(|name| self.item_specific_hardness.get(name).copied())
            .unwrap_or(self.hardness)
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct LootStack {
    pub item: String,
    pub min_count: u16,
    pub max_count: u16,
    /// Maximum additional items per Fortune level, rolled independently.
    pub fortune_bonus_per_level: u16,
}

impl LootStack {
    pub fn fixed(item: impl Into<String>, count: u16) -> Self {
        Self { item: item.into(), min_count: count, max_count: count, fortune_bonus_per_level: 0 }
    }

    pub fn ranged(item: impl Into<String>, min_count: u16, max_count: u16) -> Self {
        Self { item: item.into(), min_count, max_count, fortune_bonus_per_level: 0 }
    }

    pub fn with_fortune_bonus(mut self, per_level: u16) -> Self {
        self.fortune_bonus_per_level = per_level;
        self
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct LootOutcome {
    pub weight: u32,
    pub items: Vec<LootStack>,
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct LootComponent {
    /// Empty outcomes explicitly mean that the block drops nothing.
    pub outcomes: Vec<LootOutcome>,
    /// Optional silk-touch-specific result. None falls back to normal outcomes.
    pub silk_touch: Option<Vec<LootStack>>,
}

impl LootComponent {
    pub fn item(item: impl Into<String>) -> Self {
        Self {
            outcomes: vec![LootOutcome { weight: 1, items: vec![LootStack::fixed(item, 1)] }],
            silk_touch: None,
        }
    }

    pub fn nothing() -> Self { Self::default() }

    pub fn weighted(outcomes: impl IntoIterator<Item = LootOutcome>) -> Self {
        Self { outcomes: outcomes.into_iter().collect(), silk_touch: None }
    }

    pub fn with_silk_touch(mut self, items: impl IntoIterator<Item = LootStack>) -> Self {
        self.silk_touch = Some(items.into_iter().collect());
        self
    }

    pub fn validate(&self) -> Result<(), String> {
        if let Some(items) = &self.silk_touch { validate_loot_stacks(items)?; }
        let mut total_weight = 0u32;
        for outcome in &self.outcomes {
            if outcome.weight == 0 {
                return Err("loot outcome weight must be greater than zero".into());
            }
            total_weight = total_weight.checked_add(outcome.weight)
                .ok_or_else(|| "loot outcome weights exceed u32::MAX".to_string())?;
            validate_loot_stacks(&outcome.items)?;
        }
        Ok(())
    }
}

fn validate_loot_stacks(items: &[LootStack]) -> Result<(), String> {
    for stack in items {
        let (namespace, name) = stack.item.split_once(':')
            .ok_or_else(|| format!("loot item {:?} must be namespaced", stack.item))?;
        if namespace.is_empty() || name.is_empty() || stack.item.chars().any(char::is_whitespace) {
            return Err(format!("invalid loot item identifier {:?}", stack.item));
        }
        if stack.min_count > stack.max_count {
            return Err(format!("loot item {:?} has min_count greater than max_count", stack.item));
        }
    }
    Ok(())
}

#[derive(Clone, Debug, PartialEq)]
pub struct CollisionBoxComponent {
    pub origin: [f32; 3],
    pub size: [f32; 3],
    pub enabled: bool,
}

impl Default for CollisionBoxComponent {
    fn default() -> Self {
        Self { origin: [0.0; 3], size: [1.0; 3], enabled: true }
    }
}

impl CollisionBoxComponent {
    pub fn new(origin: [f32; 3], size: [f32; 3]) -> Self {
        Self { origin, size, enabled: true }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct FlammableComponent {
    pub catch_chance: i32,
    pub destroy_chance: i32,
}

#[derive(Clone, Debug, PartialEq)]
pub struct MoveableComponent {
    pub movement: Movement,
    pub sticky: bool,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Movement { Both, Push, Break, None }

#[derive(Clone, Debug, PartialEq)]
pub enum BlockComponent {
    Mineable(MineableComponent),
    Loot(LootComponent),
    CollisionBox(CollisionBoxComponent),
    Flammable(FlammableComponent),
    Friction(f32),
    InternalFriction(f32),
    LightDampening(i32),
    LightEmission(i32),
    MapColor([u8; 4]),
    Moveable(MoveableComponent),
    Replaceable(bool),
    Solid(bool),
    Transparent(bool),
}

impl BlockComponent {
    pub fn kind(&self) -> BlockComponentKind {
        match self {
            Self::Mineable(_) => BlockComponentKind::Mineable,
            Self::Loot(_) => BlockComponentKind::Loot,
            Self::CollisionBox(_) => BlockComponentKind::CollisionBox,
            Self::Flammable(_) => BlockComponentKind::Flammable,
            Self::Friction(_) => BlockComponentKind::Friction,
            Self::InternalFriction(_) => BlockComponentKind::InternalFriction,
            Self::LightDampening(_) => BlockComponentKind::LightDampening,
            Self::LightEmission(_) => BlockComponentKind::LightEmission,
            Self::MapColor(_) => BlockComponentKind::MapColor,
            Self::Moveable(_) => BlockComponentKind::Moveable,
            Self::Replaceable(_) => BlockComponentKind::Replaceable,
            Self::Solid(_) => BlockComponentKind::Solid,
            Self::Transparent(_) => BlockComponentKind::Transparent,
        }
    }

    pub fn validate(&self) -> Result<(), String> {
        match self {
            Self::Mineable(value) => {
                if !value.hardness.is_finite() || value.item_specific_hardness.values().any(|v| !v.is_finite()) {
                    return Err("mineable hardness values must be finite".into());
                }
            }
            Self::Loot(value) => value.validate()?,
            Self::CollisionBox(value) => {
                if value.origin.iter().chain(value.size.iter()).any(|v| !v.is_finite())
                    || value.size.iter().any(|v| *v < 0.0) {
                    return Err("collision box coordinates must be finite and sizes non-negative".into());
                }
            }
            Self::Flammable(value) => {
                if !(0..=100).contains(&value.catch_chance) || !(0..=100).contains(&value.destroy_chance) {
                    return Err("flammability chances must be in 0..=100".into());
                }
            }
            Self::Friction(value) | Self::InternalFriction(value) => {
                if !value.is_finite() || *value < 0.0 {
                    return Err("friction must be finite and non-negative".into());
                }
            }
            Self::LightDampening(value) | Self::LightEmission(value) => {
                if !(0..=15).contains(value) { return Err("light values must be in 0..=15".into()); }
            }
            Self::MapColor(_) | Self::Moveable(_) | Self::Replaceable(_) | Self::Solid(_) | Self::Transparent(_) => {}
        }
        Ok(())
    }
}

/// Component storage. Inserting a component replaces the previous component
/// of the same kind, matching Chorus's TypeId-keyed override semantics.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct BlockComponents {
    values: BTreeMap<BlockComponentKind, BlockComponent>,
}

impl BlockComponents {
    pub fn new() -> Self { Self::default() }

    pub fn insert(&mut self, component: BlockComponent) -> Option<BlockComponent> {
        self.values.insert(component.kind(), component)
    }

    pub fn get(&self, kind: BlockComponentKind) -> Option<&BlockComponent> {
        self.values.get(&kind)
    }

    pub fn mineable(&self) -> Option<&MineableComponent> {
        match self.get(BlockComponentKind::Mineable) {
            Some(BlockComponent::Mineable(value)) => Some(value),
            _ => None,
        }
    }

    pub fn loot(&self) -> Option<&LootComponent> {
        match self.get(BlockComponentKind::Loot) {
            Some(BlockComponent::Loot(value)) => Some(value),
            _ => None,
        }
    }

    pub fn contains(&self, kind: BlockComponentKind) -> bool { self.values.contains_key(&kind) }
    pub fn len(&self) -> usize { self.values.len() }
    pub fn is_empty(&self) -> bool { self.values.is_empty() }

    pub fn iter(&self) -> impl Iterator<Item = (&BlockComponentKind, &BlockComponent)> {
        self.values.iter()
    }

    pub fn merge_from(&mut self, other: &Self) {
        for component in other.values.values() { self.insert(component.clone()); }
    }

    pub fn validate(&self) -> Result<(), String> {
        for component in self.values.values() { component.validate()?; }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn inserting_same_kind_replaces_previous_value() {
        let mut components = BlockComponents::new();
        components.insert(BlockComponent::Friction(0.6));
        components.insert(BlockComponent::Friction(0.8));
        assert_eq!(components.len(), 1);
        assert_eq!(components.get(BlockComponentKind::Friction), Some(&BlockComponent::Friction(0.8)));
    }

    #[test]
    fn item_specific_hardness_overrides_default() {
        let mineable = MineableComponent::hardness(3.0)
            .with_item_hardness("minecraft:diamond_pickaxe", 0.5);
        assert_eq!(mineable.hardness_for_item(Some("minecraft:diamond_pickaxe")), 0.5);
        assert_eq!(mineable.hardness_for_item(Some("minecraft:iron_pickaxe")), 3.0);
    }

    #[test]
    fn invalid_loot_and_physics_components_are_rejected() {
        let invalid_loot = LootComponent::weighted([LootOutcome {
            weight: 0, items: vec![LootStack::fixed("minecraft:stone", 1)],
        }]);
        assert!(invalid_loot.validate().is_err());
        assert!(BlockComponent::Friction(f32::NAN).validate().is_err());
    }
}
