package cmd

import (
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
)

const maxClearItemCount = 2_147_483_647

func init() {
	Register(New("clear", "Clears items from a player's inventory.", nil, clearSelfCommand{}, clearTargetCommand{}))
	Register(New("effect", "Gives or removes status effects.", nil, effectGiveCommand{}, effectClearCommand{}))
	Register(New("enchant", "Adds an enchantment to a player's main-hand item.", nil, enchantCommand{}))
	Register(New("xp", "Adds, removes, sets or queries player experience.", []string{"experience"},
		xpAddCommand{}, xpRemoveCommand{}, xpSetCommand{}, xpQueryCommand{}))
}

// clearSelfCommand implements the no-argument Java form: /clear.
type clearSelfCommand struct{}

func (clearSelfCommand) Allow(src Source) bool { return commandOperator(src) }

func (clearSelfCommand) Run(src Source, out *Output, _ *world.Tx) {
	target, ok := src.(commandPlayerTarget)
	if !ok {
		out.Error("/clear without a target can only be used by a player.")
		return
	}
	removed := clearPlayerInventory(target, "", maxClearItemCount)
	out.Printf("Cleared %d item(s) from %s's inventory.", removed, target.Name())
}

type clearTargetCommand struct {
	Targets  []Target
	Item     Optional[string]
	MaxCount Optional[int]
}

func (clearTargetCommand) Allow(src Source) bool { return commandOperator(src) }

func (c clearTargetCommand) Run(_ Source, out *Output, _ *world.Tx) {
	maxCount := c.MaxCount.LoadOr(math.MaxInt)
	if maxCount < 0 {
		out.Error("The maximum item count cannot be negative.")
		return
	}
	itemID := ""
	if name, ok := c.Item.Load(); ok {
		it, found := world.ItemByName(commandItemName(name), 0)
		if !found {
			out.Errorf("Unknown or unavailable item: %s", commandItemName(name))
			return
		}
		itemID, _ = it.EncodeItem()
	}
	players := commandPlayers(c.Targets)
	if len(players) == 0 {
		out.Error("No player targets matched.")
		return
	}
	total := 0
	for _, target := range players {
		total += clearPlayerInventory(target, itemID, maxCount)
	}
	out.Printf("Cleared %d item(s) from %d player(s).", total, len(players))
}

func clearPlayerInventory(target commandPlayerTarget, itemID string, limit int) int {
	if itemID == "" && limit == math.MaxInt {
		removed := countInventoryItems(target.Inventory().Clear())
		removed += countInventoryItems(target.Armour().Inventory().Clear())
		target.SetHeldItems(item.Stack{}, item.Stack{})
		return removed
	}
	removed := 0
	removed += clearMatchingInventory(target.Inventory(), itemID, limit-removed)
	removed += clearMatchingInventory(target.Armour().Inventory(), itemID, limit-removed)
	main, off := target.HeldItems()
	if removed < limit && !off.Empty() {
		name, _ := off.Item().EncodeItem()
		if itemID == "" || name == itemID {
			n := min(off.Count(), limit-removed)
			target.SetHeldItems(main, off.Grow(-n))
			removed += n
		}
	}
	return removed
}

func clearMatchingInventory(inv *inventory.Inventory, itemID string, limit int) int {
	removed := 0
	for slot, stack := range inv.Slots() {
		if stack.Empty() || removed >= limit {
			continue
		}
		if itemID != "" {
			name, _ := stack.Item().EncodeItem()
			if name != itemID {
				continue
			}
		}
		n := min(stack.Count(), limit-removed)
		_ = inv.SetItem(slot, stack.Grow(-n))
		removed += n
	}
	return removed
}

func countInventoryItems(stacks []item.Stack) int {
	total := 0
	for _, stack := range stacks {
		total += stack.Count()
	}
	return total
}

// commandEffectType exposes the effect types currently implemented by Dragonfly.
type commandEffectType string

func (commandEffectType) Type() string { return "Effect" }
func (commandEffectType) Options(Source) []string {
	return []string{
		"speed", "slowness", "haste", "mining_fatigue", "strength", "instant_health",
		"instant_damage", "jump_boost", "nausea", "regeneration", "resistance",
		"fire_resistance", "water_breathing", "invisibility", "blindness", "night_vision",
		"hunger", "weakness", "poison", "wither", "health_boost", "absorption",
		"saturation", "levitation", "fatal_poison", "conduit_power", "slow_falling", "darkness",
	}
}

func commandEffect(name commandEffectType) (effect.Type, bool) {
	effects := map[commandEffectType]effect.Type{
		"speed": effect.Speed, "slowness": effect.Slowness, "haste": effect.Haste,
		"mining_fatigue": effect.MiningFatigue, "strength": effect.Strength,
		"instant_health": effect.InstantHealth, "instant_damage": effect.InstantDamage,
		"jump_boost": effect.JumpBoost, "nausea": effect.Nausea, "regeneration": effect.Regeneration,
		"resistance": effect.Resistance, "fire_resistance": effect.FireResistance,
		"water_breathing": effect.WaterBreathing, "invisibility": effect.Invisibility,
		"blindness": effect.Blindness, "night_vision": effect.NightVision, "hunger": effect.Hunger,
		"weakness": effect.Weakness, "poison": effect.Poison, "wither": effect.Wither,
		"health_boost": effect.HealthBoost, "absorption": effect.Absorption, "saturation": effect.Saturation,
		"levitation": effect.Levitation, "fatal_poison": effect.FatalPoison,
		"conduit_power": effect.ConduitPower, "slow_falling": effect.SlowFalling, "darkness": effect.Darkness,
	}
	e, ok := effects[name]
	return e, ok
}

type effectGiveCommand struct {
	Give         SubCommand
	Targets      []Target
	Effect       commandEffectType
	Seconds      Optional[int]
	Amplifier    Optional[int]
	HideParticles Optional[bool]
}

func (effectGiveCommand) Allow(src Source) bool { return commandOperator(src) }

func (c effectGiveCommand) Run(_ Source, out *Output, _ *world.Tx) {
	typ, ok := commandEffect(c.Effect)
	if !ok {
		out.Errorf("Unsupported effect: %s", c.Effect)
		return
	}
	seconds := c.Seconds.LoadOr(30)
	amplifier := c.Amplifier.LoadOr(0)
	if seconds < 1 || seconds > 31_536_000 {
		out.Error("Effect duration must be between 1 and 31536000 seconds.")
		return
	}
	if amplifier < 0 || amplifier > 255 {
		out.Error("Effect amplifier must be between 0 and 255.")
		return
	}
	hidden := c.HideParticles.LoadOr(false)
	level := amplifier + 1

	var applied effect.Effect
	if lasting, ok := typ.(effect.LastingType); ok {
		applied = effect.New(lasting, level, time.Duration(seconds)*time.Second)
	} else {
		applied = effect.NewInstant(typ, level)
	}
	if hidden {
		applied = applied.WithoutParticles()
	}
	affected := 0
	for _, target := range c.Targets {
		living, ok := target.(entity.Living)
		if !ok {
			continue
		}
		living.AddEffect(applied)
		affected++
	}
	if affected == 0 {
		out.Error("No living entity targets matched.")
		return
	}
	if _, lasting := typ.(effect.LastingType); lasting {
		out.Printf("Applied %s %d to %d entity(s) for %d second(s).", c.Effect, level, affected, seconds)
	} else {
		out.Printf("Applied instant effect %s %d to %d entity(s).", c.Effect, level, affected)
	}
}

type effectClearCommand struct {
	Clear   SubCommand
	Targets []Target
	Effect  Optional[commandEffectType]
}

func (effectClearCommand) Allow(src Source) bool { return commandOperator(src) }

func (c effectClearCommand) Run(_ Source, out *Output, _ *world.Tx) {
	var typ effect.Type
	hasType := false
	if name, ok := c.Effect.Load(); ok {
		var found bool
		typ, found = commandEffect(name)
		if !found {
			out.Errorf("Unsupported effect: %s", name)
			return
		}
		hasType = true
	}
	affected := 0
	for _, target := range c.Targets {
		living, ok := target.(entity.Living)
		if !ok {
			continue
		}
		if hasType {
			living.RemoveEffect(typ)
		} else {
			for _, active := range living.Effects() {
				living.RemoveEffect(active.Type())
			}
		}
		affected++
	}
	if affected == 0 {
		out.Error("No living entity targets matched.")
		return
	}
	out.Printf("Cleared effects from %d entity(s).", affected)
}

type commandEnchantment string

func (commandEnchantment) Type() string { return "Enchantment" }
func (commandEnchantment) Options(Source) []string {
	return []string{
		"protection", "fire_protection", "feather_falling", "blast_protection",
		"projectile_protection", "thorns", "respiration", "depth_strider", "aqua_affinity",
		"sharpness", "knockback", "fire_aspect", "efficiency", "silk_touch", "unbreaking",
		"fortune", "power", "punch", "flame", "infinity", "mending", "vanishing",
		"multishot", "piercing", "quick_charge", "soul_speed", "swift_sneak",
	}
}

func commandEnchantmentType(name commandEnchantment) (item.EnchantmentType, bool) {
	types := map[commandEnchantment]item.EnchantmentType{
		"protection": enchantment.Protection, "fire_protection": enchantment.FireProtection,
		"feather_falling": enchantment.FeatherFalling, "blast_protection": enchantment.BlastProtection,
		"projectile_protection": enchantment.ProjectileProtection, "thorns": enchantment.Thorns,
		"respiration": enchantment.Respiration, "depth_strider": enchantment.DepthStrider,
		"aqua_affinity": enchantment.AquaAffinity, "sharpness": enchantment.Sharpness,
		"knockback": enchantment.Knockback, "fire_aspect": enchantment.FireAspect,
		"efficiency": enchantment.Efficiency, "silk_touch": enchantment.SilkTouch,
		"unbreaking": enchantment.Unbreaking, "fortune": enchantment.Fortune,
		"power": enchantment.Power, "punch": enchantment.Punch, "flame": enchantment.Flame,
		"infinity": enchantment.Infinity, "mending": enchantment.Mending,
		"vanishing": enchantment.CurseOfVanishing, "multishot": enchantment.Multishot,
		"piercing": enchantment.Piercing, "quick_charge": enchantment.QuickCharge,
		"soul_speed": enchantment.SoulSpeed, "swift_sneak": enchantment.SwiftSneak,
	}
	e, ok := types[name]
	return e, ok
}

type enchantCommand struct {
	Targets     []Target
	Enchantment commandEnchantment
	Level       Optional[int]
}

func (enchantCommand) Allow(src Source) bool { return commandOperator(src) }

func (c enchantCommand) Run(_ Source, out *Output, _ *world.Tx) {
	typ, ok := commandEnchantmentType(c.Enchantment)
	if !ok {
		out.Errorf("Unsupported enchantment: %s", c.Enchantment)
		return
	}
	level := c.Level.LoadOr(1)
	if level < 1 || level > typ.MaxLevel() {
		out.Errorf("%s level must be between 1 and %d.", c.Enchantment, typ.MaxLevel())
		return
	}
	updated := 0
	for _, target := range commandPlayers(c.Targets) {
		hand, off := target.HeldItems()
		if hand.Empty() {
			continue
		}
		if !typ.CompatibleWithItem(hand.Item()) {
			continue
		}
		enchanted := hand.WithEnchantments(item.NewEnchantment(typ, level))
		if _, applied := enchanted.Enchantment(typ); !applied {
			continue
		}
		target.SetHeldItems(enchanted, off)
		updated++
	}
	if updated == 0 {
		out.Error("No target held an item compatible with the requested enchantment.")
		return
	}
	out.Printf("Applied %s %d to %d player(s).", c.Enchantment, level, updated)
}

type xpUnit string

func (xpUnit) Type() string { return "ExperienceUnit" }
func (xpUnit) Options(Source) []string { return []string{"points", "levels"} }

type experienceTarget interface {
	NamedTarget
	Experience() int
	AddExperience(int) int
	ExperienceLevel() int
	SetExperienceLevel(int)
}

type xpAddCommand struct {
	Add     SubCommand
	Targets []Target
	Amount  int
	Unit    Optional[xpUnit]
}

func (xpAddCommand) Allow(src Source) bool { return commandOperator(src) }

func (c xpAddCommand) Run(_ Source, out *Output, _ *world.Tx) {
	applyXP(c.Targets, c.Amount, c.Unit.LoadOr("points"), "add", out)
}

type xpRemoveCommand struct {
	Remove  SubCommand
	Targets []Target
	Amount  int
	Unit    Optional[xpUnit]
}

func (xpRemoveCommand) Allow(src Source) bool { return commandOperator(src) }

func (c xpRemoveCommand) Run(_ Source, out *Output, _ *world.Tx) {
	applyXP(c.Targets, c.Amount, c.Unit.LoadOr("points"), "remove", out)
}

type xpSetCommand struct {
	Set     SubCommand
	Targets []Target
	Amount  int
	Unit    Optional[xpUnit]
}

func (xpSetCommand) Allow(src Source) bool { return commandOperator(src) }

func (c xpSetCommand) Run(_ Source, out *Output, _ *world.Tx) {
	applyXP(c.Targets, c.Amount, c.Unit.LoadOr("points"), "set", out)
}

type xpQueryCommand struct {
	Query   SubCommand
	Targets []Target
	Unit    Optional[xpUnit]
}

func (xpQueryCommand) Allow(src Source) bool { return commandOperator(src) }

func (c xpQueryCommand) Run(_ Source, out *Output, _ *world.Tx) {
	unit := c.Unit.LoadOr("points")
	for _, target := range c.Targets {
		player, ok := target.(experienceTarget)
		if !ok {
			continue
		}
		switch unit {
		case "points":
			out.Printf("%s has %d experience point(s).", player.Name(), player.Experience())
		case "levels":
			out.Printf("%s has %d experience level(s).", player.Name(), player.ExperienceLevel())
		default:
			out.Errorf("Unknown experience unit: %s", unit)
			return
		}
	}
}

func applyXP(targets []Target, amount int, unit xpUnit, action string, out *Output) {
	if amount < 0 || amount > 2_000_000_000 {
		out.Error("Experience amount must be between 0 and 2000000000.")
		return
	}
	updated := 0
	for _, target := range targets {
		player, ok := target.(experienceTarget)
		if !ok {
			continue
		}
		switch unit {
		case "points":
			switch action {
			case "add":
				player.AddExperience(amount)
			case "remove":
				player.AddExperience(-amount)
			case "set":
				player.AddExperience(amount - player.Experience())
			}
		case "levels":
			level := player.ExperienceLevel()
			switch action {
			case "add":
				player.SetExperienceLevel(min(level+amount, 2_147_483_647))
			case "remove":
				player.SetExperienceLevel(max(level-amount, 0))
			case "set":
				player.SetExperienceLevel(amount)
			}
		default:
			out.Errorf("Unknown experience unit: %s", unit)
			return
		}
		updated++
	}
	if updated == 0 {
		out.Error("No player targets matched.")
		return
	}
	out.Printf("%s %d experience %s for %d player(s).", action, amount, unit, updated)
}

