//! Fixed-slot inventory operations independent of Bedrock packet formats.
use crate::item_stack::ItemStack;

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Inventory { slots: Vec<Option<ItemStack>> }

impl Inventory {
    pub fn new(slot_count: usize) -> Self { Self { slots: vec![None; slot_count] } }
    pub fn len(&self) -> usize { self.slots.len() }
    pub fn is_empty(&self) -> bool { self.slots.is_empty() }
    pub fn get(&self, slot: usize) -> Option<&ItemStack> {
        self.slots.get(slot).and_then(Option::as_ref)
    }
    pub fn set(&mut self, slot: usize, stack: Option<ItemStack>) -> Result<(), String> {
        let target = self.slots.get_mut(slot).ok_or_else(|| format!("slot {slot} out of range"))?;
        *target = stack;
        Ok(())
    }

    /// Merge compatible stacks first, then fill empty slots. Returns any remainder.
    pub fn insert(&mut self, mut incoming: ItemStack) -> Option<ItemStack> {
        for slot in &mut self.slots {
            let Some(existing) = slot.as_mut() else { continue };
            if !existing.same_kind(&incoming) { continue; }
            let moved = existing.max_count().saturating_sub(existing.count).min(incoming.count);
            existing.count += moved;
            incoming.count -= moved;
            if incoming.count == 0 { return None; }
        }
        for slot in &mut self.slots {
            if slot.is_some() { continue; }
            let moved = incoming.max_count().min(incoming.count);
            let mut placed = incoming.clone();
            placed.count = moved;
            *slot = Some(placed);
            incoming.count -= moved;
            if incoming.count == 0 { return None; }
        }
        Some(incoming)
    }

    pub fn remove(&mut self, slot: usize, count: u16) -> Result<Option<ItemStack>, String> {
        if count == 0 { return Err("remove count must be positive".into()); }
        let current = self.slots.get_mut(slot).ok_or_else(|| format!("slot {slot} out of range"))?;
        let Some(existing) = current.as_mut() else { return Ok(None) };
        let removed_count = count.min(existing.count);
        let mut removed = existing.clone();
        removed.count = removed_count;
        existing.count -= removed_count;
        if existing.count == 0 { *current = None; }
        Ok(Some(removed))
    }

    pub fn move_items(&mut self, source: usize, destination: usize, count: u16) -> Result<(), String> {
        if source == destination { return Ok(()); }
        if count == 0 { return Err("move count must be positive".into()); }
        if source >= self.slots.len() || destination >= self.slots.len() {
            return Err("source or destination slot out of range".into());
        }
        let source_stack = self.slots[source].as_ref()
            .ok_or_else(|| "source slot is empty".to_string())?;
        let moving = count.min(source_stack.count);
        match self.slots[destination].as_ref() {
            None => {
                let mut moved = source_stack.clone();
                moved.count = moving;
                self.slots[destination] = Some(moved);
                self.decrease_slot(source, moving);
            }
            Some(target) if target.same_kind(source_stack) => {
                let room = target.max_count().saturating_sub(target.count);
                if room == 0 { return Err("destination stack is full".into()); }
                let moved = moving.min(room);
                self.slots[destination].as_mut().unwrap().count += moved;
                self.decrease_slot(source, moved);
            }
            Some(_) => {
                if moving != source_stack.count {
                    return Err("cannot partially move into incompatible slot".into());
                }
                self.slots.swap(source, destination);
            }
        }
        Ok(())
    }

    fn decrease_slot(&mut self, slot: usize, count: u16) {
        if let Some(stack) = self.slots[slot].as_mut() {
            stack.count -= count;
            if stack.count == 0 { self.slots[slot] = None; }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    fn stack(name: &str, count: u16) -> ItemStack {
        ItemStack::new(name, count, 0, vec![]).unwrap()
    }
    #[test]
    fn insert_merges_then_uses_empty_slots() {
        let mut inventory = Inventory::new(2);
        inventory.set(0, Some(stack("minecraft:cobblestone", 60))).unwrap();
        assert!(inventory.insert(stack("minecraft:cobblestone", 10)).is_none());
        assert_eq!(inventory.get(0).unwrap().count, 64);
        assert_eq!(inventory.get(1).unwrap().count, 6);
    }
    #[test]
    fn full_inventory_returns_remainder_without_loss() {
        let mut inventory = Inventory::new(1);
        inventory.set(0, Some(stack("minecraft:cobblestone", 64))).unwrap();
        assert_eq!(inventory.insert(stack("minecraft:cobblestone", 10)).unwrap().count, 10);
        assert_eq!(inventory.get(0).unwrap().count, 64);
    }
    #[test]
    fn removal_splits_and_clears_stack() {
        let mut inventory = Inventory::new(1);
        inventory.set(0, Some(stack("minecraft:arrow", 12))).unwrap();
        assert_eq!(inventory.remove(0, 5).unwrap().unwrap().count, 5);
        assert_eq!(inventory.remove(0, 7).unwrap().unwrap().count, 7);
        assert!(inventory.get(0).is_none());
    }
    #[test]
    fn metadata_prevents_incompatible_merges() {
        let mut inventory = Inventory::new(2);
        inventory.set(0, Some(ItemStack::new("minecraft:enchanted_book", 1, 1, vec![]).unwrap())).unwrap();
        assert!(inventory.insert(ItemStack::new("minecraft:enchanted_book", 1, 2, vec![]).unwrap()).is_none());
        assert_eq!(inventory.get(1).unwrap().metadata, 2);
    }
}
