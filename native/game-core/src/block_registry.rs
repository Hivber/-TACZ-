//! Data-only block definitions, state permutations, component overrides, and registry.
//! Adapted from Chorus block_definition.rs, block_permutation.rs and component storage
//! at bedrock-crustaceans/Chorus revision c8115402a3bacf2014af2ce72fa5c35729040aa8.
//! Apache-2.0. No Bevy ECS, packet handlers, transport, NBT or Chorus runtime IDs
//! are imported. Permutation indices here are local state-combination indices,
//! not Bedrock runtime IDs or network palette IDs.

use crate::block_break::{BlockBreakProfile, MiningTool};
use crate::block_drops::{resolve_block_drops, BlockDropContext, DropStack};
use crate::block_state::{BlockState, BlockStateDefinition};
use crate::components::{BlockComponent, BlockComponents};
use crate::random::RandomSource;
use std::collections::{BTreeMap, HashMap, HashSet};
use std::sync::Arc;

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum StateCondition {
    Equals(BlockState),
    NotEquals(BlockState),
    OneOf(Vec<BlockState>),
}

impl StateCondition {
    fn matches(&self, state: Option<&BlockState>) -> bool {
        match (self, state) {
            (Self::Equals(expected), Some(actual)) => actual == expected,
            (Self::NotEquals(expected), Some(actual)) => actual != expected,
            (Self::OneOf(expected), Some(actual)) => expected.contains(actual),
            _ => false,
        }
    }
}

#[derive(Clone, Debug)]
pub struct BlockPermutationDefinition {
    conditions: BTreeMap<String, StateCondition>,
    components: Vec<BlockComponent>,
}

impl BlockPermutationDefinition {
    pub fn new<I>(conditions: I, components: impl IntoIterator<Item = BlockComponent>) -> Self
    where I: IntoIterator<Item = (String, StateCondition)> {
        Self {
            conditions: conditions.into_iter().collect(),
            components: components.into_iter().collect(),
        }
    }

    pub fn when_state(
        identifier: impl Into<String>,
        condition: StateCondition,
        components: impl IntoIterator<Item = BlockComponent>,
    ) -> Self {
        Self::new([(identifier.into(), condition)], components)
    }

    fn matches(&self, states: &BTreeMap<String, BlockState>) -> bool {
        self.conditions.iter().all(|(name, condition)| condition.matches(states.get(name)))
    }
}

#[derive(Clone, Debug)]
pub struct BlockDefinition {
    identifier: String,
    states: Vec<BlockStateDefinition>,
    components: Vec<BlockComponent>,
    permutations: Vec<BlockPermutationDefinition>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct BlockPermutation {
    identifier: String,
    states: BTreeMap<String, BlockState>,
    index: u16,
    components: Arc<BlockComponents>,
}

impl BlockDefinition {
    pub fn new<I>(identifier: impl Into<String>, states: I) -> Self
    where I: IntoIterator<Item = BlockStateDefinition> {
        Self::with_components(identifier, states, [], [])
    }

    pub fn with_components<I, C, P>(
        identifier: impl Into<String>,
        states: I,
        components: C,
        permutations: P,
    ) -> Self
    where
        I: IntoIterator<Item = BlockStateDefinition>,
        C: IntoIterator<Item = BlockComponent>,
        P: IntoIterator<Item = BlockPermutationDefinition>,
    {
        Self {
            identifier: identifier.into(),
            states: states.into_iter().collect(),
            components: components.into_iter().collect(),
            permutations: permutations.into_iter().collect(),
        }
    }

    pub fn identifier(&self) -> &str { &self.identifier }
    pub fn states(&self) -> &[BlockStateDefinition] { &self.states }
    pub fn components(&self) -> &[BlockComponent] { &self.components }
    pub fn permutation_definitions(&self) -> &[BlockPermutationDefinition] { &self.permutations }

    pub fn validate(&self) -> Result<usize, String> {
        let (namespace, name) = self.identifier.split_once(':')
            .ok_or_else(|| format!("block identifier {:?} must be namespaced", self.identifier))?;
        if namespace.is_empty() || name.is_empty() || self.identifier.chars().any(char::is_whitespace) {
            return Err(format!("invalid block identifier {:?}", self.identifier));
        }

        let mut names = HashSet::new();
        let mut count = 1usize;
        for state in &self.states {
            state.validate()?;
            if !names.insert(state.identifier()) {
                return Err(format!("duplicate state identifier {:?}", state.identifier()));
            }
            count = count.checked_mul(state.values().len())
                .ok_or_else(|| "block permutation count overflow".to_string())?;
            if count > u16::MAX as usize {
                return Err(format!("block {:?} exceeds 65535 permutations", self.identifier));
            }
        }

        for component in &self.components { component.validate()?; }
        for permutation in &self.permutations {
            for component in &permutation.components { component.validate()?; }
            for (state_name, condition) in &permutation.conditions {
                let definition = self.states.iter()
                    .find(|state| state.identifier() == state_name.as_str())
                    .ok_or_else(|| format!("permutation condition references unknown state {:?}", state_name))?;
                let values: Vec<&BlockState> = match condition {
                    StateCondition::Equals(value) | StateCondition::NotEquals(value) => vec![value],
                    StateCondition::OneOf(values) => values.iter().collect(),
                };
                if values.is_empty() || values.iter().any(|value| definition.value_index(value).is_none()) {
                    return Err(format!("permutation condition for state {:?} contains an invalid value", state_name));
                }
            }
        }
        Ok(count)
    }

    pub fn generate(&self) -> Result<Vec<BlockPermutation>, String> {
        let count = self.validate()?;
        let mut ordered: Vec<&BlockStateDefinition> = self.states.iter().collect();
        ordered.sort_by(|a, b| a.identifier().cmp(b.identifier()));

        let mut shared_components: HashMap<Vec<bool>, Arc<BlockComponents>> = HashMap::new();
        let mut permutations = Vec::with_capacity(count);
        for index in 0..count {
            let mut remainder = index;
            let mut states = BTreeMap::new();
            for state in ordered.iter().rev() {
                let radix = state.values().len();
                let digit = remainder % radix;
                remainder /= radix;
                states.insert(state.identifier().to_string(), state.values()[digit].clone());
            }

            let matched: Vec<bool> = self.permutations.iter()
                .map(|definition| definition.matches(&states))
                .collect();
            let components = shared_components.entry(matched.clone()).or_insert_with(|| {
                let mut components = BlockComponents::new();
                for component in &self.components { components.insert(component.clone()); }
                for (definition, is_matched) in self.permutations.iter().zip(matched.iter()) {
                    if *is_matched {
                        for component in &definition.components { components.insert(component.clone()); }
                    }
                }
                Arc::new(components)
            }).clone();

            permutations.push(BlockPermutation {
                identifier: self.identifier.clone(),
                states,
                index: index as u16,
                components,
            });
        }
        Ok(permutations)
    }
}

impl BlockPermutation {
    pub fn identifier(&self) -> &str { &self.identifier }
    pub fn states(&self) -> &BTreeMap<String, BlockState> { &self.states }
    pub fn index(&self) -> u16 { self.index }
    pub fn state(&self, identifier: &str) -> Option<&BlockState> { self.states.get(identifier) }
    pub fn components(&self) -> &BlockComponents { self.components.as_ref() }

    /// Resolve the mining profile from this exact state permutation. The held
    /// item identifier is used for item-specific hardness overrides.
    pub fn break_profile(&self, held_item: Option<&str>) -> Option<BlockBreakProfile> {
        BlockBreakProfile::from_components_for_item(self.components(), held_item)
    }

    /// Resolve drops from this exact state permutation, not from the block's
    /// base definition. This preserves state-specific loot component overrides.
    pub fn resolve_drops<R: RandomSource>(
        &self,
        context: BlockDropContext,
        rng: &mut R,
    ) -> Vec<DropStack> {
        resolve_block_drops(&self.identifier, self.components(), context, rng)
    }
}

#[derive(Default)]
pub struct BlockRegistry {
    definitions: HashMap<String, BlockDefinition>,
    permutations: HashMap<String, Vec<BlockPermutation>>,
}

impl BlockRegistry {
    pub fn new() -> Self { Self::default() }

    pub fn register(&mut self, definition: BlockDefinition) -> Result<(), String> {
        if self.definitions.contains_key(definition.identifier()) {
            return Err(format!("block {:?} is already registered", definition.identifier()));
        }
        let permutations = definition.generate()?;
        self.permutations.insert(definition.identifier().to_string(), permutations);
        self.definitions.insert(definition.identifier().to_string(), definition);
        Ok(())
    }

    pub fn definition(&self, identifier: &str) -> Option<&BlockDefinition> {
        self.definitions.get(identifier)
    }

    pub fn default_permutation(&self, identifier: &str) -> Option<&BlockPermutation> {
        self.permutations.get(identifier)?.first()
    }

    pub fn permutation_count(&self, identifier: &str) -> Option<usize> {
        self.permutations.get(identifier).map(Vec::len)
    }

    pub fn permutation(
        &self,
        identifier: &str,
        states: &BTreeMap<String, BlockState>,
    ) -> Option<&BlockPermutation> {
        self.permutations.get(identifier)?.iter()
            .find(|permutation| &permutation.states == states)
    }

    pub fn block_count(&self) -> usize { self.definitions.len() }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::components::{BlockComponent, BlockComponentKind, MineableComponent};

    fn sample_definition() -> BlockDefinition {
        BlockDefinition::new("minecraft:test_block", [
            BlockStateDefinition::boolean("open_bit", false),
            BlockStateDefinition::integer("level", 0, 2).unwrap(),
            BlockStateDefinition::enumeration("axis", ["x", "y"]).unwrap(),
        ])
    }

    #[test]
    fn generates_cartesian_product_with_stable_indices() {
        let permutations = sample_definition().generate().unwrap();
        assert_eq!(permutations.len(), 12);
        let indices: Vec<u16> = permutations.iter().map(BlockPermutation::index).collect();
        assert_eq!(indices, (0..12).collect::<Vec<_>>());
        assert_eq!(permutations[0].state("open_bit"), Some(&BlockState::Bool(false)));
        assert_eq!(permutations[0].state("level"), Some(&BlockState::Int(0)));
        assert_eq!(permutations[0].state("axis"), Some(&BlockState::from("x")));
    }

    #[test]
    fn stateless_block_has_one_default_permutation() {
        let mut registry = BlockRegistry::new();
        registry.register(BlockDefinition::new("minecraft:air", [])).unwrap();
        let permutation = registry.default_permutation("minecraft:air").unwrap();
        assert_eq!(permutation.index(), 0);
        assert!(permutation.states().is_empty());
        assert_eq!(registry.permutation_count("minecraft:air"), Some(1));
    }

    #[test]
    fn registry_resolves_a_specific_state_combination() {
        let mut registry = BlockRegistry::new();
        registry.register(sample_definition()).unwrap();
        let states = [
            ("axis".to_string(), BlockState::from("y")),
            ("level".to_string(), BlockState::Int(1)),
            ("open_bit".to_string(), BlockState::Bool(true)),
        ].into_iter().collect();
        assert_eq!(registry.permutation("minecraft:test_block", &states).unwrap().index(), 9);
    }

    #[test]
    fn invalid_and_duplicate_definitions_are_rejected() {
        let mut registry = BlockRegistry::new();
        assert!(registry.register(BlockDefinition::new("not_namespaced", [])).is_err());
        registry.register(BlockDefinition::new("minecraft:air", [])).unwrap();
        assert!(registry.register(BlockDefinition::new("minecraft:air", [])).is_err());
    }

    #[test]
    fn too_many_permutations_are_rejected_before_allocation() {
        let state = BlockStateDefinition::integer("wide", 0, 255).unwrap();
        let other = BlockStateDefinition::integer("wide2", 0, 255).unwrap();
        assert!(BlockDefinition::new("minecraft:too_many", [state, other]).generate().is_err());
    }

    #[test]
    fn conditional_component_overrides_merge_on_matching_states_only() {
        let definition = BlockDefinition::with_components(
            "minecraft:lamp",
            [BlockStateDefinition::boolean("lit", false)],
            [BlockComponent::Mineable(MineableComponent::hardness(1.0)), BlockComponent::Friction(0.6)],
            [BlockPermutationDefinition::when_state(
                "lit",
                StateCondition::Equals(BlockState::Bool(true)),
                [BlockComponent::Mineable(MineableComponent::hardness(0.5)), BlockComponent::Solid(false)],
            )],
        );
        let permutations = definition.generate().unwrap();
        assert_eq!(permutations[0].components().mineable().unwrap().hardness, 1.0);
        assert_eq!(permutations[1].components().mineable().unwrap().hardness, 0.5);
        assert_eq!(permutations[0].components().get(BlockComponentKind::Solid), None);
        assert_eq!(permutations[1].components().get(BlockComponentKind::Solid), Some(&BlockComponent::Solid(false)));
        assert_eq!(permutations[0].components().len(), 2);
    }

    #[test]
    fn permutation_resolves_mining_profile_and_item_specific_hardness() {
        let definition = BlockDefinition::with_components(
            "minecraft:custom_ore",
            [BlockStateDefinition::boolean("deepslate", false)],
            [BlockComponent::Mineable(
                MineableComponent::hardness(3.0)
                    .with_tool_requirement(crate::block_break::tool_type::PICKAXE, 2)
                    .with_item_hardness("minecraft:diamond_pickaxe", 0.75),
            )],
            [],
        );
        let permutations = definition.generate().unwrap();
        let profile = permutations[0].break_profile(Some("minecraft:diamond_pickaxe")).unwrap();
        assert_eq!(profile.hardness, 0.75);
        assert_eq!(profile.tool_type, crate::block_break::tool_type::PICKAXE);
        assert_eq!(profile.harvest_level, 2);
        assert!(permutations[0].break_profile(Some("minecraft:iron_pickaxe")).unwrap().hardness == 3.0);
        assert!(permutations[0].break_profile(None).is_some());
        assert!(BlockDefinition::new("minecraft:non_mineable", []).generate().unwrap()[0]
            .break_profile(Some("minecraft:diamond_pickaxe")).is_none());
    }

    #[test]
    fn permutation_resolves_state_specific_loot_overrides() {
        use crate::components::{LootComponent, LootStack};
        use crate::block_drops::BlockDropContext;
        struct FixedRng;
        impl RandomSource for FixedRng {
            fn next_u64(&mut self) -> u64 { 0 }
        }

        let definition = BlockDefinition::with_components(
            "minecraft:lamp",
            [BlockStateDefinition::boolean("lit", false)],
            [BlockComponent::Loot(LootComponent::item("minecraft:coal"))],
            [BlockPermutationDefinition::when_state(
                "lit",
                StateCondition::Equals(BlockState::Bool(true)),
                [BlockComponent::Loot(LootComponent::item("minecraft:glowstone_dust"))],
            )],
        );
        let permutations = definition.generate().unwrap();
        let survival = BlockDropContext { survival: true, ..Default::default() };
        let mut rng = FixedRng;
        assert_eq!(permutations[0].resolve_drops(survival, &mut rng),
            vec![DropStack { item: "minecraft:coal".into(), count: 1 }]);
        assert_eq!(permutations[1].resolve_drops(survival, &mut rng),
            vec![DropStack { item: "minecraft:glowstone_dust".into(), count: 1 }]);
        assert!(permutations[1].resolve_drops(
            BlockDropContext { survival: false, ..Default::default() }, &mut rng,
        ).is_empty());
        let _ = LootStack::fixed("minecraft:stone", 1);
    }

    #[test]
    fn invalid_permutation_state_conditions_are_rejected() {
        let definition = BlockDefinition::with_components(
            "minecraft:lamp",
            [BlockStateDefinition::boolean("lit", false)],
            [],
            [BlockPermutationDefinition::when_state(
                "lit", StateCondition::Equals(BlockState::Int(1)), [],
            )],
        );
        assert!(definition.generate().is_err());
    }
}
