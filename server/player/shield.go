package player

import (
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

const shieldRaiseDelay = 5 * time.Second / 20

// blocksWithShield blocks a directional melee or projectile attack if the player
// has raised a shield by sneaking with it in the off-hand.
func (p *Player) blocksWithShield(blockedDamage, attackDamage float64, src world.DamageSource) bool {
	if blockedDamage <= 0 || !p.sneaking || p.sneakingSince.IsZero() || time.Since(p.sneakingSince) < shieldRaiseDelay {
		return false
	}

	mainHand, offHand := p.HeldItems()
	if _, ok := offHand.Item().(item.Shield); !ok {
		return false
	}

	origin, attacker, ok := shieldAttackOrigin(src)
	if !ok || !shieldCovers(p.Rotation(), p.Position(), origin) {
		return false
	}

	if damage := shieldDurabilityDamage(attackDamage); damage > 0 {
		p.SetHeldItems(mainHand, p.damageItem(offHand, damage))
	}

	p.tx.PlaySound(p.Position(), sound.Custom{Name: "item.shield.block", Volume: 0.7, Pitch: 1})
	if living, ok := attacker.(entity.Living); ok {
		living.KnockBack(p.Position(), 0.5, 0.25)
	}
	return true
}

// shieldDurabilityDamage returns the shield durability lost for a blocked hit.
// Bedrock shields only lose durability for attacks of at least 3 damage, with
// the attack damage rounded up.
func shieldDurabilityDamage(damage float64) int {
	if damage < 3 {
		return 0
	}
	return int(math.Ceil(damage))
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
