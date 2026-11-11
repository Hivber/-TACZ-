package entity

import (
	"context"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type movementTestEntityType struct{}

func (movementTestEntityType) Open(_ *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	return &movementTestEntity{handle: handle, pos: data.Pos}
}
func (movementTestEntityType) EncodeEntity() string { return "dragonfly:movement_test" }
func (movementTestEntityType) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3)
}
func (movementTestEntityType) DecodeNBT(map[string]any, *world.EntityData) {}
func (movementTestEntityType) EncodeNBT(*world.EntityData) map[string]any {
	return nil
}

type movementTestEntityConfig struct{}

func (movementTestEntityConfig) Apply(*world.EntityData) {}

type movementTestEntity struct {
	handle *world.EntityHandle
	pos    mgl64.Vec3
	fire   time.Duration
}

func (e *movementTestEntity) Close() error            { return nil }
func (e *movementTestEntity) H() *world.EntityHandle  { return e.handle }
func (e *movementTestEntity) Position() mgl64.Vec3    { return e.pos }
func (e *movementTestEntity) Rotation() cube.Rotation { return cube.Rotation{} }
func (e *movementTestEntity) OnFireDuration() time.Duration {
	return e.fire
}

func (e *movementTestEntity) SetOnFire(duration time.Duration) {
	e.fire = max(duration, 0)
}

func (e *movementTestEntity) Extinguish() {
	e.fire = 0
}

func newMovementTestEntity(pos mgl64.Vec3, fire time.Duration) *movementTestEntity {
	handle := (world.EntitySpawnOpts{Position: pos}).New(movementTestEntityType{}, movementTestEntityConfig{})
	return &movementTestEntity{handle: handle, pos: pos, fire: fire}
}

func TestMovementComputerTriggersEntityStepper(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	orePos := cube.Pos{0, 0, 0}
	entityPos := mgl64.Vec3{0.5, 1, 0.5}
	var lit bool
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(orePos, block.RedstoneOre{Type: block.StoneOre()}, nil)
		e := newMovementTestEntity(entityPos, 0)
		computer := MovementComputer{Gravity: 0.08}
		computer.TickMovement(e, entityPos, mgl64.Vec3{}, cube.Rotation{}, tx)
		lit = tx.Block(orePos).(block.RedstoneOre).Lit
	}).Wait(context.Background())
	if !lit {
		t.Fatal("standing on redstone ore did not invoke EntityStepOn")
	}
}



func TestMovementComputerCollidesWithOtherEntities(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	w.Do(func(tx *world.Tx) {
		moving := tx.AddEntity(newMovementTestEntity(mgl64.Vec3{0.5, 1, 0.5}, 0).handle)
		_ = tx.AddEntity(newMovementTestEntity(mgl64.Vec3{1.3, 1, 0.5}, 0).handle)

		computer := MovementComputer{}
		delta, _ := computer.CheckCollision(tx, moving, moving.Position(), mgl64.Vec3{0.5, 0, 0})
		if math.Abs(delta[0]-0.2) > 0.0001 {
			t.Fatalf("expected horizontal movement to stop at entity boundary (0.2), got %v", delta[0])
		}
	}).Wait(context.Background())
}

func TestMovementComputerIgnoresEntitiesOutsideVerticalOverlap(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	w.Do(func(tx *world.Tx) {
		moving := tx.AddEntity(newMovementTestEntity(mgl64.Vec3{0.5, 1, 0.5}, 0).handle)
		_ = tx.AddEntity(newMovementTestEntity(mgl64.Vec3{1.3, 4, 0.5}, 0).handle)

		computer := MovementComputer{}
		delta, _ := computer.CheckCollision(tx, moving, moving.Position(), mgl64.Vec3{0.5, 0, 0})
		if math.Abs(delta[0]-0.5) > 0.0001 {
			t.Fatalf("expected no horizontal collision without vertical overlap, got %v", delta[0])
		}
	}).Wait(context.Background())
}

func TestMovementComputerTriggersEntityInsider(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()

	cakePos := cube.Pos{0, 0, 0}
	entityPos := mgl64.Vec3{0.5, 0.5, 0.5}
	var lit bool
	w.Do(func(tx *world.Tx) {
		tx.SetBlock(cakePos, block.Cake{Candle: true}, nil)
		e := newMovementTestEntity(entityPos, time.Second)
		computer := MovementComputer{Gravity: 0.08}
		computer.TickMovement(e, entityPos, mgl64.Vec3{}, cube.Rotation{}, tx)
		lit = tx.Block(cakePos).(block.Cake).CandleLit
	}).Wait(context.Background())
	if !lit {
		t.Fatal("a burning entity inside a candle cake did not invoke EntityInside")
	}
}
