//! Imported from MC-RS at da93a9501601fcdc284762fde88c24b44604991e.
//! Original path: crates/mc-rs-server/src/world_border.rs
//! License: MIT; see third_party/licenses/MC-RS-MIT.txt.
//! Transport-independent gameplay logic only; no MC-RS networking or packet code.

//! World border — port conceptuel. PMMP n'a pas de world border natif mais
//! Bedrock supporte via le `SetSpawnPosition` + server-side clamping. Ici on
//! modélise une border carrée autour d'un centre, applicable au déplacement.

#[derive(Debug, Clone)]
pub struct WorldBorder {
    pub center_x: f64,
    pub center_z: f64,
    pub radius: f64,
    /// Dégât par seconde quand le joueur est hors-border.
    pub damage_per_second: f32,
    /// Tolerance : distance en dehors à partir de laquelle on applique damage.
    pub damage_buffer: f64,
}

impl Default for WorldBorder {
    fn default() -> Self {
        Self {
            center_x: 0.0,
            center_z: 0.0,
            radius: 29_999_984.0, // vanilla default ≈ 60M blocks total
            damage_per_second: 1.0,
            damage_buffer: 5.0,
        }
    }
}

impl WorldBorder {
    /// Returns false for malformed border settings or non-finite coordinates.
    pub fn is_valid(&self) -> bool {
        self.center_x.is_finite()
            && self.center_z.is_finite()
            && self.radius.is_finite()
            && self.radius >= 0.0
            && self.damage_per_second.is_finite()
            && self.damage_per_second >= 0.0
            && self.damage_buffer.is_finite()
            && self.damage_buffer >= 0.0
    }

    pub fn contains(&self, x: f64, z: f64) -> bool {
        if !self.is_valid() || !x.is_finite() || !z.is_finite() {
            return false;
        }
        let dx = (x - self.center_x).abs();
        let dz = (z - self.center_z).abs();
        dx <= self.radius && dz <= self.radius
    }

    /// Distance hors-border (0 si dans la border).
    pub fn distance_outside(&self, x: f64, z: f64) -> f64 {
        if !self.is_valid() || !x.is_finite() || !z.is_finite() {
            return f64::INFINITY;
        }
        let dx = (x - self.center_x).abs();
        let dz = (z - self.center_z).abs();
        let ox = (dx - self.radius).max(0.0);
        let oz = (dz - self.radius).max(0.0);
        (ox * ox + oz * oz).sqrt()
    }

    /// Retourne Some(damage) si le joueur doit prendre damage ce tick (20 TPS).
    pub fn damage_for_position(&self, x: f64, z: f64) -> Option<f32> {
        if !self.is_valid() {
            return None;
        }
        let dist = self.distance_outside(x, z);
        if dist > self.damage_buffer {
            Some(self.damage_per_second / 20.0)
        } else {
            None
        }
    }

    /// Clamp position to border edge (teleport back inside).
    pub fn clamp(&self, x: f64, z: f64) -> (f64, f64) {
        if !self.is_valid() || !x.is_finite() || !z.is_finite() {
            return (x, z);
        }
        let cx = x.clamp(self.center_x - self.radius, self.center_x + self.radius);
        let cz = z.clamp(self.center_z - self.radius, self.center_z + self.radius);
        (cx, cz)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn contains_inside() {
        let wb = WorldBorder {
            center_x: 0.0,
            center_z: 0.0,
            radius: 100.0,
            ..Default::default()
        };
        assert!(wb.contains(50.0, 50.0));
        assert!(!wb.contains(150.0, 0.0));
    }

    #[test]
    fn damage_outside_buffer() {
        let wb = WorldBorder {
            center_x: 0.0,
            center_z: 0.0,
            radius: 100.0,
            damage_per_second: 2.0,
            damage_buffer: 5.0,
        };
        // Just outside — no damage yet.
        assert!(wb.damage_for_position(102.0, 0.0).is_none());
        // Way outside — damage.
        assert!(wb.damage_for_position(200.0, 0.0).is_some());
    }

    #[test]
    fn clamp_to_edge() {
        let wb = WorldBorder {
            center_x: 0.0,
            center_z: 0.0,
            radius: 100.0,
            ..Default::default()
        };
        assert_eq!(wb.clamp(150.0, -200.0), (100.0, -100.0));
    }

    #[test]
    fn invalid_border_configuration_is_rejected_safely() {
        let wb = WorldBorder { radius: f64::NAN, ..Default::default() };
        assert!(!wb.contains(0.0, 0.0));
        assert_eq!(wb.damage_for_position(1000.0, 1000.0), None);
        assert_eq!(wb.clamp(1.0, 2.0), (1.0, 2.0));
    }
}
