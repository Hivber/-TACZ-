//! Block targeting ray traversal adapted from MC-RS Amanatides-Woo voxel DDA.
//! Upstream: pedrokarim/mc-rs, crates/mc-rs-server/src/raytrace.rs,
//! revision da93a9501601fcdc284762fde88c24b44604991e (MIT).
//!
//! Targetability is not entity collision. Plants and other non-solid blocks may
//! be selectable/minable while having no collision volume. The callback means
//! "has a selectable hit shape", never "is solid" or "blocks entity movement".
//! This is voxel-cell traversal; partial hit boxes require a shape test per cell.

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Vec3 { pub x: f64, pub y: f64, pub z: f64 }

impl Vec3 {
    pub const fn new(x: f64, y: f64, z: f64) -> Self { Self { x, y, z } }
    fn is_finite(self) -> bool { self.x.is_finite() && self.y.is_finite() && self.z.is_finite() }
    fn length(self) -> f64 { self.x.hypot(self.y).hypot(self.z) }
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum HitFace { Down, Up, North, South, West, East }

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct BlockHit { pub x: i32, pub y: i32, pub z: i32, pub face: HitFace, pub distance: f64 }

/// Return the first selectable voxel intersected by a normalized ray.
/// Do not pass a collision-solid predicate: that makes grass, flowers and petals untargetable.
pub fn cast_block_ray(
    origin: Vec3,
    direction: Vec3,
    max_distance: f64,
    is_selectable: impl Fn(i32, i32, i32) -> bool,
) -> Option<BlockHit> {
    if !origin.is_finite() || !direction.is_finite() || !max_distance.is_finite() || max_distance < 0.0 {
        return None;
    }
    let length = direction.length();
    if !length.is_finite() || length <= f64::EPSILON { return None; }
    let dir = Vec3::new(direction.x / length, direction.y / length, direction.z / length);

    let mut x = origin.x.floor() as i32;
    let mut y = origin.y.floor() as i32;
    let mut z = origin.z.floor() as i32;
    let step_x = dir.x.signum() as i32;
    let step_y = dir.y.signum() as i32;
    let step_z = dir.z.signum() as i32;
    let delta_x = if step_x == 0 { f64::INFINITY } else { 1.0 / dir.x.abs() };
    let delta_y = if step_y == 0 { f64::INFINITY } else { 1.0 / dir.y.abs() };
    let delta_z = if step_z == 0 { f64::INFINITY } else { 1.0 / dir.z.abs() };
    let boundary_x = if step_x > 0 { x as f64 + 1.0 } else { x as f64 };
    let boundary_y = if step_y > 0 { y as f64 + 1.0 } else { y as f64 };
    let boundary_z = if step_z > 0 { z as f64 + 1.0 } else { z as f64 };
    let mut max_x = if step_x == 0 { f64::INFINITY } else { (boundary_x - origin.x) / dir.x };
    let mut max_y = if step_y == 0 { f64::INFINITY } else { (boundary_y - origin.y) / dir.y };
    let mut max_z = if step_z == 0 { f64::INFINITY } else { (boundary_z - origin.z) / dir.z };

    let mut distance = 0.0;
    let mut entered_face = HitFace::Up;
    loop {
        if distance > max_distance { return None; }
        if is_selectable(x, y, z) {
            return Some(BlockHit { x, y, z, face: entered_face, distance });
        }
        // Deterministic tie handling advances one axis per iteration so cells
        // touched at exact voxel edges/corners are not skipped.
        if max_x <= max_y && max_x <= max_z {
            distance = max_x;
            if !distance.is_finite() { return None; }
            x = x.checked_add(step_x)?;
            max_x += delta_x;
            entered_face = if step_x > 0 { HitFace::West } else { HitFace::East };
        } else if max_y <= max_z {
            distance = max_y;
            if !distance.is_finite() { return None; }
            y = y.checked_add(step_y)?;
            max_y += delta_y;
            entered_face = if step_y > 0 { HitFace::Down } else { HitFace::Up };
        } else {
            distance = max_z;
            if !distance.is_finite() { return None; }
            z = z.checked_add(step_z)?;
            max_z += delta_z;
            entered_face = if step_z > 0 { HitFace::North } else { HitFace::South };
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn targets_non_solid_plant_when_selectable() {
        let hit = cast_block_ray(Vec3::new(0.5, 0.5, 0.5), Vec3::new(1.0, 0.0, 0.0), 5.0,
            |x, y, z| (x, y, z) == (2, 0, 0)).expect("selectable plant must be targetable");
        assert_eq!((hit.x, hit.y, hit.z), (2, 0, 0));
        assert_eq!(hit.face, HitFace::West);
        assert!((hit.distance - 1.5).abs() < 1e-9);
    }

    #[test]
    fn non_selectable_air_does_not_intercept_ray() {
        assert!(cast_block_ray(Vec3::new(0.5, 0.5, 0.5), Vec3::new(1.0, 0.0, 0.0), 3.0,
            |_, _, _| false).is_none());
    }

    #[test]
    fn normalizes_direction_and_respects_maximum_reach() {
        let hit = cast_block_ray(Vec3::new(0.5, 0.5, 0.5), Vec3::new(4.0, 0.0, 0.0), 2.0,
            |x, _, _| x == 2).unwrap();
        assert!((hit.distance - 1.5).abs() < 1e-9);
        assert!(cast_block_ray(Vec3::new(0.5, 0.5, 0.5), Vec3::new(4.0, 0.0, 0.0), 1.0,
            |x, _, _| x == 2).is_none());
    }

    #[test]
    fn rejects_invalid_rays() {
        assert!(cast_block_ray(Vec3::new(0.0, 0.0, 0.0), Vec3::new(0.0, 0.0, 0.0), 3.0, |_,_,_| true).is_none());
        assert!(cast_block_ray(Vec3::new(f64::NAN, 0.0, 0.0), Vec3::new(1.0, 0.0, 0.0), 3.0, |_,_,_| true).is_none());
        assert!(cast_block_ray(Vec3::new(0.0, 0.0, 0.0), Vec3::new(1.0, 0.0, 0.0), f64::INFINITY, |_,_,_| true).is_none());
    }

    #[test]
    fn reports_correct_face_for_negative_direction() {
        let hit = cast_block_ray(Vec3::new(2.5, 0.5, 0.5), Vec3::new(-1.0, 0.0, 0.0), 3.0,
            |x, _, _| x == 1).unwrap();
        assert_eq!(hit.face, HitFace::East);
        assert_eq!(hit.distance, 0.5);
    }
}
