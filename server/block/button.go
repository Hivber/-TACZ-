package block

import (
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	// ButtonStone is the stone button.
	ButtonStone = iota
	// ButtonPolishedBlackstone is the polished blackstone button.
	ButtonPolishedBlackstone
	// ButtonOak is the oak button.
	ButtonOak
	// ButtonSpruce is the spruce button.
	ButtonSpruce
	// ButtonBirch is the birch button.
	ButtonBirch
	// ButtonJungle is the jungle button.
	ButtonJungle
	// ButtonAcacia is the acacia button.
	ButtonAcacia
	// ButtonDarkOak is the dark oak button.
	ButtonDarkOak
	// ButtonMangrove is the mangrove button.
	ButtonMangrove
	// ButtonCherry is the cherry button.
	ButtonCherry
	// ButtonPaleOak is the pale oak button.
	ButtonPaleOak
	// ButtonPoplar is the poplar button.
	ButtonPoplar
	// ButtonBamboo is the bamboo button.
	ButtonBamboo
	// ButtonCrimson is the crimson button.
	ButtonCrimson
	// ButtonWarped is the warped button.
	ButtonWarped
	buttonKindCount
)

var buttonBlockNames = [...]string{
	"minecraft:stone_button",
	"minecraft:polished_blackstone_button",
	"minecraft:wooden_button",
	"minecraft:spruce_button",
	"minecraft:birch_button",
	"minecraft:jungle_button",
	"minecraft:acacia_button",
	"minecraft:dark_oak_button",
	"minecraft:mangrove_button",
	"minecraft:cherry_button",
	"minecraft:pale_oak_button",
	"minecraft:poplar_button",
	"minecraft:bamboo_button",
	"minecraft:crimson_button",
	"minecraft:warped_button",
}

// Button is an attachable, temporarily powered redstone source.
type Button struct {
	empty
	transparent
	flowingWaterDisplacer

	// Kind identifies the material and behaviour of the button.
	Kind int
	// Facing is the face of the button pointing away from its supporting block.
	Facing cube.Face
	// Powered reports whether the button is currently pressed.
	Powered bool
}

var (
	_ world.RedstonePowerSource       = Button{}
	_ world.RedstoneStrongPowerSource = Button{}
	_ world.ScheduledTicker           = Button{}
	_ world.NeighbourUpdateTicker     = Button{}
)

// RedstonePower weakly powers adjacent redstone components while the button is pressed.
func (b Button) RedstonePower(cube.Pos, *world.Tx, cube.Face) int {
	if b.Powered {
		return 15
	}
	return 0
}

// RedstoneStrongPower strongly powers only the block to which the button is attached.
func (b Button) RedstoneStrongPower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if b.Powered && b.Facing.Opposite() == face {
		return 15
	}
	return 0
}

// SideClosed reports whether liquid is blocked from flowing through a side of the button.
func (Button) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// NeighbourUpdateTick removes the button when its supporting face is no longer solid.
func (b Button) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	support := pos.Side(b.Facing.Opposite())
	if !tx.Block(support).Model().FaceSolid(support, b.Facing, tx) {
		breakBlock(b, pos, tx)
	}
}

// UseOnBlock places the button on the face clicked by the user.
func (b Button) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, face, used := firstReplaceable(tx, pos, face, b)
	if !used {
		return false
	}
	support := pos.Side(face.Opposite())
	if !tx.Block(support).Model().FaceSolid(support, face, tx) {
		return false
	}

	b.Facing = face
	b.Powered = false
	place(tx, pos, b, user, ctx)
	return placed(ctx)
}

// Activate presses the button and schedules its automatic release.
// A pressed button cannot be pressed again until it has released.
func (Button) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User, _ *item.UseContext) bool {
	current, ok := tx.Block(pos).(Button)
	if !ok || current.Powered {
		return false
	}
	current.Powered = true
	tx.SetBlock(pos, current, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.PowerOn{})
	tx.ScheduleBlockUpdate(pos, current, current.pulseDuration())
	return true
}

// ScheduledTick releases a pressed button after its pulse duration.
func (b Button) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	current, ok := tx.Block(pos).(Button)
	if !ok || !current.Powered {
		return
	}
	current.Powered = false
	tx.SetBlock(pos, current, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.PowerOff{})
}

func (b Button) pulseDuration() time.Duration {
	if b.Kind == ButtonStone || b.Kind == ButtonPolishedBlackstone {
		// Stone-family buttons remain pressed for 20 game ticks (one second).
		return time.Second
	}
	// Wooden buttons remain pressed for 30 game ticks (one and a half seconds).
	return 3 * time.Second / 2
}

// BreakInfo returns the hardness and drops of the button.
func (b Button) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, nothingEffective, oneOf(Button{Kind: b.Kind}))
}

// EncodeItem returns the item identifier of the button.
func (b Button) EncodeItem() (string, int16) {
	return buttonBlockNames[b.Kind], 0
}

// EncodeBlock returns the Bedrock block identifier and states of the button.
func (b Button) EncodeBlock() (string, map[string]any) {
	return buttonBlockNames[b.Kind], map[string]any{
		"button_pressed_bit": b.Powered,
		"facing_direction":   int32(b.Facing),
	}
}

// Model returns the thin collision model of the attached button.
func (b Button) Model() world.BlockModel {
	return model.Button{Facing: b.Facing, Pressed: b.Powered}
}

// allButtons returns every button kind, face and powered state.
func allButtons() (all []world.Block) {
	for kind := 0; kind < buttonKindCount; kind++ {
		for _, facing := range cube.Faces() {
			all = append(all,
				Button{Kind: kind, Facing: facing},
				Button{Kind: kind, Facing: facing, Powered: true},
			)
		}
	}
	return
}
