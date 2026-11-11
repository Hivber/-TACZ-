//! Data-only block definitions, permutations, and registry.
//! Adapted from Chorus block_definition.rs, block_permutation.rs and block_registry.rs
//! at bedrock-crustaceans/Chorus revision c8115402a3bacf2014af2ce72fa5c35729040aa8.
//! Apache-2.0. No Bevy ECS, packet handlers, transport, NBT or Chorus runtime IDs
//! are imported. Permutation indices here are local state-combination indices,
//! not Bedrock runtime IDs or network palette IDs.

use crate::block_state::{BlockState, BlockStateDefinition};
use std::collections::{BTreeMap, HashMap};

#[derive(Clone, Debug)]
pub struct BlockDefinition {
    identifier: String,
    states: Vec<BlockStateDefinition>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct BlockPermutation {
    identifier: String,
    states: BTreeMap<String, BlockState>,
    index: u16,
}

impl BlockDefinition {
    pub fn new<I>(identifier: impl Into<String>, states: I) -> Self
    where I: IntoIterator<Item = BlockStateDefinition> {
        Self { identifier: identifier.into(), states: states.into_iter().collect() }
    }

    pub fn identifier(&self) -> &str { &self.identifier }
    pub fn states(&self) -> &[BlockStateDefinition] { &self.states }

    pub fn validate(&self) -> Result<usize, String> {
        let (namespace, name) = self.identifier.split_once(':')
            .ok_or_else(|| format!("block identifier {:?} must be namespaced", self.identifier))?;
        if namespace.is_empty() || name.is_empty() || self.identifier.chars().any(char::is_whitespace) {
            return Err(format!("invalid block identifier {:?}", self.identifier));
        }

        let mut names = std::collections::HashSet::new();
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
        Ok(count)
    }

    pub fn generate(&self) -> Result<Vec<BlockPermutation>, String> {
        let count = self.validate()?;
        let mut ordered: Vec<&BlockStateDefinition> = self.states.iter().collect();
        ordered.sort_by(|a, b| a.identifier().cmp(b.identifier()));

        let mut permutations = Vec::with_capacity(count);
        for index in 0..count {
            let mut remainder = index;
            let mut states = BTreeMap::new();
            // Mixed-radix decoding mirrors Chorus's sorted state index order.
            for state in ordered.iter().rev() {
                let radix = state.values().len();
                let digit = remainder % radix;
                remainder /= radix;
                states.insert(state.identifier().to_string(), state.values()[digit].clone());
            }
            permutations.push(BlockPermutation {
                identifier: self.identifier.clone(),
                states,
                index: index as u16,
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

    pub fn permutation(&self, identifier: &str, states: &BTreeMap<String, BlockState>) -> Option<&BlockPermutation> {
        self.permutations.get(identifier)?.iter().find(|permutation| &permutation.states == states)
    }

    pub fn block_count(&self) -> usize { self.definitions.len() }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample_definition() -> BlockDefinition {
        BlockDefinition::new("minecraft:test_block", [
            BlockStateDefinition::boolean("open_bit", false),
            BlockStateDefinition::integer("level", 0, 2).unwrap(),
            BlockStateDefinition::enumeration("axis", ["x", "y"]).unwrap(),
        ])
    }

    #[test]
    fn generates_cartesian_product_with_stable_indices() {
        let definition = sample_definition();
        let permutations = definition.generate().unwrap();
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
        let permutation = registry.permutation("minecraft:test_block", &states).unwrap();
        assert_eq!(permutation.index(), 9);
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
}
