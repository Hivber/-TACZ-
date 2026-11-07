package block

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"math/rand/v2"
	"time"
)

// Lectern is a librarian's job site block generated in villages. It holds a book for multiple players to read.
type Lectern struct {
	bass
	sourceWaterDisplacer

	// Facing represents the direction the Lectern is facing.
	Facing cube.Direction
	// Book is the book currently held by the Lectern.
	Book item.Stack
	// Page is the zero-based page index currently selected in the book UI.
	Page int
	// Powered indicates that the lectern is emitting its short page-turn redstone pulse.
	Powered bool
}

var (
	_ world.RedstonePowerSource = Lectern{}
	_ world.ScheduledTicker     = Lectern{}
)

// RedstonePower emits a full-strength pulse from every side while a player turns a page.
func (l Lectern) RedstonePower(cube.Pos, *world.Tx, cube.Face) int {
	if l.Powered {
		return 15
	}
	return 0
}

// ScheduledTick ends the short redstone pulse after one redstone tick (two game ticks).
func (l Lectern) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(Lectern)
	if !ok || !current.Powered {
		return
	}
	current.Powered = false
	tx.SetBlock(pos, current, nil)
}

// Model ...
func (Lectern) Model() world.BlockModel {
	return model.Lectern{}
}

// FuelInfo ...
func (Lectern) FuelInfo() item.FuelInfo {
	return newFuelInfo(time.Second * 15)
}

// SideClosed ...
func (Lectern) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// BreakInfo ...
func (l Lectern) BreakInfo() BreakInfo {
	d := []item.Stack{item.NewStack(Lectern{}, 1)}
	if !l.Book.Empty() {
		d = append(d, l.Book)
	}
	return newBreakInfo(2.5, alwaysHarvestable, axeEffective, simpleDrops(d...))
}

// UseOnBlock ...
func (l Lectern) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, l)
	if !used {
		return false
	}
	l.Facing = user.Rotation().Direction().Opposite()
	place(tx, pos, l, user, ctx)
	return placed(ctx)
}

// readableBook represents a book that can be read through a lectern.
type readableBook interface {
	// TotalPages returns the total number of pages in the book.
	TotalPages() int
	// Page returns a specific page from the book and true when the page exists. It will otherwise return an empty string
	// and false.
	Page(page int) (string, bool)
}

// Activate ...
func (l Lectern) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, ctx *item.UseContext) bool {
	if !l.Book.Empty() {
		if opener, ok := u.(ContainerOpener); ok {
			opener.OpenBlockContainer(pos, tx)
			return true
		}
		return false
	}

	held, _ := u.HeldItems()
	if _, ok := held.Item().(readableBook); !ok {
		// We can't put a non-book item on the lectern.
		return false
	}

	// Only one book is taken from the stack held, so only one may be put on the lectern.
	l.Book, l.Page = held.Grow(-held.Count()+1), 0
	tx.SetBlock(pos, l, nil)

	tx.PlaySound(pos.Vec3Centre(), sound.LecternBookPlace{})
	ctx.SubtractFromCount(1)
	return true
}

// Punch ...
func (l Lectern) Punch(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User) {
	if l.Book.Empty() {
		// We can't remove a book from the lectern if there isn't one.
		return
	}

	dropItem(tx, l.Book, pos.Side(cube.FaceUp).Vec3Middle())

	l.Book = item.Stack{}
	l.Powered = false
	tx.SetBlock(pos, l, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.Attack{})
}

// TurnPage updates the page the lectern is currently on to the page given.
func (l Lectern) TurnPage(pos cube.Pos, tx *world.Tx, page int) error {
	if page == l.Page {
		// We're already on the correct page, so we don't need to do anything.
		return nil
	}
	if l.Book.Empty() {
		return fmt.Errorf("lectern at %v is empty", pos)
	}
	if r, ok := l.Book.Item().(readableBook); ok && (page >= r.TotalPages() || page < 0) {
		return fmt.Errorf("page number %d is out of bounds", page)
	}
	l.Page = page
	l.Powered = true
	tx.SetBlock(pos, l, nil)
	tx.ScheduleBlockUpdate(pos, l, redstoneTicks(1))
	return nil
}

// ComparatorOutput returns the book-position signal using Bedrock Edition's two-page spread.
func (l Lectern) ComparatorOutput(cube.Pos, *world.Tx) int {
	if l.Book.Empty() {
		return 0
	}
	book, ok := l.Book.Item().(readableBook)
	if !ok {
		return 0
	}
	pages := book.TotalPages()
	if pages <= 3 {
		return 15
	}

	// Bedrock displays two pages at once. Only the highest even page in the
	// spread contributes, so odd trailing pages do not create another step.
	maxSpread := pages / 2
	currentSpread := l.Page/2 + 1
	currentSpread = min(max(currentSpread, 1), maxSpread)
	if maxSpread <= 1 {
		return 15
	}
	return 1 + 14*(currentSpread-1)/(maxSpread-1)
}

// EncodeNBT ...
func (l Lectern) EncodeNBT() map[string]any {
	m := map[string]any{
		"hasBook": boolByte(!l.Book.Empty()),
		"page":    int32(l.Page),
		"id":      "Lectern",
	}
	if r, ok := l.Book.Item().(readableBook); ok {
		m["book"] = item.WriteNBT(l.Book, true)
		m["totalPages"] = int32(r.TotalPages())
	}
	return m
}

// DecodeNBT ...
func (l Lectern) DecodeNBT(m map[string]any) any {
	l.Page = int(nbtconv.Int32(m, "page"))
	l.Book = item.MapNBT(m, "book")
	return l
}

// EncodeItem ...
func (Lectern) EncodeItem() (name string, meta int16) {
	return "minecraft:lectern", 0
}

// EncodeBlock ...
func (l Lectern) EncodeBlock() (string, map[string]any) {
	return "minecraft:lectern", map[string]any{
		"minecraft:cardinal_direction": l.Facing.String(),
		"powered_bit":                  uint8(boolByte(l.Powered)),
	}
}

// allLecterns ...
func allLecterns() (lecterns []world.Block) {
	for _, f := range cube.Directions() {
		for _, powered := range []bool{false, true} {
			lecterns = append(lecterns, Lectern{Facing: f, Powered: powered})
		}
	}
	return
}
