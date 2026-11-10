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
	"github.com/df-mc/dragonfly/server/item/potion"
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

type mobSpawnCategory uint8

const (
	mobSpawnNone mobSpawnCategory = iota
	mobSpawnAnimal
	mobSpawnMonster
)

type mobKind struct {
	id       string
	maxHP    float64
	speed    float64
	damage   float64
	height   float64
	width    float64
	hostile  bool
	ranged   bool
	sunBurn  bool
	lootItem world.Item

	spawnCategory mobSpawnCategory
	spawnWeight   int
	groupMin      int
	groupMax      int
	minLight      uint8
	maxLight      uint8
	biomeTag      string
	grassRequired bool
}

var (
	PigType = mobType{kind: mobKind{
		id: "minecraft:pig", maxHP: 10, speed: 0.055, width: 0.9, height: 0.9,
		lootItem: item.Porkchop{}, spawnCategory: mobSpawnAnimal, spawnWeight: 10,
		groupMin: 1, groupMax: 3, minLight: 7, maxLight: 15, biomeTag: "animal", grassRequired: true,
	}}
	CowType = mobType{kind: mobKind{
		id: "minecraft:cow", maxHP: 10, speed: 0.05, width: 0.9, height: 1.4,
		lootItem: item.Beef{}, spawnCategory: mobSpawnAnimal, spawnWeight: 8,
		groupMin: 2, groupMax: 4, minLight: 7, maxLight: 15, biomeTag: "animal", grassRequired: true,
	}}
	SheepType = mobType{kind: mobKind{
		id: "minecraft:sheep", maxHP: 8, speed: 0.055, width: 0.9, height: 1.3,
		lootItem: item.Mutton{}, spawnCategory: mobSpawnAnimal, spawnWeight: 12,
		groupMin: 2, groupMax: 4, minLight: 7, maxLight: 15, biomeTag: "animal", grassRequired: true,
	}}
	ChickenType = mobType{kind: mobKind{
		id: "minecraft:chicken", maxHP: 4, speed: 0.06, width: 0.4, height: 0.7,
		lootItem: item.Chicken{}, spawnCategory: mobSpawnAnimal, spawnWeight: 10,
		groupMin: 2, groupMax: 4, minLight: 7, maxLight: 15, biomeTag: "animal", grassRequired: true,
	}}
	ZombieType = mobType{kind: mobKind{
		id: "minecraft:zombie", maxHP: 20, speed: 0.075, damage: 3, width: 0.6, height: 1.95,
		hostile: true, sunBurn: true, lootItem: item.RottenFlesh{},
		spawnCategory: mobSpawnMonster, spawnWeight: 95, groupMin: 2, groupMax: 4,
		minLight: 0, maxLight: 7, biomeTag: "monster",
	}}
	SkeletonType = mobType{kind: mobKind{
		id: "minecraft:skeleton", maxHP: 20, speed: 0.065, damage: 2, width: 0.6, height: 1.95,
		hostile: true, ranged: true, sunBurn: true, lootItem: item.Bone{},
		spawnCategory: mobSpawnMonster, spawnWeight: 100, groupMin: 1, groupMax: 2,
		minLight: 0, maxLight: 7, biomeTag: "monster",
	}}
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
	b := newMobBehaviour(t.kind, false)
	if health, ok := nbt["Health"].(float32); ok {
		b.health = NewHealthManager(float64(health), t.kind.maxHP)
	}
	if natural, ok := nbt["NaturalSpawn"].(bool); ok {
		b.naturalSpawn = natural
	}
	if dead, ok := nbt["Dead"].(bool); ok {
		b.dead = dead
	}
	if ticks, ok := nbt["DeathTicks"].(int32); ok {
		b.deathTicks = int(ticks)
	}
	if ticks, ok := nbt["DespawnTicks"].(int32); ok {
		b.despawnTicks = min(max(int(ticks), 0), mobDespawnInactiveTicks)
	}
	if b.health.Health() <= 0 {
		b.dead = true
	}
	if b.dead && b.deathTicks <= 0 {
		b.deathTicks = mobDeathAnimationTicks
	}
	data.Data = b
}

func (t mobType) EncodeNBT(data *world.EntityData) map[string]any {
	b, _ := data.Data.(*mobBehaviour)
	if b == nil {
		return map[string]any{}
	}
	return map[string]any{
		"Health":       float32(b.health.Health()),
		"NaturalSpawn": b.naturalSpawn,
		"Dead":         b.dead,
		"DeathTicks":   int32(b.deathTicks),
		"DespawnTicks": int32(b.despawnTicks),
	}
}

var mobTypes = map[string]mobType{
	PigType.kind.id:      PigType,
	CowType.kind.id:      CowType,
	SheepType.kind.id:    SheepType,
	ChickenType.kind.id:  ChickenType,
	ZombieType.kind.id:   ZombieType,
	SkeletonType.kind.id: SkeletonType,
}

// newMobHandle creates a handle using the entity type and the profile passed.
func newMobHandle(opts world.EntitySpawnOpts, t mobType, natural bool) *world.EntityHandle {
	return opts.New(t, newMobBehaviour(t.kind, natural))
}

// NewMob creates a living mob handle for a vanilla entity identifier. The name
// may be passed with or without the minecraft: namespace. It returns nil when
// the entity has no server-side implementation yet.
func NewMob(opts world.EntitySpawnOpts, name string) *world.EntityHandle {
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	t, ok := mobTypes[name]
	if !ok {
		return nil
	}
	return newMobHandle(opts, t, false)
}

// NewNaturalMob creates a mob from the natural-spawning system.
func NewNaturalMob(opts world.EntitySpawnOpts, name string) *world.EntityHandle {
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	t, ok := mobTypes[name]
	if !ok || t.kind.spawnCategory == mobSpawnNone {
		return nil
	}
	return newMobHandle(opts, t, true)
}

type mobBehaviour struct {
	BaseBehaviour
	kind            mobKind
	health          *HealthManager
	effects         *EffectManager
	movement        *MovementComputer
	immuneUntil     time.Time
	lastDamage      float64
	attackCooldown  int
	target          *world.EntityHandle
	wanderTarget    mgl64.Vec3
	wanderTicks     int
	dead            bool
	deathTicks      int
	despawnTicks    int
	deathActionSent bool
	naturalSpawn    bool
}

func newMobBehaviour(kind mobKind, natural bool) *mobBehaviour {
	return &mobBehaviour{
		BaseBehaviour: NewBaseBehaviour(),
		kind:          kind,
		health:        NewHealthManager(kind.maxHP, kind.maxHP),
		effects:       NewEffectManager(),
		naturalSpawn:  natural,
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

// Tick advances mob effects, basic goal-based movement, melee/ranged attacks,
// daylight burning and natural despawning.
const mobDeathAnimationTicks = 22

func (m *Mob) Tick(tx *world.Tx, current int64) {
	b := m.mobState()
	if b.dead {
		if !b.deathActionSent {
			for _, viewer := range tx.Viewers(m.Position()) {
				viewer.ViewEntityAction(m, DeathAction{})
			}
			b.deathActionSent = true
		}
		if b.deathTicks > 0 {
			b.deathTicks--
			return
		}
		m.dropLoot()
		_ = m.Close()
		return
	}
	b.effects.Tick(m, tx)
	if b.naturalSpawn && m.tickDespawn(tx) {
		return
	}
	if b.kind.sunBurn && current%20 == 0 {
		m.tickSunBurn(tx)
		if m.Dead() {
			return
		}
	}
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
		maxTargetDistance := 28.0
		if b.kind.ranged {
			maxTargetDistance = 18
		}
		if current%10 == 0 || b.target == nil {
			b.target = b.closestPlayer(m, tx, maxTargetDistance)
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
					living.Hurt(mobAttackDamage(m), AttackDamageSource{Attacker: m})
				}
				b.attackCooldown = 20
			}
			return
		}
		if b.kind.ranged && distance >= 3 && distance <= 18 {
			m.data.Vel[0], m.data.Vel[2] = 0, 0
			if b.attackCooldown == 0 {
				b.fireArrow(m, target)
				b.attackCooldown = 40
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

func (b *mobBehaviour) closestPlayer(m *Mob, tx *world.Tx, maxDistance float64) *world.EntityHandle {
	var nearest *world.EntityHandle
	best := maxDistance * maxDistance
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

func (b *mobBehaviour) fireArrow(m *Mob, target world.Entity) {
	create := m.tx.World().EntityRegistry().Config().Arrow
	if create == nil {
		return
	}
	origin := m.Position().Add(mgl64.Vec3{0, 1.45})
	targetPos := target.Position().Add(mgl64.Vec3{0, 0.9})
	direction := targetPos.Sub(origin)
	if direction.LenSqr() == 0 {
		return
	}
	velocity := direction.Normalize().Mul(1.6)
	opts := world.EntitySpawnOpts{Position: origin, Velocity: velocity}
	arrow := create(opts, world.ArrowSpawnConfig{
		Damage:              2,
		Owner:               m,
		ObtainArrowOnPickup: true,
		Tip:                 potion.Water(),
	})
	if arrow != nil {
		m.tx.AddEntity(arrow)
	}
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

func mobAttackDamage(m *Mob) float64 {
	damage := m.mobState().kind.damage
	switch m.tx.World().Difficulty() {
	case world.DifficultyEasy:
		damage *= 0.5
	case world.DifficultyHard:
		damage *= 1.5
	}
	return damage
}

func (m *Mob) tickSunBurn(tx *world.Tx) {
	pos := cube.PosFromVec3(m.Position().Add(mgl64.Vec3{0, 0.5}))
	if _, wet := tx.Liquid(pos); wet || tx.RainingAt(pos) {
		m.Extinguish()
		return
	}
	timeOfDay := tx.World().Time() % 24000
	daylight := timeOfDay < 12000 && tx.SkyLight(pos) >= 15
	if daylight {
		m.SetOnFire(8 * time.Second)
	}
	if m.OnFireDuration() > 0 {
		m.Hurt(1, FireDamageSource{})
	}
}

const mobDespawnInactiveTicks = 30 * 20

// mobImmediateDespawnDistance returns the Bedrock immediate-despawn distance
// for the world's simulation distance.
func mobImmediateDespawnDistance(tickRange int) float64 {
	if tickRange <= 4 {
		return 44
	}
	return 128
}

// nextMobDespawnTicks tracks consecutive ticks spent more than 32 blocks from
// every player. The counter resets as soon as a player comes within that range.
func nextMobDespawnTicks(distanceSquared float64, inactivityTicks int) int {
	if distanceSquared <= 32*32 {
		return 0
	}
	return min(inactivityTicks+1, mobDespawnInactiveTicks)
}

func mobDistanceSquared(a, b mgl64.Vec3) float64 {
	return a.Sub(b).LenSqr()
}

func (m *Mob) tickDespawn(tx *world.Tx) bool {
	if m.NameTag() != "" {
		return false
	}
	nearest := nearestPlayerDistanceSquared(m.Position(), tx)
	maxDistance := mobImmediateDespawnDistance(tx.World().TickRange())
	if nearest > maxDistance*maxDistance {
		_ = m.Close()
		return true
	}

	b := m.mobState()
	b.despawnTicks = nextMobDespawnTicks(nearest, b.despawnTicks)
	if b.despawnTicks >= mobDespawnInactiveTicks && rand.IntN(800) == 0 {
		_ = m.Close()
		return true
	}
	return false
}

func nearestPlayerDistanceSquared(pos mgl64.Vec3, tx *world.Tx) float64 {
	nearest := math.Inf(1)
	for player := range tx.Players() {
		if gm, ok := player.(interface{ GameMode() world.GameMode }); ok && !gm.GameMode().HasCollision() {
			continue
		}
		if !mobWithinChunkTaxicabDistance(player.Position(), cube.PosFromVec3(pos), tx.World().TickRange()) {
			continue
		}
		nearest = min(nearest, mobDistanceSquared(player.Position(), pos))
	}
	return nearest
}

func (m *Mob) Health() float64    { return m.mobState().health.Health() }
func (m *Mob) MaxHealth() float64 { return m.mobState().health.MaxHealth() }
func (m *Mob) SetMaxHealth(v float64) {
	m.mobState().health.SetMaxHealth(v)
	m.updateState()
}
func (m *Mob) Dead() bool { return m.mobState().dead || m.Health() <= 0 }

// Hurt applies melee, projectile, fire and explosion damage to a mob.
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
	b.despawnTicks = 0
	b.health.AddHealth(-damage)
	for _, viewer := range m.tx.Viewers(m.Position()) {
		viewer.ViewEntityAction(m, HurtAction{})
	}
	m.updateState()
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
	b.deathTicks = mobDeathAnimationTicks
	b.deathActionSent = true
	m.data.Vel = mgl64.Vec3{}
	for _, viewer := range m.tx.Viewers(m.Position()) {
		viewer.ViewEntityAction(m, DeathAction{})
	}
	m.tx.PlaySound(m.Position(), sound.Custom{Name: b.kind.id + ".death", Volume: 1, Pitch: 1})
}

// dropLoot creates the mob's drops after its death animation has completed.
func (m *Mob) dropLoot() {
	b := m.mobState()
	if !m.tx.World().GameRuleBool("domobloot", true) || b.kind.lootItem == nil {
		return
	}
	create := m.tx.World().EntityRegistry().Config().Item
	if create == nil {
		return
	}
	opts := world.EntitySpawnOpts{Position: m.Position().Add(mgl64.Vec3{0, 0.25})}
	m.tx.AddEntity(create(opts, item.NewStack(b.kind.lootItem, 1)))
}

func (m *Mob) Heal(amount float64, _ world.HealingSource) float64 {
	before := m.Health()
	if m.Dead() || amount <= 0 {
		return 0
	}
	m.mobState().health.AddHealth(amount)
	m.updateState()
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

func (m *Mob) AddEffect(e effect.Effect)  { m.mobState().effects.Add(e, m) }
func (m *Mob) RemoveEffect(e effect.Type) { m.mobState().effects.Remove(e, m) }
func (m *Mob) Effects() []effect.Effect   { return m.mobState().effects.Effects() }
func (m *Mob) Speed() float64             { return m.mobState().kind.speed }
func (m *Mob) SetSpeed(v float64)         { m.mobState().kind.speed = max(v, 0) }

// mobKindCanSpawn reports whether a mob profile is allowed by the supplied biome and light.
func mobKindCanSpawn(kind mobKind, biome world.Biome, light uint8) bool {
	if kind.spawnCategory == mobSpawnNone || kind.spawnWeight <= 0 || kind.biomeTag == "" {
		return false
	}
	if light < kind.minLight || light > kind.maxLight {
		return false
	}
	for _, tag := range biome.Tags() {
		if tag == kind.biomeTag {
			return true
		}
	}
	return false
}

func chooseNaturalMob(category mobSpawnCategory, biome world.Biome, light uint8) (mobKind, bool) {
	eligible := make([]mobKind, 0, len(mobTypes))
	totalWeight := 0
	for _, typ := range mobTypes {
		kind := typ.kind
		if kind.spawnCategory != category || !mobKindCanSpawn(kind, biome, light) {
			continue
		}
		eligible = append(eligible, kind)
		totalWeight += kind.spawnWeight
	}
	if totalWeight == 0 {
		return mobKind{}, false
	}
	choice := rand.IntN(totalWeight)
	for _, kind := range eligible {
		choice -= kind.spawnWeight
		if choice < 0 {
			return kind, true
		}
	}
	return mobKind{}, false
}

func mobSpawnGroupSize(kind mobKind) int {
	minimum, maximum := max(kind.groupMin, 1), max(kind.groupMax, kind.groupMin, 1)
	return minimum + rand.IntN(maximum-minimum+1)
}

// naturalSpawnDistanceLimits returns the Bedrock spawn shell and horizontal chunk-centre limit.
func naturalSpawnDistanceLimits(tickRange int) (minDistance, maxDistance, maxHorizontalDistance float64) {
	minDistance, maxDistance, maxHorizontalDistance = 24, 128, 96
	if tickRange <= 4 {
		maxDistance, maxHorizontalDistance = 44, 44
	}
	return
}

// naturalSpawnDistanceAllowed checks the three-dimensional feet distance and,
// above simulation distance four, the horizontal distance to the candidate
// chunk centre. The distance-four shell has no additional chunk-centre limit.
func naturalSpawnDistanceAllowed(playerPos mgl64.Vec3, spawnPos cube.Pos, minDistance, maxDistance, maxHorizontalDistance float64, tickRange int) bool {
	if !mobWithinChunkTaxicabDistance(playerPos, spawnPos, tickRange) {
		return false
	}
	mobPos := mgl64.Vec3{float64(spawnPos[0]) + 0.5, float64(spawnPos[1]), float64(spawnPos[2]) + 0.5}
	if distanceSquared := mobDistanceSquared(playerPos, mobPos); distanceSquared < minDistance*minDistance || distanceSquared > maxDistance*maxDistance {
		return false
	}
	if tickRange <= 4 {
		return true
	}
	chunkCentreX := math.Floor(float64(spawnPos[0])/16)*16 + 8
	chunkCentreZ := math.Floor(float64(spawnPos[2])/16)*16 + 8
	dx, dz := chunkCentreX-playerPos[0], chunkCentreZ-playerPos[2]
	return dx*dx+dz*dz <= maxHorizontalDistance*maxHorizontalDistance
}

// mobWithinChunkTaxicabDistance applies Bedrock's simulation-distance chunk
// boundary for simulation distances above four. Distance four uses its own
// 44-block spherical despawn/spawn boundary without this additional filter.
func mobWithinChunkTaxicabDistance(playerPos mgl64.Vec3, blockPos cube.Pos, tickRange int) bool {
	if tickRange <= 4 {
		return true
	}
	playerChunkX, playerChunkZ := int(math.Floor(playerPos[0]/16)), int(math.Floor(playerPos[2]/16))
	entityChunkX, entityChunkZ := int(math.Floor(float64(blockPos[0])/16)), int(math.Floor(float64(blockPos[2])/16))
	chunkDistance := int(math.Abs(float64(playerChunkX-entityChunkX)) + math.Abs(float64(playerChunkZ-entityChunkZ)))
	return chunkDistance <= tickRange-1
}

// SpawnNaturalMobs performs bounded, loaded-chunk-only surface spawning around players.
// The first implementation follows the category/weight/group/biome/light rule structure
// used by Bedrock servers, while keeping the scan budget small for mobile hosts.
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
	tickRange := w.TickRange()
	minSpawnDistance, maxSpawnDistance, maxHorizontalSpawnDistance := naturalSpawnDistanceLimits(tickRange)

	animals, monsters := 0, 0
	for e := range tx.Entities() {
		mob, ok := e.(*Mob)
		if !ok {
			continue
		}
		if nearestPlayerDistanceSquared(mob.Position(), tx) > 128*128 {
			continue
		}
		switch mob.kind.spawnCategory {
		case mobSpawnMonster:
			monsters++
		case mobSpawnAnimal:
			animals++
		}
	}
	animalCap, monsterCap := min(40, len(players)*10), min(70, len(players)*15)
	timeOfDay := w.Time() % 24000
	night := timeOfDay >= 13000 && timeOfDay < 23000
	monsterAllowed := night && w.Difficulty() != world.DifficultyPeaceful

	for _, player := range players {
		for attempt := 0; attempt < 2; attempt++ {
			category := mobSpawnAnimal
			if monsterAllowed && monsters < monsterCap && rand.IntN(4) != 0 {
				category = mobSpawnMonster
			} else if animals >= animalCap {
				if monsterAllowed && monsters < monsterCap {
					category = mobSpawnMonster
				} else {
					continue
				}
			}
			if category == mobSpawnMonster && monsters >= monsterCap {
				continue
			}
			if category == mobSpawnAnimal && animals >= animalCap {
				continue
			}

			p := player.Position()
			// Sample across the full horizontal disk. The actual 3D spawn shell is
			// applied after the surface height and candidate chunk are known.
			distance := rand.Float64() * maxHorizontalSpawnDistance
			angle := rand.Float64() * math.Pi * 2
			x := int(math.Floor(p[0] + math.Cos(angle)*distance))
			z := int(math.Floor(p[2] + math.Sin(angle)*distance))
			spawnPos, ground, biome, light, ok := naturalSpawnSurface(tx, x, z, p[1])
			if !ok || !naturalSpawnDistanceAllowed(p, spawnPos, minSpawnDistance, maxSpawnDistance, maxHorizontalSpawnDistance, tickRange) {
				continue
			}
			kind, ok := chooseNaturalMob(category, biome, light)
			if !ok || (kind.grassRequired && !isGrassBlock(ground)) {
				continue
			}

			groupSize := mobSpawnGroupSize(kind)
			for member := 0; member < groupSize; member++ {
				if category == mobSpawnAnimal && animals >= animalCap || category == mobSpawnMonster && monsters >= monsterCap {
					break
				}
				memberPos := spawnPos
				if member > 0 {
					sx, sz := x+rand.IntN(9)-4, z+rand.IntN(9)-4
					var surfaceOK bool
					memberPos, ground, biome, light, surfaceOK = naturalSpawnSurface(tx, sx, sz, p[1])
					if !surfaceOK || !mobKindCanSpawn(kind, biome, light) || (kind.grassRequired && !isGrassBlock(ground)) {
						continue
					}
				}
				if !naturalSpawnDistanceAllowed(p, memberPos, minSpawnDistance, maxSpawnDistance, maxHorizontalSpawnDistance, tickRange) {
					continue
				}
				if naturalSpawnBlocked(tx, kind, memberPos) {
					continue
				}
				opts := world.EntitySpawnOpts{Position: mgl64.Vec3{float64(memberPos[0]) + 0.5, float64(memberPos[1]), float64(memberPos[2]) + 0.5}}
				handle := NewNaturalMob(opts, kind.id)
				if handle == nil {
					continue
				}
				tx.AddEntity(handle)
				if category == mobSpawnAnimal {
					animals++
				} else {
					monsters++
				}
			}
		}
	}
}

func naturalSpawnSurface(tx *world.Tx, x, z int, referenceY float64) (cube.Pos, world.Block, world.Biome, uint8, bool) {
	probe := cube.Pos{x, int(math.Floor(referenceY)), z}
	if _, loaded := tx.BlockLoaded(probe); !loaded {
		return cube.Pos{}, nil, nil, 0, false
	}
	y := tx.HighestLightBlocker(x, z)
	groundPos := cube.Pos{x, y, z}
	ground, loaded := tx.BlockLoaded(groundPos)
	if !loaded || len(ground.Model().BBox(groundPos, tx)) == 0 {
		return cube.Pos{}, nil, nil, 0, false
	}
	spawnPos := cube.Pos{x, y + 1, z}
	if spawnPos.OutOfBounds(tx.Range()) {
		return cube.Pos{}, nil, nil, 0, false
	}
	light := tx.Light(spawnPos)
	biome := tx.Biome(spawnPos)
	return spawnPos, ground, biome, light, true
}

func isGrassBlock(b world.Block) bool {
	_, ok := b.(block.Grass)
	return ok
}

func naturalSpawnBlocked(tx *world.Tx, kind mobKind, pos cube.Pos) bool {
	for y := 0; y < int(math.Ceil(kind.height)); y++ {
		checkPos := pos.Add(cube.Pos{0, y, 0})
		b, loaded := tx.BlockLoaded(checkPos)
		if !loaded {
			return true
		}
		if _, air := b.(block.Air); !air {
			return true
		}
	}
	box := cube.Box(-kind.width/2, 0, -kind.width/2, kind.width/2, kind.height, kind.width/2).
		Translate(mgl64.Vec3{float64(pos[0]) + 0.5, float64(pos[1]), float64(pos[2]) + 0.5})
	for range tx.EntitiesWithin(box.Grow(0.05)) {
		return true
	}
	return false
}
