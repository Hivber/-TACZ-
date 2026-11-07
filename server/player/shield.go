package player

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	shieldRaiseDelay              = 5 * time.Second / 20
	shieldTransitionDuration      = 2 * time.Second / 20
	shieldBlockFlagDuration       = 5 * time.Second / 20
	shieldAttackInterruptDuration = 6 * time.Second / 20
)

// ShieldBlocking reports whether the player's shield is currently raised.
func (p *Player) ShieldBlocking() bool { return p.shieldBlocking }

// ShieldTransitionBlocking reports whether the shield is transitioning between raised and lowered.
func (p *Player) ShieldTransitionBlocking() bool {
	return !p.shieldTransitionEnd.IsZero() && time.Now().Before(p.shieldTransitionEnd)
}

// ShieldBlocked reports whether the player's shield has recently blocked a hit.
func (p *Player) ShieldBlocked() bool {
	return !p.shieldBlockedEnd.IsZero() && time.Now().Before(p.shieldBlockedEnd)
}

// ShieldBlockedUsingDamagedShield reports whether the most recent shield block damaged the shield.
func (p *Player) ShieldBlockedUsingDamagedShield() bool {
	return !p.shieldDamagedEnd.IsZero() && time.Now().Before(p.shieldDamagedEnd)
}

// shieldInEitherHand returns the shield stack and whether it is in the off-hand.
func (p *Player) shieldInEitherHand() (item.Stack, bool, bool) {
	mainHand, offHand := p.HeldItems()
	if _, ok := offHand.Item().(item.Shield); ok {
		return offHand, true, true
	}
	if _, ok := mainHand.Item().(item.Shield); ok {
		return mainHand, false, true
	}
	return item.Stack{}, false, false
}

// updateShieldBlockingState synchronises the Bedrock blocking state with the player's
// crouch state, held shield and five-tick raise delay.
func (p *Player) updateShieldBlockingState() {
	_, _, hasShield := p.shieldInEitherHand()
	now := time.Now()
	active := hasShield && p.sneaking && !p.sneakingSince.IsZero() &&
		now.Sub(p.sneakingSince) >= shieldRaiseDelay &&
		now.After(p.shieldReenableAt) && p.GameMode().AllowsInteraction()
	if active == p.shieldBlocking {
		return
	}
	p.shieldBlocking = active
	p.shieldTransitionEnd = now.Add(shieldTransitionDuration)
	p.updateState()
}

// tickShieldState expires transient shield metadata flags and updates the blocking state.
func (p *Player) tickShieldState() {
	p.updateShieldBlockingState()
	now := time.Now()
	changed := false
	for _, until := range []*time.Time{&p.shieldTransitionEnd, &p.shieldBlockedEnd, &p.shieldDamagedEnd} {
		if !until.IsZero() && !now.Before(*until) {
			*until = time.Time{}
			changed = true
		}
	}
	if changed {
		p.updateState()
	}
}

// interruptShieldBlocking temporarily lowers the shield after the player attacks.
func (p *Player) interruptShieldBlocking() {
	if !p.sneaking {
		return
	}
	if _, _, hasShield := p.shieldInEitherHand(); !hasShield {
		return
	}
	p.shieldReenableAt = time.Now().Add(shieldAttackInterruptDuration)
	if p.shieldBlocking {
		p.shieldBlocking = false
		p.shieldTransitionEnd = time.Now().Add(shieldTransitionDuration)
		p.updateState()
	}
}

// blocksWithShield blocks a directional melee or projectile attack if the player has raised a shield.
func (p *Player) blocksWithShield(blockedDamage, attackDamage float64, src world.DamageSource) bool {
	if blockedDamage <= 0 || !src.ReducedByArmour() || !p.shieldBlocking {
		return false
	}
	shield, inOffHand, hasShield := p.shieldInEitherHand()
	if !hasShield {
		return false
	}
	origin, attacker, ok := shieldAttackOrigin(src)
	if !ok || !shieldCovers(p.Rotation(), p.Position(), origin) {
		return false
	}

	mainHand, offHand := p.HeldItems()
	beforeDurability := shield.Durability()
	if damage := shieldDurabilityDamage(attackDamage); damage > 0 {
		damaged := p.damageItem(shield, damage)
		if inOffHand {
			offHand = damaged
		} else {
			mainHand = damaged
		}
		p.SetHeldItems(mainHand, offHand)
		if damaged.Durability() < beforeDurability {
			p.shieldDamagedEnd = time.Now().Add(shieldBlockFlagDuration)
		}
	}
	now := time.Now()
	p.shieldBlockedEnd = now.Add(shieldBlockFlagDuration)
	p.tx.PlaySound(p.Position(), sound.Custom{Name: "item.shield.block", Volume: 0.7, Pitch: 1})
	if living, ok := attacker.(entity.Living); ok {
		living.KnockBack(p.Position(), 0.5, 0.25)
	}
	p.updateState()
	return true
}

// shieldDurabilityDamage returns the shield durability lost for a blocked hit.
// Bedrock shields only lose durability for attacks of at least 3 damage, with
// the attack damage rounded up.
func shieldDurabilityDamage(damage float64) int {
	if damage < 3 {
		return 0
	}
	return int(damage) + 1
}

// shieldAttackOrigin returns the position and entity responsible for an attack
// that a shield can block. Environmental and explosion damage are excluded.
func shieldAttackOrigin(src world.DamageSource) (mgl64.Vec3, world.Entity, bool) {
	switch s := src.(type) {
	case entity.AttackDamageSource:
		if s.Attacker != nil {
			return s.Attacker.Position(), s.Attacker, true
		}
	case entity.ProjectileDamageSource:
		if s.Projectile != nil {
			return s.Projectile.Position(), nil, true
		}
		if s.Owner != nil {
			return s.Owner.Position(), nil, true
		}
	}
	return mgl64.Vec3{}, nil, false
}

// shieldCovers reports whether sourcePosition is within the horizontal forward
// hemisphere of the player. Pitch is intentionally ignored.
func shieldCovers(rotation cube.Rotation, position, sourcePosition mgl64.Vec3) bool {
	facing := rotation.Vec3()
	facing[1] = 0
	toSource := sourcePosition.Sub(position)
	toSource[1] = 0
	return facing.LenSqr() > 0 && toSource.LenSqr() > 0 && facing.Dot(toSource) > 0
}
