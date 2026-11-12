//! Protocol-independent item stack identity and count validation.
//! Inspired by Chorus crates/item/src/item_stack.rs; packet descriptors,
//! network IDs and protocol-specific user-data envelopes are intentionally excluded.
use crate::stack_size::max_stack_size;
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct ItemStack {
    pub identifier: String,
    pub count: u16,
    pub metadata: u32,
    pub user_data: Vec<u8>,
}

impl ItemStack {
    pub fn new(identifier: impl Into<String>, count: u16, metadata: u32,
        user_data: Vec<u8>) -> Result<Self, String> {
        let identifier = identifier.into();
        validate_identifier(&identifier)?;
        if count == 0 || count > max_stack_size(&identifier) {
            return Err(format!("invalid count {count} for item {identifier:?}"));
        }
        Ok(Self { identifier, count, metadata, user_data })
    }

    pub fn max_count(&self) -> u16 { max_stack_size(&self.identifier) }

    /// Validate values received over IPC; deserialization does not call new().
    pub fn is_valid(&self) -> bool {
        validate_identifier(&self.identifier).is_ok() && self.count > 0
            && self.count <= self.max_count() && self.user_data.len() <= 65_536
    }

    pub fn same_kind(&self, other: &Self) -> bool {
        self.identifier == other.identifier
            && self.metadata == other.metadata
            && self.user_data == other.user_data
    }
}

fn validate_identifier(identifier: &str) -> Result<(), String> {
    let Some((namespace, name)) = identifier.split_once(':') else {
        return Err(format!("item identifier {identifier:?} must be namespaced"));
    };
    if namespace.is_empty() || name.is_empty()
        || identifier.chars().any(char::is_whitespace)
        || identifier.matches(':').count() != 1 {
        return Err(format!("invalid item identifier {identifier:?}"));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn item_identity_includes_metadata_and_user_data() {
        let a = ItemStack::new("minecraft:enchanted_book", 1, 2, vec![1, 2]).unwrap();
        let b = ItemStack::new("minecraft:enchanted_book", 1, 2, vec![1, 2]).unwrap();
        let c = ItemStack::new("minecraft:enchanted_book", 1, 3, vec![1, 2]).unwrap();
        assert!(a.same_kind(&b));
        assert!(!a.same_kind(&c));
    }
    #[test]
    fn count_and_identifier_are_validated() {
        assert!(ItemStack::new("minecraft:diamond_sword", 2, 0, vec![]).is_err());
        assert!(ItemStack::new("minecraft:cobblestone", 65, 0, vec![]).is_err());
        assert!(ItemStack::new("diamond", 1, 0, vec![]).is_err());
    }
}
