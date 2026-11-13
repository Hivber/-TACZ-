//! Pure combat and health-state resolution adapted from MC-RS combat.rs.
//! Event callbacks, packet dispatch and entity registries remain outside this
//! module. The Go/Dragonfly adapter must call this only with semantic combat
//! inputs and then apply the returned outcome to the live entity.
//! Upstream: https://github.com/pedrokarim/mc-rs @ da93a9501601fcdc284762fde88c24b44604991e
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).

use crate::armor::ArmorStats;
use crate::damage_modifiers::compute_final_damage;
use crate::damage_sources::DamageCause;

pub const DEFAULT_NO_DAMAGE_TICKS: u16 = 10;
pub const DEFAULT_SPAWN_INVULNERABILITY_TICKS: u16 = 20;
pub const DEFAULT_KNOCKBACK_FORCE: f32 = 0.4;
pub const DEFAULT_KNOCKBACK_VERTICAL: f32 = 0.4;

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct HealthState {
    pub current: f32,
    pub maximum: f32,
    pub absorption: f32,
}

impl HealthState {
    pub fn new(maximum: f32) -> Result<Self, String> {
        if !maximum.is_finite() || maximum <= 0.0 {
            return Err("maximum health must be finite and positive".into());
        }
        Ok(Self { current: maximum, maximum, absorption: 0.0 })
    }

    pub fn set_current(&mut self, value: f32) -> Result<(), String> {
        if !value.is_finite() { return Err("health must be finite".into()); }
        self.current = value.clamp(0.0, self.maximum);
        Ok(())
    }

    pub fn set_absorption(&mut self, value: f32) -> Result<(), String> {
        if !value.is_finite() || value < 0.0 {
            return Err("absorption must be finite and non-negative".into());
        }
        self.absorption = value;
        Ok(())
    }

    pub fn heal(&mut self, amount: f32) -> Result<f32, String> {
        if !amount.is_finite() || amount < 0.0 {
            return Err("healing amount must be finite and non-negative".into());
        }
        let before = self.current;
        self.current = (self.current + amount).min(self.maximum);
        Ok(self.current - before)
    }

    pub fn is_dead(&self) -> bool { self.current <= 0.0 }
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct CombatState {
    /// Spawn/respawn invulnerability, decremented independently of hurt-time.
    pub spawn_invulnerability_ticks: u16,
    /// Hurt-time window during which equal/weaker subsequent hits are ignored.
    pub hurt_time_ticks: u16,
    /// Raw base damage of the previous successful hit, used for stronger-hit delta.
    pub last_damage_base: Option<f32>,
}

impl CombatState {
    pub fn new() -> Self {
        Self {
            spawn_invulnerability_ticks: DEFAULT_SPAWN_INVULNERABILITY_TICKS,
            hurt_time_ticks: 0,
            last_damage_base: None,
        }
    }

    pub fn tick(&mut self) {
        self.spawn_invulnerability_ticks = self.spawn_invulnerability_ticks.saturating_sub(1);
        self.hurt_time_ticks = self.hurt_time_ticks.saturating_sub(1);
        if self.hurt_time_ticks == 0 { self.last_damage_base = None; }
    }

    pub fn clear_spawn_protection(&mut self) {
        self.spawn_invulnerability_ticks = 0;
    }
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DamageRequest {
    pub cause: DamageCause,
    pub base_damage: f32,
    pub resistance_level: u32,
    pub knockback_force: f32,
    pub target_position: [f32; 3],
    pub attacker_position: Option<[f32; 3]>,
}

#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct DamageOutcome {
    pub cancelled: bool,
    pub base_damage: f32,
    pub mitigated_damage: f32,
    pub absorbed_damage: f32,
    pub health_damage: f32,
    pub remaining_health: f32,
    pub remaining_absorption: f32,
    pub died: bool,
    /// Velocity delta (x, y, z), to be applied by the host entity physics system.
    pub knockback: Option<[f32; 3]>,
}

fn calculate_knockback(
    cause: DamageCause,
    target: [f32; 3],
    attacker: Option<[f32; 3]>,
    force: f32,
    resistance: f32,
) -> Option<[f32; 3]> {
    if !cause.causes_knockback() { return None; }
    let attacker = attacker?;
    if target.iter().chain(attacker.iter()).any(|v| !v.is_finite()) { return None; }
    if !force.is_finite() || force <= 0.0 { return None; }
    let dx = target[0] - attacker[0];
    let dz = target[2] - attacker[2];
    let magnitude = (dx * dx + dz * dz).sqrt();
    if magnitude <= f32::EPSILON { return None; }
    let effective_force = force * (1.0 - resistance.clamp(0.0, 1.0));
    Some([
        dx / magnitude * effective_force,
        DEFAULT_KNOCKBACK_VERTICAL,
        dz / magnitude * effective_force,
    ])
}

/// Apply one semantic damage request. Invalid numeric input is rejected before
/// mutating health or timers. Armor/protection/resistance are resolved from the
/// target's current equipment/effect snapshot.
pub fn apply_damage(
    health: &mut HealthState,
    combat: &mut CombatState,
    armor: ArmorStats,
    request: DamageRequest,
) -> Result<DamageOutcome, String> {
    if !request.base_damage.is_finite() || request.base_damage < 0.0 {
        return Err("base damage must be finite and non-negative".into());
    }
    if !request.knockback_force.is_finite() || request.knockback_force < 0.0 {
        return Err("knockback force must be finite and non-negative".into());
    }
    if request.target_position.iter().any(|v| !v.is_finite())
        || request.attacker_position.is_some_and(|p| p.iter().any(|v| !v.is_finite())) {
        return Err("combat positions must be finite".into());
    }
    if health.is_dead() || request.base_damage == 0.0 {
        return Ok(DamageOutcome {
            cancelled: true,
            remaining_health: health.current,
            remaining_absorption: health.absorption,
            died: health.is_dead(),
            ..Default::default()
        });
    }

    if combat.spawn_invulnerability_ticks > 0 && request.cause != DamageCause::Suicide {
        return Ok(DamageOutcome {
            cancelled: true,
            base_damage: request.base_damage,
            remaining_health: health.current,
            remaining_absorption: health.absorption,
            ..Default::default()
        });
    }

    let mut effective_base = request.base_damage;
    if combat.hurt_time_ticks > 0 {
        let previous = combat.last_damage_base.unwrap_or(0.0);
        if previous >= effective_base {
            return Ok(DamageOutcome {
                cancelled: true,
                base_damage: request.base_damage,
                remaining_health: health.current,
                remaining_absorption: health.absorption,
                ..Default::default()
            });
        }
        effective_base -= previous;
    }
    if effective_base <= 0.0 {
        return Ok(DamageOutcome {
            cancelled: true,
            base_damage: request.base_damage,
            remaining_health: health.current,
            remaining_absorption: health.absorption,
            ..Default::default()
        });
    }

    let mitigated = compute_final_damage(effective_base, request.cause, armor, request.resistance_level);
    let absorbed = mitigated.min(health.absorption);
    health.absorption = (health.absorption - absorbed).max(0.0);
    let health_damage = (mitigated - absorbed).max(0.0);
    health.current = (health.current - health_damage).max(0.0);

    combat.hurt_time_ticks = DEFAULT_NO_DAMAGE_TICKS;
    combat.last_damage_base = Some(request.base_damage);

    let knockback = calculate_knockback(
        request.cause,
        request.target_position,
        request.attacker_position,
        request.knockback_force,
        armor.knockback_resistance,
    );

    Ok(DamageOutcome {
        cancelled: false,
        base_damage: request.base_damage,
        mitigated_damage: mitigated,
        absorbed_damage: absorbed,
        health_damage,
        remaining_health: health.current,
        remaining_absorption: health.absorption,
        died: health.is_dead(),
        knockback,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::armor::{ArmorLoadout, ArmorPiece, ProtectionEnchantments};
    use crate::armor_tier::{ArmorSlot, ArmorTier};

    fn request(base_damage: f32) -> DamageRequest {
        DamageRequest {
            cause: DamageCause::Entity,
            base_damage,
            resistance_level: 0,
            knockback_force: DEFAULT_KNOCKBACK_FORCE,
            target_position: [0.0, 64.0, 0.0],
            attacker_position: Some([1.0, 64.0, 0.0]),
        }
    }

    #[test]
    fn spawn_invulnerability_blocks_normal_damage_but_not_suicide() {
        let mut hp = HealthState::new(20.0).unwrap();
        let mut state = CombatState::new();
        let outcome = apply_damage(&mut hp, &mut state, ArmorStats::default(), request(5.0)).unwrap();
        assert!(outcome.cancelled);
        assert_eq!(hp.current, 20.0);

        let mut suicide = request(25.0);
        suicide.cause = DamageCause::Suicide;
        let outcome = apply_damage(&mut hp, &mut state, ArmorStats::default(), suicide).unwrap();
        assert!(outcome.died);
        assert_eq!(hp.current, 0.0);
    }

    #[test]
    fn hurt_time_ignores_weaker_hits_and_applies_only_stronger_hit_delta() {
        let mut hp = HealthState::new(20.0).unwrap();
        let mut state = CombatState::new();
        state.clear_spawn_protection();
        assert_eq!(apply_damage(&mut hp, &mut state, ArmorStats::default(), request(5.0)).unwrap().health_damage, 5.0);
        let weaker = apply_damage(&mut hp, &mut state, ArmorStats::default(), request(3.0)).unwrap();
        assert!(weaker.cancelled);
        assert_eq!(hp.current, 15.0);
        let stronger = apply_damage(&mut hp, &mut state, ArmorStats::default(), request(8.0)).unwrap();
        assert_eq!(stronger.health_damage, 3.0);
        assert_eq!(hp.current, 12.0);
    }

    #[test]
    fn armor_protection_resistance_and_absorption_are_applied_in_order() {
        let mut loadout = ArmorLoadout::empty();
        loadout.equip(ArmorPiece {
            material: ArmorTier::Iron,
            slot: ArmorSlot::Chestplate,
            enchantments: ProtectionEnchantments { protection: 2, ..Default::default() },
        }).unwrap();
        let stats = loadout.stats();
        let mut hp = HealthState::new(20.0).unwrap();
        hp.set_absorption(2.0).unwrap();
        let mut state = CombatState::new();
        state.clear_spawn_protection();
        let mut input = request(10.0);
        input.resistance_level = 1;
        let outcome = apply_damage(&mut hp, &mut state, stats, input).unwrap();
        assert!(outcome.mitigated_damage < 10.0);
        assert_eq!(outcome.absorbed_damage, 2.0);
        assert!(outcome.health_damage > 0.0);
        assert_eq!(hp.absorption, 0.0);
    }

    #[test]
    fn knockback_respects_armor_and_damage_kind() {
        let mut loadout = ArmorLoadout::empty();
        loadout.equip(ArmorPiece {
            material: ArmorTier::Netherite,
            slot: ArmorSlot::Boots,
            enchantments: Default::default(),
        }).unwrap();
        let mut hp = HealthState::new(20.0).unwrap();
        let mut state = CombatState::new();
        state.clear_spawn_protection();
        let outcome = apply_damage(&mut hp, &mut state, loadout.stats(), request(1.0)).unwrap();
        let velocity = outcome.knockback.unwrap();
        assert!((velocity[0] + 0.36).abs() < 0.001);
        assert!((velocity[1] - 0.4).abs() < f32::EPSILON);
    }

    #[test]
    fn non_knockback_causes_do_not_produce_velocity() {
        let mut hp = HealthState::new(20.0).unwrap();
        let mut state = CombatState::new();
        state.clear_spawn_protection();
        let mut input = request(2.0);
        input.cause = DamageCause::Fall;
        assert!(apply_damage(&mut hp, &mut state, ArmorStats::default(), input).unwrap().knockback.is_none());
    }

    #[test]
    fn invalid_inputs_do_not_mutate_combat_or_health() {
        let mut hp = HealthState::new(20.0).unwrap();
        let mut state = CombatState::new();
        let before = state.clone();
        let mut input = request(f32::NAN);
        assert!(apply_damage(&mut hp, &mut state, ArmorStats::default(), input).is_err());
        assert_eq!(hp.current, 20.0);
        assert_eq!(state, before);
        input = request(4.0);
        input.target_position[1] = f32::INFINITY;
        assert!(apply_damage(&mut hp, &mut state, ArmorStats::default(), input).is_err());
        assert_eq!(state, before);
    }
}
