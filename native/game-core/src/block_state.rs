//! Owned block-state model adapted from Chorus `crates/block/src/state/block_state.rs`.
//! Upstream: bedrock-crustaceans/Chorus @ c8115402a3bacf2014af2ce72fa5c35729040aa8.
//! Apache-2.0. This port intentionally removes atomicow/Bevy/NBT dependencies.

use std::collections::BTreeSet;
use std::sync::Arc;

#[derive(Clone, Debug, Eq, PartialEq, Ord, PartialOrd, Hash)]
pub enum BlockState {
    Bool(bool),
    Int(i32),
    Enum(Arc<str>),
}

impl From<bool> for BlockState {
    fn from(value: bool) -> Self { Self::Bool(value) }
}

impl From<i32> for BlockState {
    fn from(value: i32) -> Self { Self::Int(value) }
}

impl From<&str> for BlockState {
    fn from(value: &str) -> Self { Self::Enum(Arc::from(value)) }
}

impl From<String> for BlockState {
    fn from(value: String) -> Self { Self::Enum(Arc::from(value)) }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct BlockStateDefinition {
    identifier: String,
    values: Vec<BlockState>,
}

impl BlockStateDefinition {
    pub fn boolean(identifier: impl Into<String>, default: bool) -> Self {
        Self { identifier: identifier.into(), values: vec![BlockState::Bool(default), BlockState::Bool(!default)] }
    }

    pub fn integer(identifier: impl Into<String>, min: i32, max: i32) -> Result<Self, String> {
        let (min, max) = if min <= max { (min, max) } else { (max, min) };
        let count = (max as i64 - min as i64 + 1) as usize;
        if count < 2 || count > u16::MAX as usize {
            return Err(format!("integer state range must contain 2..=65535 values, got {count}"));
        }
        Ok(Self { identifier: identifier.into(), values: (min..=max).map(BlockState::Int).collect() })
    }

    pub fn enumeration<I, S>(identifier: impl Into<String>, values: I) -> Result<Self, String>
    where I: IntoIterator<Item = S>, S: Into<String> {
        let values: Vec<BlockState> = values.into_iter().map(|v| BlockState::Enum(Arc::from(v.into()))).collect();
        let definition = Self { identifier: identifier.into(), values };
        definition.validate()?;
        Ok(definition)
    }

    pub fn identifier(&self) -> &str { &self.identifier }
    pub fn values(&self) -> &[BlockState] { &self.values }
    pub fn default_value(&self) -> &BlockState { &self.values[0] }

    pub fn validate(&self) -> Result<(), String> {
        if self.identifier.trim().is_empty() {
            return Err("block-state identifier must not be empty".into());
        }
        if self.values.len() < 2 {
            return Err(format!("state {:?} must have at least two values", self.identifier));
        }
        let unique: BTreeSet<_> = self.values.iter().collect();
        if unique.len() != self.values.len() {
            return Err(format!("state {:?} contains duplicate values", self.identifier));
        }
        let kind = std::mem::discriminant(&self.values[0]);
        if self.values.iter().any(|value| std::mem::discriminant(value) != kind) {
            return Err(format!("state {:?} mixes value types", self.identifier));
        }
        Ok(())
    }

    pub fn value_index(&self, value: &BlockState) -> Option<usize> {
        self.values.iter().position(|candidate| candidate == value)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bool_default_order_is_preserved() {
        let state = BlockStateDefinition::boolean("open_bit", false);
        assert_eq!(state.values(), &[BlockState::Bool(false), BlockState::Bool(true)]);
    }

    #[test]
    fn integer_range_is_inclusive_and_normalized() {
        let state = BlockStateDefinition::integer("age", 3, 1).unwrap();
        assert_eq!(state.values(), &[1.into(), 2.into(), 3.into()]);
    }

    #[test]
    fn duplicate_enum_values_are_rejected() {
        assert!(BlockStateDefinition::enumeration("axis", ["x", "x"]).is_err());
    }
}
