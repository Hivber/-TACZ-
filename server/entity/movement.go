package entity

import (
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// MovementComputer is used to compute movement of an entity. When constructed, the Gravity of the entity
// the movement is computed for must be passed.
type MovementComputer struct {
	Gravity, Drag     float64
	// StepHeight is the maximum height this entity may automatically step up while grounded.
	DragBeforeGravity bool
	StepHeight         float64

	onGround bool
}

// Movement represents the movement of a world.Entity as a result of a call to MovementComputer.TickMovement. The
// resulting position and velocity can be obtained by calling Position and Velocity. These can be sent to viewers by
// calling Send.
type Movement struct {
	v                    []world.Viewer
	e                    world.Entity
	pos, vel, dpos, dvel mgl64.Vec3
	rot                  cube.Rotation
	onGround             bool
}

// Send sends the Movement to any viewers watching the entity at the time of the movement. If the position/velocity
// changes were negligible, nothing is sent.
func (m *Movement) Send() {
	posChanged := !m.dpos.ApproxEqualThreshold(zeroVec3, epsilon)
	velChanged := !m.dvel.ApproxEqualThreshold(zeroVec3, epsilon)

	for _, v := range m.v {
		if posChanged {
			v.ViewEntityMovement(m.e, m.pos, m.rot, m.onGround)
		}
		if velChanged {
			v.ViewEntityVelocity(m.e, m.vel)
		}
	}
}

// Position returns the position as a result of the Movement as an mgl64.Vec3.
func (m *Movement) Position() mgl64.Vec3 {
	return m.pos
}

// Velocity returns the velocity after the Movement as an mgl64.Vec3.
func (m *Movement) Velocity() mgl64.Vec3 {
	return m.vel
}

// Rotation returns the rotation, yaw and pitch, of the entity after the Movement.
func (m *Movement) Rotation() cube.Rotation {
	return m.rot
}

// TickMovement performs a movement tick on an entity. Velocity is applied and changed according to the values
// of its Drag and Gravity.
// The new position of the entity after movement is returned.
// The resulting Movement can be sent to viewers by calling Movement.Send.
func (c *MovementComputer) TickMovement(e world.Entity, pos, vel mgl64.Vec3, rot cube.Rotation, tx *world.Tx) *Movement {
	viewers := tx.Viewers(pos)

	velBefore := vel
	vel = c.applyHorizontalForces(tx, pos, c.applyVerticalForces(vel))
	dPos, vel := c.CheckCollision(tx, e, pos, vel)
	newPos := pos.Add(dPos)
	c.updateBlockInteractions(tx, e, newPos)

	return &Movement{v: viewers, e: e,
		pos: newPos, vel: vel, dpos: dPos, dvel: vel.Sub(velBefore),
		rot: rot, onGround: c.onGround,
	}
}

// updateBlockInteractions calls the block callbacks for entities that are
// inside blocks or standing on top of them. These callbacks drive vanilla
// behaviour such as pressure plates, redstone ore and lit candle cakes.
func (c *MovementComputer) updateBlockInteractions(tx *world.Tx, e world.Entity, pos mgl64.Vec3) {
	entityBox := e.H().Type().BBox(e).Translate(pos)
	minPos, maxPos := entityBox.Min(), entityBox.Max()
	const epsilon = 0.00001
	maxBlock := cube.PosFromVec3(maxPos.Sub(mgl64.Vec3{epsilon, epsilon, epsilon}))

	for blockPos := range cube.Range3D(cube.PosFromVec3(minPos), maxBlock) {
		if insider, ok := tx.Block(blockPos).(block.EntityInsider); ok {
			insider.EntityInside(blockPos, tx, e)
		}
	}

	if !c.onGround {
		return
	}
	supportY := int(math.Floor(minPos[1] - epsilon))
	minX, minZ := int(math.Floor(minPos[0])), int(math.Floor(minPos[2]))
	maxX, maxZ := int(math.Floor(maxPos[0]-epsilon)), int(math.Floor(maxPos[2]-epsilon))
	for x := minX; x <= maxX; x++ {
		for z := minZ; z <= maxZ; z++ {
			blockPos := cube.Pos{x, supportY, z}
			if stepper, ok := tx.Block(blockPos).(block.EntityStepper); ok {
				stepper.EntityStepOn(blockPos, tx, e)
			}
		}
	}
}

// OnGround checks if the entity that this computer calculates is currently on the ground.
func (c *MovementComputer) OnGround() bool {
	return c.onGround
}

// zeroVec3 is a mgl64.Vec3 with zero values.
var zeroVec3 mgl64.Vec3

// epsilon is the epsilon used for thresholds for change used for change in position and velocity.
const epsilon = 0.001

// applyVerticalForces applies gravity and drag on the Y axis, based on the Gravity and Drag values set.
func (c *MovementComputer) applyVerticalForces(vel mgl64.Vec3) mgl64.Vec3 {
	if c.DragBeforeGravity {
		vel[1] *= 1 - c.Drag
	}
	vel[1] -= c.Gravity
	if !c.DragBeforeGravity {
		vel[1] *= 1 - c.Drag
	}
	return vel
}

// applyHorizontalForces applies friction to the velocity based on the Drag value, reducing it on the X and Z axes.
func (c *MovementComputer) applyHorizontalForces(tx *world.Tx, pos, vel mgl64.Vec3) mgl64.Vec3 {
	friction := 1 - c.Drag
	if c.onGround {
		if f, ok := tx.Block(cube.PosFromVec3(pos).Side(cube.FaceDown)).(interface {
			Friction() float64
		}); ok {
			friction *= f.Friction()
		} else {
			friction *= 0.6
		}
	}
	vel[0] *= friction
	vel[2] *= friction
	return vel
}

// CheckCollision handles the collision of the entity with blocks, adapting the velocity of the entity if it
// happens to collide with a block.
// The final velocity and the Vec3 that the entity should move is returned.
func (c *MovementComputer) CheckCollision(tx *world.Tx, e world.Entity, pos, vel mgl64.Vec3) (mgl64.Vec3, mgl64.Vec3) {
	deltaX, deltaY, deltaZ := vel[0], vel[1], vel[2]
	wasOnGround := c.onGround

	// Entities only ever have a single bounding box.
	entityBBox := e.H().Type().BBox(e).Translate(pos)
	blocks := blockBBoxsAround(tx, entityBBox.Extend(vel))

	// Query by the swept bounding box, not just entity origins. Include the
	// possible step-up volume so entities above the initial movement path count.
	searchBox := entityBBox.Extend(vel).Grow(c.StepHeight + epsilon)
	var entityBoxes []cube.BBox
	if collidable, ok := e.H().Type().(interface {
		CanCollideWith(world.Entity) bool
	}); ok {
		for other := range tx.EntitiesIntersecting(searchBox) {
			if other.H() == e.H() || !collidable.CanCollideWith(other) {
				continue
			}
			entityBoxes = append(entityBoxes, other.H().Type().BBox(other).Translate(other.Position()))
		}
	}

	if !mgl64.FloatEqualThreshold(deltaY, 0, epsilon) {
		// First we move the entity BBox on the Y axis.
		for _, blockBBox := range blocks {
			deltaY = entityBBox.YOffset(blockBBox, deltaY)
		}
		for _, otherBBox := range entityBoxes {
			deltaY = entityBBox.YOffset(otherBBox, deltaY)
		}
		entityBBox = entityBBox.Translate(mgl64.Vec3{0, deltaY})
	}
	if !mgl64.FloatEqualThreshold(deltaX, 0, epsilon) {
		// Then on the X axis.
		for _, blockBBox := range blocks {
			deltaX = entityBBox.XOffset(blockBBox, deltaX)
		}
		for _, otherBBox := range entityBoxes {
			deltaX = entityBBox.XOffset(otherBBox, deltaX)
		}
		entityBBox = entityBBox.Translate(mgl64.Vec3{deltaX})
	}
	if !mgl64.FloatEqualThreshold(deltaZ, 0, epsilon) {
		// And finally on the Z axis.
		for _, blockBBox := range blocks {
			deltaZ = entityBBox.ZOffset(blockBBox, deltaZ)
		}
		for _, otherBBox := range entityBoxes {
			deltaZ = entityBBox.ZOffset(otherBBox, deltaZ)
		}
	}
	if wasOnGround && c.StepHeight > epsilon && (!mgl64.FloatEqual(deltaX, vel[0]) || !mgl64.FloatEqual(deltaZ, vel[2])) {
		stepBlocks := blockBBoxsAround(tx, entityBBox.Grow(c.StepHeight))
		if steppedBox, stepX, stepY, stepZ, ok := tryStepUp(entityBBox, stepBlocks, entityBoxes, vel[0], vel[2], c.StepHeight); ok &&
			stepX*stepX+stepZ*stepZ > deltaX*deltaX+deltaZ*deltaZ+epsilon {
			entityBBox = steppedBox
			deltaX, deltaY, deltaZ = stepX, deltaY+stepY, stepZ
			c.onGround = true
		}
	}

	if !mgl64.FloatEqual(vel[1], 0) {
		// The Y velocity of the entity is currently not 0, meaning it is moving either up or down. We can
		// then assume the entity is not currently on the ground.
		c.onGround = false
	}
	if !mgl64.FloatEqual(deltaX, vel[0]) {
		vel[0] = 0
	}
	if !mgl64.FloatEqual(deltaY, vel[1]) {
		// The entity either hit the ground or hit the ceiling.
		if vel[1] < 0 {
			// The entity was going down, so we can assume it is now on the ground.
			c.onGround = true
		}
		vel[1] = 0
	}
	if !mgl64.FloatEqual(deltaZ, vel[2]) {
		vel[2] = 0
	}
	return mgl64.Vec3{deltaX, deltaY, deltaZ}, vel
}

// tryStepUp tests an alternate grounded path: lift, move horizontally, then settle onto the
// first collision surface below. It succeeds only when the downward phase finds support.
func tryStepUp(box cube.BBox, blocks, entities []cube.BBox, dx, dz, stepHeight float64) (cube.BBox, float64, float64, float64, bool) {
	up := stepHeight
	for _, obstacle := range blocks {
		up = box.YOffset(obstacle, up)
	}
	for _, obstacle := range entities {
		up = box.YOffset(obstacle, up)
	}
	if up <= epsilon {
		return box, 0, 0, 0, false
	}

	raised := box.Translate(mgl64.Vec3{0, up})
	stepX := dx
	for _, obstacle := range blocks {
		stepX = raised.XOffset(obstacle, stepX)
	}
	for _, obstacle := range entities {
		stepX = raised.XOffset(obstacle, stepX)
	}
	raised = raised.Translate(mgl64.Vec3{stepX})

	stepZ := dz
	for _, obstacle := range blocks {
		stepZ = raised.ZOffset(obstacle, stepZ)
	}
	for _, obstacle := range entities {
		stepZ = raised.ZOffset(obstacle, stepZ)
	}
	raised = raised.Translate(mgl64.Vec3{0, 0, stepZ})

	down := -up
	for _, obstacle := range blocks {
		down = raised.YOffset(obstacle, down)
	}
	for _, obstacle := range entities {
		down = raised.YOffset(obstacle, down)
	}
	if down <= -up+epsilon {
		return box, 0, 0, 0, false
	}
	return raised.Translate(mgl64.Vec3{0, down}), stepX, up + down, stepZ, true
}

// blockBBoxsAround returns all blocks around the entity passed, using the BBox passed to make a prediction of
// what blocks need to have their BBox returned.
func blockBBoxsAround(tx *world.Tx, box cube.BBox) []cube.BBox {
	grown := box.Grow(0.25)
	min, max := grown.Min(), grown.Max()
	minX, minY, minZ := int(math.Floor(min[0])), int(math.Floor(min[1])), int(math.Floor(min[2]))
	// The maximum bounds are exclusive: A block starting exactly at the box's
	// maximum cannot collide with it.
	maxX, maxY, maxZ := int(math.Ceil(max[0])), int(math.Ceil(max[1])), int(math.Ceil(max[2]))

	// A prediction of one BBox per block, plus an additional 2, in case. Allocate
	// it lazily so that entities moving through air do not allocate an empty slice
	// every tick.
	predicted := (maxX-minX)*(maxY-minY)*(maxZ-minZ) + 2
	var blockBBoxs []cube.BBox
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			for z := minZ; z < maxZ; z++ {
				pos := cube.Pos{x, y, z}
				boxes := tx.Block(pos).Model().BBox(pos, tx)
				if len(boxes) != 0 && blockBBoxs == nil {
					blockBBoxs = make([]cube.BBox, 0, predicted)
				}
				for _, box := range boxes {
					blockBBoxs = append(blockBBoxs, box.Translate(mgl64.Vec3{float64(x), float64(y), float64(z)}))
				}
			}
		}
	}
	return blockBBoxs
}
