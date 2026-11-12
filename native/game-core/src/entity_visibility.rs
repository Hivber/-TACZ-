//! Per-player entity visibility culling adapted from MC-RS.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! Revision: da93a9501601fcdc284762fde88c24b44604991e
//! Original: crates/mc-rs-server/src/entity_culling.rs
//! License: MIT; see ../../third_party/licenses/MC-RS-MIT.txt.
//!
//! This module is transport-independent: it decides visibility transitions but
//! does not create or send Bedrock packets. Dragonfly remains the packet adapter.

use std::collections::{HashMap, HashSet};

/// Squared horizontal distance. Y is intentionally ignored to preserve the
/// upstream server's horizontal entity-view semantics.
pub fn distance_squared_xz(a: [f32; 3], b: [f32; 3]) -> f32 {
    let dx = a[0] - b[0];
    let dz = a[2] - b[2];
    dx * dx + dz * dz
}

/// Entity visibility radius in blocks, based on the viewer's chunk distance.
/// A four-chunk floor prevents extremely small client view distances from
/// hiding every nearby entity.
pub fn entity_view_radius_blocks(view_distance_chunks: i32) -> f32 {
    (view_distance_chunks.max(4) * 16) as f32
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum VisibilityTransition {
    StillHidden,
    StillVisible,
    JustEntered,
    JustLeft,
}

/// Classifies one entity's visibility and updates the viewer's visible set.
pub fn classify_transition(
    visible_entities: &mut HashSet<i64>,
    entity_id: i64,
    entity_position: [f32; 3],
    viewer_position: [f32; 3],
    view_distance_chunks: i32,
) -> VisibilityTransition {
    if !entity_position.iter().chain(viewer_position.iter()).all(|v| v.is_finite()) {
        return if visible_entities.remove(&entity_id) {
            VisibilityTransition::JustLeft
        } else {
            VisibilityTransition::StillHidden
        };
    }

    let radius = entity_view_radius_blocks(view_distance_chunks);
    let in_view = distance_squared_xz(viewer_position, entity_position) <= radius * radius;
    let was_visible = visible_entities.contains(&entity_id);

    match (was_visible, in_view) {
        (false, false) => VisibilityTransition::StillHidden,
        (true, true) => VisibilityTransition::StillVisible,
        (false, true) => {
            visible_entities.insert(entity_id);
            VisibilityTransition::JustEntered
        }
        (true, false) => {
            visible_entities.remove(&entity_id);
            VisibilityTransition::JustLeft
        }
    }
}

/// Maintains visibility sets independently for each connected viewer.
#[derive(Default)]
pub struct EntityVisibility {
    visible_by_viewer: HashMap<String, HashSet<i64>>,
}

impl EntityVisibility {
    pub fn update(
        &mut self,
        viewer_id: &str,
        entity_id: i64,
        entity_position: [f32; 3],
        viewer_position: [f32; 3],
        view_distance_chunks: i32,
    ) -> VisibilityTransition {
        classify_transition(
            self.visible_by_viewer.entry(viewer_id.to_owned()).or_default(),
            entity_id,
            entity_position,
            viewer_position,
            view_distance_chunks,
        )
    }

    pub fn is_visible(&self, viewer_id: &str, entity_id: i64) -> bool {
        self.visible_by_viewer
            .get(viewer_id)
            .is_some_and(|entities| entities.contains(&entity_id))
    }

    pub fn remove_entity(&mut self, entity_id: i64) {
        for entities in self.visible_by_viewer.values_mut() {
            entities.remove(&entity_id);
        }
    }

    pub fn remove_viewer(&mut self, viewer_id: &str) {
        self.visible_by_viewer.remove(viewer_id);
    }

    pub fn visible_count(&self, viewer_id: &str) -> usize {
        self.visible_by_viewer.get(viewer_id).map_or(0, HashSet::len)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn horizontal_distance_ignores_vertical_difference() {
        assert_eq!(distance_squared_xz([0.0, 100.0, 0.0], [3.0, -50.0, 4.0]), 25.0);
    }

    #[test]
    fn view_radius_has_four_chunk_minimum() {
        assert_eq!(entity_view_radius_blocks(1), 64.0);
        assert_eq!(entity_view_radius_blocks(8), 128.0);
    }

    #[test]
    fn all_visibility_transitions_are_classified() {
        let mut visible = HashSet::new();
        let viewer = [0.0, 0.0, 0.0];
        assert_eq!(classify_transition(&mut visible, 7, [1000.0, 0.0, 0.0], viewer, 4), VisibilityTransition::StillHidden);
        assert_eq!(classify_transition(&mut visible, 7, [10.0, 0.0, 0.0], viewer, 4), VisibilityTransition::JustEntered);
        assert_eq!(classify_transition(&mut visible, 7, [12.0, 0.0, 0.0], viewer, 4), VisibilityTransition::StillVisible);
        assert_eq!(classify_transition(&mut visible, 7, [1000.0, 0.0, 0.0], viewer, 4), VisibilityTransition::JustLeft);
    }

    #[test]
    fn viewers_have_independent_visibility_sets_and_cleanup() {
        let mut visibility = EntityVisibility::default();
        assert_eq!(visibility.update("a", 1, [1.0, 0.0, 1.0], [0.0; 3], 8), VisibilityTransition::JustEntered);
        assert!(!visibility.is_visible("b", 1));
        assert_eq!(visibility.visible_count("a"), 1);
        visibility.remove_entity(1);
        assert!(!visibility.is_visible("a", 1));
        visibility.update("a", 2, [1.0, 0.0, 1.0], [0.0; 3], 8);
        visibility.remove_viewer("a");
        assert_eq!(visibility.visible_count("a"), 0);
    }

    #[test]
    fn invalid_positions_cannot_leave_entities_stuck_visible() {
        let mut visible = HashSet::from([9]);
        assert_eq!(classify_transition(&mut visible, 9, [f32::NAN, 0.0, 0.0], [0.0; 3], 8), VisibilityTransition::JustLeft);
        assert!(!visible.contains(&9));
    }
}
