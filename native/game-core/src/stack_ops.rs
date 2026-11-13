//! Selectively ported from MC-RS at da93a9501601fcdc284762fde88c24b44604991e.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).
//! ItemStack merge and split operations adapted from MC-RS item_stack_merge.rs.
//! Unlike the upstream tuple of numeric runtime IDs, this operates on the
//! local namespaced ItemStack and includes all metadata/user data in identity.

use crate::item_stack::ItemStack;

pub fn can_merge(a: &ItemStack, b: &ItemStack) -> bool {
    a.same_kind(b) && a.count < a.max_count()
}

pub fn merge_stacks(a: &mut ItemStack, b: &mut ItemStack) -> u16 {
    if !a.same_kind(b) { return 0; }
    let space = a.max_count().saturating_sub(a.count);
    let transfer = space.min(b.count);
    a.count += transfer;
    b.count -= transfer;
    transfer
}

pub fn split_half(stack: &mut ItemStack) -> Result<ItemStack, String> {
    let taken = stack.count - stack.count / 2;
    if taken == 0 { return Err("cannot split an empty stack".into()); }
    stack.count -= taken;
    ItemStack::new(stack.identifier.clone(), taken, stack.metadata, stack.user_data.clone())
}

#[cfg(test)]
mod tests {
    use super::*;
    fn stack(name: &str, count: u16, meta: u32, data: Vec<u8>) -> ItemStack {
        ItemStack::new(name, count, meta, data).unwrap()
    }
    #[test] fn merge_respects_item_identity_and_stack_limit() {
        let mut a=stack("minecraft:cobblestone",60,0,vec![]);
        let mut b=stack("minecraft:cobblestone",20,0,vec![]);
        assert_eq!(merge_stacks(&mut a,&mut b),4);
        assert_eq!((a.count,b.count),(64,16));
    }
    #[test] fn enchanted_or_differently_damaged_stacks_do_not_merge() {
        let mut a=stack("minecraft:enchanted_book",1,1,vec![1]);
        let mut b=stack("minecraft:enchanted_book",1,2,vec![1]);
        assert_eq!(merge_stacks(&mut a,&mut b),0);
        let mut c=stack("minecraft:enchanted_book",1,1,vec![2]);
        assert_eq!(merge_stacks(&mut a,&mut c),0);
    }
    #[test] fn split_half_preserves_metadata_and_user_data() {
        let mut original=stack("minecraft:diamond",5,3,vec![8,9]);
        let split=split_half(&mut original).unwrap();
        assert_eq!((original.count,split.count),(2,3));
        assert_eq!(split.metadata,3);
        assert_eq!(split.user_data,vec![8,9]);
    }
}
