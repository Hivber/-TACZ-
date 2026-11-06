package cmd

import (
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player/bossbar"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// This file adds a small Java-style command compatibility layer using the
// native Dragonfly item, inventory, movement and boss-bar APIs.

func init() {
	Register(New("give", "Gives items to one or more players.", nil, giveCommand{}))
	Register(New("item", "Replaces an item in a player's hand, armour or inventory slot.", nil, itemCommand{}))
	Register(New("attribute", "Queries or changes supported player attributes.", nil, attributeCommand{}))
	Register(New("bossbar", "Displays a boss bar for one or more players.", nil, bossbarCommand{}))
	Register(New("bossbarclear", "Hides the current boss bar from one or more players.", nil, bossbarClearCommand{}))
}

func commandOperator(src Source) bool {
	if op, ok := src.(interface{ Operator() bool }); ok {
		return op.Operator()
	}
	// Trusted console sources do not implement Operator.
	return true
}

func commandItemName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	return name
}

type commandPlayerTarget interface {
	Target
	Name() string
	Collect(item.Stack) (int, bool)
	Inventory() *inventory.Inventory
	Armour() *inventory.Armour
	HeldItems() (item.Stack, item.Stack)
	SetHeldItems(item.Stack, item.Stack)
	Speed() float64
	SetSpeed(float64)
	FlightSpeed() float64
	SetFlightSpeed(float64)
	MaxHealth() float64
	SetMaxHealth(float64)
	Scale() float64
	SetScale(float64)
	SendBossBar(bossbar.BossBar)
	RemoveBossBar()
}

func commandPlayers(targets []Target) []commandPlayerTarget {
	players := make([]commandPlayerTarget, 0, len(targets))
	for _, target := range targets {
		if p, ok := target.(commandPlayerTarget); ok {
			players = append(players, p)
		}
	}
	return players
}

type giveCommand struct {
	Targets []Target
	Item    string
	Count   Optional[int]
}

func (giveCommand) Allow(src Source) bool { return commandOperator(src) }

func (g giveCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("/give must be executed from a player or a world-bound command source.")
		return
	}
	count := g.Count.LoadOr(1)
	if count < 1 || count > 1000000 {
		out.Error("Item count must be between 1 and 1000000.")
		return
	}
	name := commandItemName(g.Item)
	registered, ok := world.ItemByName(name, 0)
	if !ok {
		out.Errorf("Unknown or unavailable item: %s", name)
		return
	}
	targets := commandPlayers(g.Targets)
	if len(targets) == 0 {
		out.Error("No player targets matched.")
		return
	}

	givenPlayers := 0
	for _, target := range targets {
		stack := item.NewStack(registered, count)
		added, _ := target.Collect(stack)
		remaining := count - added
		for remaining > 0 {
			n := min(remaining, stack.MaxCount())
			dropped := item.NewStack(registered, n)
			opts := world.EntitySpawnOpts{Position: target.Position().Add(mgl64.Vec3{0, 0.25})}
			tx.AddEntity(entity.NewItemPickupDelay(opts, dropped, 0))
			remaining -= n
		}
		givenPlayers++
	}
	out.Printf("Gave %d item(s) to %d player(s).", count, givenPlayers)
}

type itemSlot string

func (itemSlot) Type() string { return "ItemSlot" }
func (itemSlot) Options(Source) []string {
	options := []string{
		"weapon.mainhand",
		"weapon.offhand",
		"armor.head",
		"armor.chest",
		"armor.legs",
		"armor.feet",
	}
	for i := 0; i < 36; i++ {
		options = append(options, "inventory."+strconv.Itoa(i))
	}
	return options
}

type itemCommand struct {
	Targets []Target
	Slot    itemSlot
	Item    string
	Count   Optional[int]
}

func (itemCommand) Allow(src Source) bool { return commandOperator(src) }

func (c itemCommand) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("/item must be executed from a player or a world-bound command source.")
		return
	}
	count := c.Count.LoadOr(1)
	if count < 1 || count > 64 {
		out.Error("Item count must be between 1 and 64.")
		return
	}
	name := commandItemName(c.Item)
	registered, ok := world.ItemByName(name, 0)
	if !ok {
		out.Errorf("Unknown or unavailable item: %s", name)
		return
	}
	stack := item.NewStack(registered, count)
	if count > stack.MaxCount() {
		out.Errorf("%s can only stack up to %d.", name, stack.MaxCount())
		return
	}
	targets := commandPlayers(c.Targets)
	if len(targets) == 0 {
		out.Error("No player targets matched.")
		return
	}

	updated := 0
	for _, target := range targets {
		switch string(c.Slot) {
		case "weapon.mainhand":
			_, off := target.HeldItems()
			target.SetHeldItems(stack, off)
		case "weapon.offhand":
			main, _ := target.HeldItems()
			target.SetHeldItems(main, stack)
		case "armor.head":
			target.Armour().SetHelmet(stack)
		case "armor.chest":
			target.Armour().SetChestplate(stack)
		case "armor.legs":
			target.Armour().SetLeggings(stack)
		case "armor.feet":
			target.Armour().SetBoots(stack)
		default:
			if strings.HasPrefix(string(c.Slot), "inventory.") {
				slot, err := strconv.Atoi(strings.TrimPrefix(string(c.Slot), "inventory."))
				if err != nil || slot < 0 || slot >= target.Inventory().Size() {
					continue
				}
				_ = target.Inventory().SetItem(slot, stack)
			} else {
				continue
			}
		}
		updated++
	}
	out.Printf("Replaced %s for %d player(s).", c.Slot, updated)
}

type playerAttribute string

func (playerAttribute) Type() string { return "PlayerAttribute" }
func (playerAttribute) Options(Source) []string {
	return []string{
		"minecraft:movement_speed",
		"minecraft:flying_speed",
		"minecraft:max_health",
		"minecraft:scale",
	}
}

type attributeCommand struct {
	Targets   []Target
	Attribute playerAttribute
	Value     Optional[float64]
}

func (attributeCommand) Allow(src Source) bool { return commandOperator(src) }

func (c attributeCommand) Run(_ Source, out *Output, _ *world.Tx) {
	targets := commandPlayers(c.Targets)
	if len(targets) == 0 {
		out.Error("No player targets matched.")
		return
	}
	value, set := c.Value.Load()
	if set && (value < 0 || value > 1024) {
		out.Error("Attribute value must be between 0 and 1024.")
		return
	}

	for _, target := range targets {
		var current float64
		switch string(c.Attribute) {
		case "minecraft:movement_speed":
			current = target.Speed()
			if set {
				if value > 2 {
					out.Error("movement_speed is limited to 2.0 by this server.")
					return
				}
				target.SetSpeed(value)
			}
		case "minecraft:flying_speed":
			current = target.FlightSpeed()
			if set {
				if value > 1 {
					out.Error("flying_speed is limited to 1.0 by this server.")
					return
				}
				target.SetFlightSpeed(value)
			}
		case "minecraft:max_health":
			current = target.MaxHealth()
			if set {
				if value <= 0 {
					out.Error("max_health must be greater than 0.")
					return
				}
				target.SetMaxHealth(value)
			}
		case "minecraft:scale":
			current = target.Scale()
			if set {
				if value <= 0 || value > 10 {
					out.Error("scale must be greater than 0 and no greater than 10.")
					return
				}
				target.SetScale(value)
			}
		default:
			out.Errorf("Unsupported attribute: %s", c.Attribute)
			return
		}
		if set {
			out.Printf("%s for %s set to %.3f.", c.Attribute, target.Name(), value)
		} else {
			out.Printf("%s for %s is %.3f.", c.Attribute, target.Name(), current)
		}
	}
}

type bossbarColour string

func (bossbarColour) Type() string { return "BossBarColour" }
func (bossbarColour) Options(Source) []string {
	return []string{"pink", "blue", "red", "green", "yellow", "purple", "white"}
}

func bossbarColourValue(name bossbarColour) bossbar.Colour {
	switch strings.ToLower(string(name)) {
	case "pink":
		return bossbar.Pink()
	case "blue":
		return bossbar.Blue()
	case "red":
		return bossbar.Red()
	case "green":
		return bossbar.Green()
	case "yellow":
		return bossbar.Yellow()
	case "white":
		return bossbar.White()
	default:
		return bossbar.Purple()
	}
}

type bossbarCommand struct {
	Targets  []Target
	Progress float64
	Colour   bossbarColour
	Text     Varargs
}

func (bossbarCommand) Allow(src Source) bool { return commandOperator(src) }

func (c bossbarCommand) Run(_ Source, out *Output, _ *world.Tx) {
	if c.Progress < 0 || c.Progress > 1 {
		out.Error("Boss bar progress must be between 0 and 1.")
		return
	}
	targets := commandPlayers(c.Targets)
	if len(targets) == 0 {
		out.Error("No player targets matched.")
		return
	}
	bar := bossbar.New(string(c.Text)).
		WithHealthPercentage(c.Progress).
		WithColour(bossbarColourValue(c.Colour))
	for _, target := range targets {
		target.SendBossBar(bar)
	}
	out.Printf("Displayed boss bar for %d player(s).", len(targets))
}

type bossbarClearCommand struct {
	Targets []Target
}

func (bossbarClearCommand) Allow(src Source) bool { return commandOperator(src) }

func (c bossbarClearCommand) Run(_ Source, out *Output, _ *world.Tx) {
	targets := commandPlayers(c.Targets)
	if len(targets) == 0 {
		out.Error("No player targets matched.")
		return
	}
	for _, target := range targets {
		target.RemoveBossBar()
	}
	out.Printf("Cleared boss bar for %d player(s).", len(targets))
}
