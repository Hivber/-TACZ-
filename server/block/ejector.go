package block

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

// Dispenser is a redstone-activated container that ejects one item from a random non-empty slot.
type Dispenser struct {
	ejector
}

// Dropper is a redstone-activated container that inserts items into the container it faces, or ejects them.
type Dropper struct {
	ejector
}

type ejector struct {
	solid
	sourceWaterDisplacer

	// Facing is the face towards which the dispenser or dropper ejects items.
	Facing cube.Face
	// Triggered records whether the block has already reacted to its current redstone input.
	Triggered bool
	// CustomName is an optional name displayed in the container UI.
	CustomName string

	inventory *inventory.Inventory
	viewerMu  *sync.RWMutex
	viewers   map[ContainerViewer]struct{}
}

var (
	_ Container                      = Dispenser{}
	_ Container                      = Dropper{}
	_ world.NBTer                    = Dispenser{}
	_ world.NBTer                    = Dropper{}
	_ world.RedstonePowerConsumer    = Dispenser{}
	_ world.RedstonePowerConsumer    = Dropper{}
	_ world.RedstonePowerPostUpdater = Dispenser{}
	_ world.RedstonePowerPostUpdater = Dropper{}
	_ world.ScheduledTicker          = Dispenser{}
	_ world.ScheduledTicker          = Dropper{}
	_ ComparatorInput                = Dispenser{}
	_ ComparatorInput                = Dropper{}
)

// NewDispenser creates an initialised dispenser with a nine-slot inventory.
func NewDispenser() Dispenser { return Dispenser{ejector: newEjector()} }

// NewDropper creates an initialised dropper with a nine-slot inventory.
func NewDropper() Dropper { return Dropper{ejector: newEjector()} }

func newEjector() ejector {
	m := new(sync.RWMutex)
	viewers := make(map[ContainerViewer]struct{}, 1)
	return ejector{
		inventory: inventory.New(9, func(slot int, _, after item.Stack) {
			m.RLock()
			defer m.RUnlock()
			for viewer := range viewers {
				viewer.ViewSlotChange(slot, after)
			}
		}),
		viewerMu: m,
		viewers:  viewers,
	}
}

func (e ejector) ContainerSize() int { return 9 }

func (e ejector) Inventory(*world.Tx, cube.Pos) *inventory.Inventory {
	return e.inventory
}

func (e ejector) AddViewer(v ContainerViewer, _ *world.Tx, _ cube.Pos) {
	e.viewerMu.Lock()
	defer e.viewerMu.Unlock()
	e.viewers[v] = struct{}{}
}

func (e ejector) RemoveViewer(v ContainerViewer, _ *world.Tx, _ cube.Pos) {
	e.viewerMu.Lock()
	defer e.viewerMu.Unlock()
	delete(e.viewers, v)
}

func (e ejector) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, user item.User, _ *item.UseContext) bool {
	if opener, ok := user.(ContainerOpener); ok {
		opener.OpenBlockContainer(pos, tx)
		return true
	}
	return false
}

func (e ejector) ComparatorOutput(_ cube.Pos, _ *world.Tx) int {
	return comparatorContainerSignal(e.inventory)
}

func (e ejector) withName(a ...any) ejector {
	e.CustomName = strings.TrimSuffix(fmt.Sprintln(a...), "\n")
	return e
}

func (e ejector) breakInfo(drop world.Item) BreakInfo {
	return newBreakInfo(3.5, pickaxeHarvestable, pickaxeEffective, oneOf(drop)).
		withBlastResistance(3.5).
		withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
			if e.inventory == nil {
				return
			}
			for _, stack := range e.inventory.Clear() {
				dropItem(tx, stack, pos.Vec3Centre())
			}
		})
}

func (e ejector) encodeNBT(id string) map[string]any {
	if e.inventory == nil {
		facing, triggered, customName := e.Facing, e.Triggered, e.CustomName
		e = newEjector()
		e.Facing, e.Triggered, e.CustomName = facing, triggered, customName
	}
	data := map[string]any{
		"id":    id,
		"Items": nbtconv.InvToNBT(e.inventory),
	}
	if e.CustomName != "" {
		data["CustomName"] = e.CustomName
	}
	return data
}

func (e ejector) decodeNBT(data map[string]any) ejector {
	facing, triggered := e.Facing, e.Triggered
	e = newEjector()
	e.Facing, e.Triggered = facing, triggered
	e.CustomName = nbtconv.String(data, "CustomName")
	nbtconv.InvFromNBT(e.inventory, nbtconv.Slice(data, "Items"))
	return e
}

func (e ejector) redstonePowerUpdate(power int) (ejector, bool) {
	triggered := power > 0
	if e.Triggered == triggered {
		return e, false
	}
	e.Triggered = triggered
	return e, true
}

// RedstonePowerUpdate tracks whether the dispenser is currently powered.
func (d Dispenser) RedstonePowerUpdate(_ cube.Pos, _ *world.Tx, power int) (world.Block, bool) {
	e, changed := d.ejector.redstonePowerUpdate(power)
	d.ejector = e
	return d, changed
}

// RedstonePowerPostUpdate schedules one dispense on the rising edge of power.
func (d Dispenser) RedstonePowerPostUpdate(pos cube.Pos, tx *world.Tx, _, after world.Block, oldPower, newPower int) {
	if oldPower == 0 && newPower > 0 {
		tx.ScheduleBlockUpdate(pos, after, redstoneTicks(4))
	}
}

// RedstonePowerUpdate tracks whether the dropper is currently powered.
func (d Dropper) RedstonePowerUpdate(_ cube.Pos, _ *world.Tx, power int) (world.Block, bool) {
	e, changed := d.ejector.redstonePowerUpdate(power)
	d.ejector = e
	return d, changed
}

// RedstonePowerPostUpdate schedules one dropper operation on the rising edge of power.
func (d Dropper) RedstonePowerPostUpdate(pos cube.Pos, tx *world.Tx, _, after world.Block, oldPower, newPower int) {
	if oldPower == 0 && newPower > 0 {
		tx.ScheduleBlockUpdate(pos, after, redstoneTicks(4))
	}
}

// ScheduledTick performs the delayed dispenser operation.
func (d Dispenser) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if current, ok := tx.Block(pos).(Dispenser); ok {
		current.ejector.dispense(pos, tx, false)
	}
}

// ScheduledTick performs the delayed dropper operation.
func (d Dropper) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if current, ok := tx.Block(pos).(Dropper); ok {
		current.ejector.dispense(pos, tx, true)
	}
}

// UseOnBlock places the dispenser facing away from the player.
func (d Dispenser) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, d)
	if !used {
		return false
	}
	d = NewDispenser()
	d.Facing = calculateFace(user, pos)
	place(tx, pos, d, user, ctx)
	return placed(ctx)
}

// UseOnBlock places the dropper facing away from the player.
func (d Dropper) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, d)
	if !used {
		return false
	}
	d = NewDropper()
	d.Facing = calculateFace(user, pos)
	place(tx, pos, d, user, ctx)
	return placed(ctx)
}

func (d Dispenser) BreakInfo() BreakInfo { return d.ejector.breakInfo(d) }
func (d Dropper) BreakInfo() BreakInfo   { return d.ejector.breakInfo(d) }

func (Dispenser) EncodeItem() (string, int16) { return "minecraft:dispenser", 0 }
func (Dropper) EncodeItem() (string, int16)   { return "minecraft:dropper", 0 }

func (d Dispenser) EncodeBlock() (string, map[string]any) {
	return "minecraft:dispenser", map[string]any{
		"facing_direction": int32(d.Facing),
		"triggered_bit":    d.Triggered,
	}
}

func (d Dropper) EncodeBlock() (string, map[string]any) {
	return "minecraft:dropper", map[string]any{
		"facing_direction": int32(d.Facing),
		"triggered_bit":    d.Triggered,
	}
}

func (d Dispenser) EncodeNBT() map[string]any { return d.ejector.encodeNBT("Dispenser") }

func (d Dispenser) DecodeNBT(data map[string]any) any {
	d.ejector = d.ejector.decodeNBT(data)
	return d
}

func (d Dropper) EncodeNBT() map[string]any { return d.ejector.encodeNBT("Dropper") }

func (d Dropper) DecodeNBT(data map[string]any) any {
	d.ejector = d.ejector.decodeNBT(data)
	return d
}

func (d Dispenser) WithName(a ...any) world.Item {
	d.ejector = d.ejector.withName(a...)
	return d
}

func (d Dropper) WithName(a ...any) world.Item {
	d.ejector = d.ejector.withName(a...)
	return d
}

// ComparatorOutput exposes the fullness of the dispenser inventory to comparators.
func (d Dispenser) ComparatorOutput(pos cube.Pos, tx *world.Tx) int {
	return d.ejector.ComparatorOutput(pos, tx)
}

// ComparatorOutput exposes the fullness of the dropper inventory to comparators.
func (d Dropper) ComparatorOutput(pos cube.Pos, tx *world.Tx) int {
	return d.ejector.ComparatorOutput(pos, tx)
}

// dispense chooses one non-empty slot and handles one item.
func (e ejector) dispense(pos cube.Pos, tx *world.Tx, dropper bool) {
	if e.inventory == nil {
		tx.PlaySound(pos.Vec3Centre(), sound.Click{})
		return
	}
	slot := randomEjectorSlot(e.inventory.Slots())
	if slot < 0 {
		tx.PlaySound(pos.Vec3Centre(), sound.Click{})
		return
	}
	stack, _ := e.inventory.Item(slot)
	one := stack.Grow(1 - stack.Count())

	if dropper {
		front := pos.Side(e.Facing)
		if target, loaded := tx.BlockLoaded(front); loaded {
			if container, ok := target.(Container); ok {
				if insertEjectorItem(container.Inventory(tx, front), one) {
					_ = e.inventory.SetItem(slot, stack.Grow(-1))
					tx.SetBlockEntity(pos, Dropper{e})
				} else {
					tx.PlaySound(pos.Vec3Centre(), sound.Click{})
				}
				return
			}
		}
	}

	if !dropper && e.dispenseProjectile(pos, tx, one) {
		_ = e.inventory.SetItem(slot, stack.Grow(-1))
		tx.SetBlockEntity(pos, Dispenser{e})
		return
	}
	e.dispenseItemEntity(pos, tx, one)
	_ = e.inventory.SetItem(slot, stack.Grow(-1))
	if dropper {
		tx.SetBlockEntity(pos, Dropper{e})
	} else {
		tx.SetBlockEntity(pos, Dispenser{e})
	}
}

// randomEjectorSlot selects a non-empty slot with a uniform distribution.
func randomEjectorSlot(slots []item.Stack) int {
	selected, seen := -1, 0
	for slot, stack := range slots {
		if stack.Empty() {
			continue
		}
		seen++
		if rand.IntN(seen) == 0 {
			selected = slot
		}
	}
	return selected
}

// insertEjectorItem attempts to insert exactly one item into a destination inventory.
func insertEjectorItem(destination *inventory.Inventory, stack item.Stack) bool {
	if destination == nil || stack.Empty() {
		return false
	}
	added, _ := destination.AddItem(stack.Grow(1 - stack.Count()))
	return added == 1
}

func ejectorFaceVector(face cube.Face) mgl64.Vec3 {
	switch face {
	case cube.FaceDown:
		return mgl64.Vec3{0, -1, 0}
	case cube.FaceUp:
		return mgl64.Vec3{0, 1, 0}
	case cube.FaceNorth:
		return mgl64.Vec3{0, 0, -1}
	case cube.FaceSouth:
		return mgl64.Vec3{0, 0, 1}
	case cube.FaceWest:
		return mgl64.Vec3{-1, 0, 0}
	case cube.FaceEast:
		return mgl64.Vec3{1, 0, 0}
	default:
		return mgl64.Vec3{}
	}
}

func ejectorDispensePosition(pos cube.Pos, face cube.Face) mgl64.Vec3 {
	direction := ejectorFaceVector(face)
	origin := pos.Vec3().Add(mgl64.Vec3{0.5, 0.5, 0.5}).Add(direction.Mul(0.7))
	if face.Axis() == cube.Y {
		origin[1] -= 0.125
	} else {
		origin[1] -= 0.15625
	}
	return origin
}

func (e ejector) dispenseItemEntity(pos cube.Pos, tx *world.Tx, stack item.Stack) {
	face := ejectorFaceVector(e.Facing)
	origin := ejectorDispensePosition(pos, e.Facing)
	offset := rand.Float64()*0.1 + 0.2
	velocity := mgl64.Vec3{
		face[0]*offset + rand.NormFloat64()*0.045,
		0.2 + rand.NormFloat64()*0.045,
		face[2]*offset + rand.NormFloat64()*0.045,
	}
	tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: origin, Velocity: velocity}, stack))
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
}

func (e ejector) dispenseProjectile(pos cube.Pos, tx *world.Tx, stack item.Stack) bool {
	face := ejectorFaceVector(e.Facing)
	velocity := face.Add(mgl64.Vec3{0, 0.1, 0}).Normalize().Mul(1.1)
	velocity = velocity.Add(mgl64.Vec3{
		rand.NormFloat64() * 0.015,
		rand.NormFloat64() * 0.015,
		rand.NormFloat64() * 0.015,
	})
	opts := world.EntitySpawnOpts{Position: ejectorDispensePosition(pos, e.Facing), Velocity: velocity}

	switch projectile := stack.Item().(type) {
	case item.Arrow:
		conf := entity.ProjectileBehaviourConfig{
			Gravity:               0.05,
			Drag:                  0.01,
			Damage:                2,
			Potion:                projectile.Tip,
			Sound:                 sound.ArrowHit{},
			SurviveBlockCollision: true,
			PickupItem:            item.NewStack(projectile, 1),
		}
		tx.AddEntity(opts.New(entity.ArrowType, conf))
	case item.Egg:
		conf := entity.ProjectileBehaviourConfig{
			Gravity:       0.03,
			Drag:          0.01,
			Damage:        -1,
			Particle:      particle.EggSmash{},
			ParticleCount: 6,
			Hit:           ejectorEggHit,
		}
		tx.AddEntity(opts.New(entity.EggType, conf))
	case item.Snowball:
		conf := entity.ProjectileBehaviourConfig{
			Gravity:       0.03,
			Drag:          0.01,
			Damage:        -1,
			Particle:      particle.SnowballPoof{},
			ParticleCount: 6,
		}
		tx.AddEntity(opts.New(entity.SnowballType, conf))
	case item.Firework:
		tx.AddEntity(entity.NewFirework(opts, projectile))
	default:
		return false
	}
	tx.PlaySound(pos.Vec3Centre(), sound.BowShoot{})
	return true
}

func ejectorEggHit(_ *entity.Ent, tx *world.Tx, target trace.Result) {
	if rand.IntN(8) != 0 {
		return
	}
	count := 1
	if rand.IntN(32) == 0 {
		count = 4
	}
	create := tx.World().EntityRegistry().Config().Mob
	if create == nil {
		return
	}
	position := target.Position().Add(mgl64.Vec3{0, 0.25, 0})
	for range count {
		if handle := create(world.EntitySpawnOpts{Position: position}, "minecraft:chicken"); handle != nil {
			tx.AddEntity(handle)
		}
	}
}

// allDispensers returns all six facing directions and both triggered states.
func allDispensers() (all []world.Block) {
	for _, facing := range cube.Faces() {
		all = append(all, Dispenser{ejector: ejector{Facing: facing}})
		all = append(all, Dispenser{ejector: ejector{Facing: facing, Triggered: true}})
	}
	return
}

// allDroppers returns all six facing directions and both triggered states.
func allDroppers() (all []world.Block) {
	for _, facing := range cube.Faces() {
		all = append(all, Dropper{ejector: ejector{Facing: facing}})
		all = append(all, Dropper{ejector: ejector{Facing: facing, Triggered: true}})
	}
	return
}
