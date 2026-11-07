package entity

import (
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

// Mob is the shared living-entity implementation used by the first set of
// server-side vanilla mobs. AI and health state live in the behaviour so they
// are preserved when the world saves and reloads an entity.
type Mob struct {
	*Ent
	kind mobKind
}

type mobKind struct {
	id       string
	maxHP    float64
	speed    float64
	damage   float64
	height   float64
	width    float64
	hostile  bool
	lootItem world.Item
}

var (
	PigType      = mobType{kind: mobKind{id: "minecraft:pig", maxHP: 10, speed: 0.055, width: 0.9, height: 0.9, lootItem: item.Porkchop{}}}
	CowType      = mobType{kind: mobKind{id: "minecraft:cow", maxHP: 10, speed: 0.05, width: 0.9, height: 1.4, lootItem: item.Beef{}}}
	SheepType    = mobType{kind: mobKind{id: "minecraft:sheep", maxHP: 8, speed: 0.055, width: 0.9, height: 1.3, lootItem: item.Mutton{}}}
	ChickenType  = mobType{kind: mobKind{id: "minecraft:chicken", maxHP: 4, speed: 0.06, width: 0.4, height: 0.7, lootItem: item.Chicken{}}}
	ZombieType   = mobType{kind: mobKind{id: "minecraft:zombie", maxHP: 20, speed: 0.075, damage: 3, width: 0.6, height: 1.95, hostile: true, lootItem: item.RottenFlesh{}}}
	SkeletonType = mobType{kind: mobKind{id: "minecraft:skeleton", maxHP: 20, speed: 0.065, damage: 2, width: 0.6, height: 1.95, hostile: true, lootItem: item.Bone{}}}
)

type mobType struct{ kind mobKind }

func (t mobType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	return &Mob{Ent: Open(tx, handle, data), kind: t.kind}
}

func (t mobType) EncodeEntity() string { return t.kind.id }

func (t mobType) BBox(world.Entity) cube.BBox {
	k := t.kind
	return cube.Box(-k.width/2, 0, -k.width/2, k.width/2, k.height, k.width/2)
}

func (t mobType) DecodeNBT(nbt map[string]any, data *world.EntityData) {
	b := newMobBehaviour(t.kind)
	if health, ok := nbt["Health"].(float32); ok && health > 0 {
		b.health = NewHealthManager(float64(health), t.kind.maxHP)
	}
	data.Data = b
}

func (t mobType) EncodeNBT(data *world.EntityData) map[string]any {
	b, _ := data.Data.(*mobBehaviour)
	if b == nil {
		return map[string]any{}
	}
	return map[string]any{"Health": float32(b.health.Health())}
}

var mobTypes = map[string]mobType{
	PigType.kind.id:      PigType,
	CowType.kind.id:      CowType,
	SheepType.kind.id:    SheepType,
	ChickenType.kind.id:  ChickenType,
	ZombieType.kind.id:   ZombieType,
	SkeletonType.kind.id: SkeletonType,
}

// NewMob creates a living mob handle for a vanilla entity identifier. The name
// may be passed with or without the minecraft: namespace.
func NewMob(opts world.EntitySpawnOpts, name string) *world.EntityHandle {
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	t, ok := mobTypes[name]
	if !ok {
		return nil
	}
	return opts.New(t, newMobBehaviour(t.kind))
}

type mobBehaviour struct {
	BaseBehaviour
	kind           mobKind
	health         *HealthManager
	effects        *EffectManager
	movement       *MovementComputer
	immuneUntil    time.Time
	lastDamage     float64
	attackCooldown int
	target         *world.EntityHandle
	wanderTarget   mgl64.Vec3
	wanderTicks    int
	dead           bool
}

func newMobBehaviour(kind mobKind) *mobBehaviour {
	return &mobBehaviour{
		BaseBehaviour: NewBaseBehaviour(),
		kind:          kind,
		health:        NewHealthManager(kind.maxHP, kind.maxHP),
		effects:       NewEffectManager(),
		movement: &MovementComputer{
			Gravity:           0.08,
			Drag:              0.02,
			DragBeforeGravity: true,
		},
	}
}


func (b *mobBehaviour) Apply(data *world.EntityData) {
	data.Data = b
}

func (b *mobBehaviour) Tick(e *Ent, tx *world.Tx) *Movement {
	movement := b.movement.TickMovement(e, e.data.Pos, e.data.Vel, e.data.Rot, tx)
	e.data.Pos, e.data.Vel = movement.pos, movement.vel
	return movement
}

// Tick advances mob effects, basic goal-based movement, and melee AI.
func (m *Mob) Tick(tx *world.Tx, current int64) {
	b := m.mobState()
	if b.dead {
		_ = m.Close()
		return
	}
	b.effects.Tick(m, tx)
	b.tickAI(m, tx, current)
	m.Ent.Tick(tx, current)
}

func (m *Mob) mobState() *mobBehaviour {
	return m.data.Data.(*mobBehaviour)
}

func (b *mobBehaviour) tickAI(m *Mob, tx *world.Tx, current int64) {
	if b.attackCooldown > 0 {
		b.attackCooldown--
	}
	if b.kind.hostile {
		if current%10 == 0 || b.target == nil {
			b.target = b.closestPlayer(m, tx)
		}
		if b.target == nil {
			m.data.Vel[0], m.data.Vel[2] = 0, 0
			return
		}
		target, ok := b.target.Entity(tx)
		if !ok {
			b.target = nil
			m.data.Vel[0], m.data.Vel[2] = 0, 0
			return
		}
		delta := target.Position().Sub(m.Position())
		delta[1] = 0
		distance := delta.Len()
		if distance <= 1.6 {
			m.data.Vel[0], m.data.Vel[2] = 0, 0
			if b.attackCooldown == 0 {
				if living, ok := target.(Living); ok {
					living.Hurt(b.kind.damage, AttackDamageSource{Attacker: m})
				}
				b.attackCooldown = 20
			}
			return
		}
		b.moveToward(m, delta)
		return
	}

	if current%40 == 0 || b.wanderTicks <= 0 {
		b.wanderTicks = 20 + rand.IntN(61)
		dx, dz := rand.IntN(13)-6, rand.IntN(13)-6
		b.wanderTarget = m.Position().Add(mgl64.Vec3{float64(dx), 0, float64(dz)})
	}
	if b.wanderTicks > 0 {
		b.wanderTicks--
	}
	delta := b.wanderTarget.Sub(m.Position())
	delta[1] = 0
	if delta.Len() < 0.7 {
		m.data.Vel[0], m.data.Vel[2] = 0, 0
		return
	}
	b.moveToward(m, delta)
}

func (b *mobBehaviour) closestPlayer(m *Mob, tx *world.Tx) *world.EntityHandle {
	var nearest *world.EntityHandle
	best := 20.0 * 20.0
	for candidate := range tx.Players() {
		if gm, ok := candidate.(interface{ GameMode() world.GameMode }); ok && !gm.GameMode().HasCollision() {
			continue
		}
		delta := candidate.Position().Sub(m.Position())
		delta[1] = 0
		if distance := delta.LenSqr(); distance < best {
			if _, living := candidate.(Living); !living {
				continue
			}
			best, nearest = distance, candidate.H()
		}
	}
	return nearest
}

func (b *mobBehaviour) moveToward(m *Mob, delta mgl64.Vec3) {
	delta[1] = 0
	if delta.LenSqr() == 0 {
		m.data.Vel[0], m.data.Vel[2] = 0, 0
		return
	}
	direction := delta.Normalize()
	m.data.Vel[0], m.data.Vel[2] = direction[0]*b.kind.speed, direction[2]*b.kind.speed
	m.data.Rot = cube.Rotation{mgl64.RadToDeg(math.Atan2(direction[0], direction[2])), 0}
}

func (m *Mob) Health() float64 { return m.mobState().health.Health() }
func (m *Mob) MaxHealth() float64 { return m.mobState().health.MaxHealth() }
func (m *Mob) SetMaxHealth(v float64) {
	m.mobState().health.SetMaxHealth(v)
	m.Ent.updateState()
}
func (m *Mob) Dead() bool { return m.mobState().dead || m.Health() <= 0 }

// Hurt applies melee, projectile and explosion damage to a mob.
func (m *Mob) Hurt(damage float64, src world.DamageSource) (float64, bool) {
	b := m.mobState()
	if b.dead || damage < 0 || (src.Fire() && hasMobEffect(b, effect.FireResistance)) {
		return 0, false
	}
	damage = max(damage, 0)
	if src.ReducedByResistance() {
		if resistance, ok := b.effects.Effect(effect.Resistance); ok {
			damage *= effect.Resistance.Multiplier(src, resistance.Level())
		}
	}
	immune := time.Now().Before(b.immuneUntil)
	if immune {
		damage -= b.lastDamage
		if damage <= 0 {
			return 0, false
		}
	}
	if damage <= 0 {
		return 0, false
	}
	b.immuneUntil, b.lastDamage = time.Now().Add(time.Second/2), damage
	b.health.AddHealth(-damage)
	for _, viewer := range m.tx.Viewers(m.Position()) {
		viewer.ViewEntityAction(m, HurtAction{})
	}
	m.Ent.updateState()
	if m.Dead() {
		m.die()
	}
	return damage, true
}

func hasMobEffect(b *mobBehaviour, typ effect.Type) bool {
	_, ok := b.effects.Effect(typ)
	return ok
}

func (m *Mob) die() {
	b := m.mobState()
	if b.dead {
		return
	}
	b.dead = true
	for _, viewer := range m.tx.Viewers(m.Position()) {
		viewer.ViewEntityAction(m, DeathAction{})
	}
	m.tx.PlaySound(m.Position(), sound.Custom{Name: b.kind.id + ".death", Volume: 1, Pitch: 1})
	if m.tx.World().GameRuleBool("domobloot", true) && b.kind.lootItem != nil {
		create := m.tx.World().EntityRegistry().Config().Item
		if create != nil {
			opts := world.EntitySpawnOpts{Position: m.Position().Add(mgl64.Vec3{0, 0.25})}
			m.tx.AddEntity(create(opts, item.NewStack(b.kind.lootItem, 1)))
		}
	}
	_ = m.Close()
}

func (m *Mob) Heal(amount float64, _ world.HealingSource) float64 {
	before := m.Health()
	if m.Dead() || amount <= 0 {
		return 0
	}
	m.mobState().health.AddHealth(amount)
	m.Ent.updateState()
	return m.Health() - before
}

func (m *Mob) KnockBack(src mgl64.Vec3, force, height float64) {
	if m.Dead() {
		return
	}
	delta := m.Position().Sub(src)
	delta[1] = 0
	if delta.LenSqr() == 0 {
		return
	}
	delta = delta.Normalize()
	velocity := m.Velocity()
	velocity[0], velocity[1], velocity[2] = delta[0]*force, height, delta[2]*force
	m.SetVelocity(velocity)
}

func (m *Mob) AddEffect(e effect.Effect) { m.mobState().effects.Add(e, m) }
func (m *Mob) RemoveEffect(e effect.Type) { m.mobState().effects.Remove(e, m) }
func (m *Mob) Effects() []effect.Effect { return m.mobState().effects.Effects() }
func (m *Mob) Speed() float64 { return m.mobState().kind.speed }
func (m *Mob) SetSpeed(v float64) { m.mobState().kind.speed = max(v, 0) }

// SpawnNaturalMobs is called by the world tick loop. It performs a bounded,
// loaded-chunk-only spawn attempt around each active player. The first pass
// covers common surface animals and basic night-hostile mobs.
func SpawnNaturalMobs(tx *world.Tx) {
	w := tx.World()
	if w.Dimension() != world.Overworld || !w.GameRuleBool("domobspawning", true) {
		return
	}
	players := make([]world.Entity, 0, 4)
	for player := range tx.Players() {
		if gm, ok := player.(interface{ GameMode() world.GameMode }); ok && !gm.GameMode().HasCollision() {
			continue
		}
		players = append(players, player)
	}
	if len(players) == 0 {
		return
	}

	animals, hostiles := 0, 0
	for e := range tx.Entities() {
		mob, ok := e.(*Mob)
		if !ok {
			continue
		}
		if mob.kind.hostile {
			hostiles++
		} else {
			animals++
		}
	}
	animalCap, hostileCap := min(24, len(players)*6), min(24, len(players)*6)
	timeOfDay := w.Time() % 24000
	night := timeOfDay >= 13000 && timeOfDay < 23000
	canSpawnHostile := night && w.Difficulty() != world.DifficultyPeaceful && hostiles < hostileCap

	for _, player := range players {
		if animals >= animalCap && !canSpawnHostile {
			return
		}
		if rand.IntN(3) == 0 {
			continue
		}
		pos := player.Position()
		distance := float64(12 + rand.IntN(17))
		angle := rand.Float64() * math.Pi * 2
		x := int(math.Floor(pos[0] + math.Cos(angle)*distance))
		z := int(math.Floor(pos[2] + math.Sin(angle)*distance))
		probe := cube.Pos{x, int(math.Floor(pos[1])), z}
		if _, loaded := tx.BlockLoaded(probe); !loaded {
			continue
		}
		y := tx.HighestLightBlocker(x, z)
		spawnPos := cube.Pos{x, y + 1, z}
		if spawnPos.OutOfBounds(tx.Range()) {
			continue
		}
		ground, loaded := tx.BlockLoaded(cube.Pos{x, y, z})
		if !loaded || len(ground.Model().BBox(cube.Pos{x, y, z}, tx)) == 0 {
			continue
		}
		if _, loaded := tx.BlockLoaded(spawnPos); !loaded {
			continue
		}
		if _, air := tx.Block(spawnPos).(block.Air); !air {
			continue
		}
		head := spawnPos.Side(cube.FaceUp)
		if _, air := tx.Block(head).(block.Air); !air {
			continue
		}
		light := tx.Light(spawnPos)

		kind := ""
		if canSpawnHostile && hostiles < hostileCap && light <= 7 {
			if rand.IntN(3) == 0 {
				kind = "minecraft:skeleton"
			} else {
				kind = "minecraft:zombie"
			}
			hostiles++
		} else if animals < animalCap && light >= 9 {
			if _, grass := ground.(block.Grass); !grass {
				continue
			}
			switch rand.IntN(4) {
			case 0:
				kind = "minecraft:pig"
			case 1:
				kind = "minecraft:cow"
			case 2:
				kind = "minecraft:sheep"
			default:
				kind = "minecraft:chicken"
			}
			animals++
		}
		if kind == "" {
			continue
		}
		if create := w.EntityRegistry().Config().Mob; create != nil {
			opts := world.EntitySpawnOpts{Position: mgl64.Vec3{float64(x) + 0.5, float64(y + 1), float64(z) + 0.5}}
			tx.AddEntity(create(opts, kind))
		}
	}
}
