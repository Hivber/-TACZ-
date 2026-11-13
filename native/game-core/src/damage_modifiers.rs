//! Damage mitigation stages adapted from MC-RS damage_modifiers.rs and armor.rs.
//! Inputs are semantic values only; no effect manager, inventory packet types,
//! or client/network dependencies are imported.
//! Upstream: https://github.com/pedrokarim/mc-rs @ da93a9501601fcdc284762fde88c24b44604991e
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).

use crate::armor::ArmorStats;
use crate::damage_sources::{DamageCause, DamageKind};

/// Protection-type enchantment EPF. Bedrock/PMMP source caps at 20 EPF.
pub fn protection_reduction(
    protection: u32,
    fire_protection: u32,
    blast_protection: u32,
    projectile_protection: u32,
    kind: DamageKind,
) -> f32 {
    let epf = match kind {
        DamageKind::All => protection,
        DamageKind::Fire => protection.saturating_add(fire_protection.saturating_mul(2)),
        DamageKind::Blast => protection.saturating_add(blast_protection.saturating_mul(2)),
        DamageKind::Projectile => protection.saturating_add(projectile_protection.saturating_mul(2)),
    };
    epf.min(20) as f32 / 25.0
}

/// MC-RS resistance implementation: amplifier/level N reduces damage by 20% * N.
pub fn resistance_reduction(level: u32) -> f32 {
    (level as f32 * 0.2).min(1.0)
}

/// Armor formula used by the selected MC-RS/PocketMine-MP port.
/// This is a source-derived baseline, not a verified BDS 26.60 parity claim.
pub fn armor_reduction(damage: f32, defense_points: u32, toughness: f32) -> f32 {
    if defense_points == 0 || damage <= 0.0 { return damage.max(0.0); }
    let defense = defense_points as f32;
    let reduction_points = (defense / 5.0)
        .max(defense - damage / (2.0 + toughness.max(0.0) / 4.0))
        .min(20.0);
    (damage * (1.0 - reduction_points / 25.0)).max(0.0)
}

pub fn damage_kind(cause: DamageCause) -> DamageKind {
    match cause {
        DamageCause::Fire | DamageCause::Lava => DamageKind::Fire,
        DamageCause::BlockExplosion | DamageCause::EntityExplosion => DamageKind::Blast,
        DamageCause::Projectile => DamageKind::Projectile,
        _ => DamageKind::All,
    }
}

/// Runs armor -> protection -> resistance in the same order as MC-RS.
/// Unblockable damage bypasses armor; unprotectable damage bypasses Protection.
pub fn compute_final_damage(
    base: f32,
    cause: DamageCause,
    armor: ArmorStats,
    resistance_level: u32,
) -> f32 {
    let after_armor = if cause.is_armor_blockable() {
        armor_reduction(base, armor.defense_points, armor.toughness)
    } else {
        base.max(0.0)
    };
    let protection = if cause.is_protectable() {
        protection_reduction(
            armor.enchantments.protection,
            armor.enchantments.fire_protection,
            armor.enchantments.blast_protection,
            armor.enchantments.projectile_protection,
            damage_kind(cause),
        )
    } else {
        0.0
    };
    let resistance = resistance_reduction(resistance_level);
    (after_armor * (1.0 - protection) * (1.0 - resistance)).max(0.0)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn protection_epf_is_capped_at_twenty() {
        assert_eq!(protection_reduction(30, 0, 0, 0, DamageKind::All), 0.8);
    }

    #[test]
    fn fire_protection_only_adds_its_bonus_for_fire_damage() {
        let fire = protection_reduction(0, 4, 0, 0, DamageKind::Fire);
        let generic = protection_reduction(0, 4, 0, 0, DamageKind::All);
        assert!(fire > generic);
    }

    #[test]
    fn armor_formula_matches_source_reference_case() {
        // Source example: 10 damage against 7 defense and no toughness => 9.2.
        assert!((armor_reduction(10.0, 7, 0.0) - 9.2).abs() < 0.001);
    }

    #[test]
    fn void_and_starvation_bypass_armor_and_protection() {
        let armor = ArmorStats {
            defense_points: 20,
            toughness: 8.0,
            enchantments: crate::armor::ProtectionEnchantments { protection: 20, ..Default::default() },
            ..Default::default()
        };
        assert_eq!(compute_final_damage(10.0, DamageCause::Void, armor, 0), 10.0);
        assert_eq!(compute_final_damage(10.0, DamageCause::Starvation, armor, 0), 10.0);
    }

    #[test]
    fn resistance_caps_at_full_reduction() {
        assert_eq!(resistance_reduction(5), 1.0);
        assert_eq!(resistance_reduction(8), 1.0);
    }
}
