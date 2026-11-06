package cmd

import (
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/world"
)

func init() {
	Register(New("clear", "Clears items from a player's inventory.", nil, clearSelfCommand{}, clearTargetsCommand{}))
	Register(New("effect", "Gives or clears status effects on players.", nil, effectGiveCommand{}, effectClearCommand{}))
	Register(New("xp", "Adds, removes or queries player experience.", []string{"experience"}, xpAddCommand{}, xpRemoveCommand{}, xpQueryCommand{}))
	Register(New("enchant", "Adds an enchantment to the item held by a player.", nil, enchantCommand{}))
}

// clearSelfCommand implements the Java-style /clear form without arguments.
type clearSelfCommand struct{}

func (clearSelfCommand) Allow(src Source) bool { return commandOperator(src) }

func (clearSelfCommand) Run(src Source, out *Output, _ *world.Tx) {
	target, ok := src.(commandPlayerTarget)
	if !ok {
		out.Error("/clear without a target must be executed by a player.")
		return
	}
	removed := clearPlayerInventory(target, "", int(^uint(0)>>1))
	out.Printf("Cleared %d item(s) from %s.", removed, target.Name())
}

// clearTargetsCommand implements /clear <targets> [item] [maxCount]. A maxCount
// of zero means unlimited, matching the Bedrock command convention.
type clearTargetsCommand struct {
	Targets  []Target
	Item     Optional[string]
	MaxCount Optional[int]
}

func (clearTargetsCommand) Allow(src Source) bool { return commandOperator(src) }

func (c clearTargetsCommand) Run(_ Source, out *Output, _ *world.Tx) {
	limit := c.MaxCount.LoadOr(0)
	if limit < 0 {
		out.Error("maxCount cannot be negative.")
		return
	}

	wantedName := ""
	if raw, supplied := c.Item.Load(); supplied {
		registered, ok := world.ItemByName(commandItemName(raw), 0)
		if !ok {
			out.Errorf("Unknown or unavailable item: %s", raw)
			return
		}
		wantedName, _ = registered.EncodeItem()
	}

	players := commandPlayers(c.Targets)
	if len(players) == 0 {
		out.Error("No player targets matched.")
		return
	}
	removed := 0
	for _, target := range players {
		remaining := limit
		if remaining == 0 {
			remaining = int(^uint(0) >> 1)
		}
		removed += clearPlayerInventory(target, wantedName, remaining)
	}
	out.Printf("Cleared %d item(s) from %d player(s).", removed, len(players))
}

// clearPlayerInventory removes matching items from the player's normal
// inventory and hotbar. The main-hand slot is part of this inventory.
func clearPlayerInventory(target commandPlayerTarget, wantedName string, limit int) int {
	removed := 0
	for slot, stack := range target.Inventory().Slots() {
		if stack.Empty() {
			continue
		}
		if wantedName != "" {
			name, _ := stack.Item().EncodeItem()
			if name != wantedName {
				continue
			}
		}
		count := stack.Count()
		if count > limit-removed {
			count = limit - removed
		}
		if count <= 0 {
			break
		}
		_ = target.Inventory().SetItem(slot, stack.Grow(-count))
		removed += count
		if removed >= limit {
			break
		}
	}
	return removed
}

type effectName string

func (effectName) Type() string { return "EffectName" }
func (effectName) Options(Source) []string {
	return []string{
		"speed", "slowness", "haste", "mining_fatigue", "strength", "instant_health", "instant_damage",
		"jump_boost", "nausea", "regeneration", "resistance", "fire_resistance", "water_breathing",
		"invisibility", "blindness", "night_vision", "hunger", "weakness", "poison", "wither", "health_boost",
		"absorption", "saturation", "levitation", "fatal_poison", "conduit_power", "slow_falling", "darkness", "all",
		"minecraft:speed", "minecraft:slowness", "minecraft:haste", "minecraft:mining_fatigue", "minecraft:strength",
		"minecraft:instant_health", "minecraft:instant_damage", "minecraft:jump_boost", "minecraft:nausea",
		"minecraft:regeneration", "minecraft:resistance", "minecraft:fire_resistance", "minecraft:water_breathing",
		"minecraft:invisibility", "minecraft:blindness", "minecraft:night_vision", "minecraft:hunger", "minecraft:weakness",
		"minecraft:poison", "minecraft:wither", "minecraft:health_boost", "minecraft:absorption", "minecraft:saturation",
		"minecraft:levitation", "minecraft:fatal_poison", "minecraft:conduit_power", "minecraft:slow_falling", "minecraft:darkness",
		"minecraft:all",
	}
}

var commandEffectTypes = map[string]effect.Type{
	"speed":           effect.Speed,
	"slowness":        effect.Slowness,
	"haste":           effect.Haste,
	"mining_fatigue":  effect.MiningFatigue,
	"strength":        effect.Strength,
	"instant_health":  effect.InstantHealth,
	"instant_damage":  effect.InstantDamage,
	"jump_boost":      effect.JumpBoost,
	"nausea":          effect.Nausea,
	"regeneration":    effect.Regeneration,
	"resistance":      effect.Resistance,
	"fire_resistance": effect.FireResistance,
	"water_breathing": effect.WaterBreathing,
	"invisibility":    effect.Invisibility,
	"blindness":       effect.Blindness,
	"night_vision":    effect.NightVision,
	"hunger":          effect.Hunger,
	"weakness":        effect.Weakness,
	"poison":          effect.Poison,
	"wither":          effect.Wither,
	"health_boost":    effect.HealthBoost,
	"absorption":      effect.Absorption,
	"saturation":      effect.Saturation,
	"levitation":      effect.Levitation,
	"fatal_poison":    effect.FatalPoison,
	"conduit_power":   effect.ConduitPower,
	"slow_falling":    effect.SlowFalling,
	"darkness":        effect.Darkness,
}

func commandEffectType(name effectName) (effect.Type, bool) {
	key := strings.ToLower(strings.TrimSpace(string(name)))
	key = strings.TrimPrefix(key, "minecraft:")
	t, ok := commandEffectTypes[key]
	return t, ok
}

type commandEffectTarget interface {
	Target
	Name() string
	AddEffect(effect.Effect)
	RemoveEffect(effect.Type)
	Effects() []effect.Effect
}

func commandEffectPlayers(targets []Target) []commandEffectTarget {
	players := make([]commandEffectTarget, 0, len(targets))
	for _, target := range targets {
		if p, ok := target.(commandEffectTarget); ok {
			players = append(players, p)
		}
	}
	return players
}

type effectGiveCommand struct {
	Give          SubCommand `cmd:"give"`
	Targets       []Target
	Effect        effectName
	Seconds       Optional[int]
	Amplifier     Optional[int]
	HideParticles Optional[bool]
}

func (effectGiveCommand) Allow(src Source) bool { return commandOperator(src) }

func (c effectGiveCommand) Run(_ Source, out *Output, _ *world.Tx) {
	t, ok := commandEffectType(c.Effect)
	if !ok {
		out.Errorf("Unknown effect: %s", c.Effect)
		return
	}
	seconds := c.Seconds.LoadOr(30)
	amplifier := c.Amplifier.LoadOr(0)
	hideParticles := c.HideParticles.LoadOr(false)
	if seconds < 1 || seconds > 604800 {
		out.Error("Effect duration must be between 1 and 604800 seconds.")
		return
	}
	if amplifier < 0 || amplifier > 30 {
		out.Error("Effect amplifier must be between 0 and 30.")
		return
	}
	players := commandEffectPlayers(c.Targets)
	if len(players) == 0 {
		out.Error("No player targets matched.")
		return
	}

	var applied effect.Effect
	if lasting, ok := t.(effect.LastingType); ok {
		applied = effect.New(lasting, amplifier+1, time.Duration(seconds)*time.Second)
	} else {
		applied = effect.NewInstant(t, amplifier)
	}
	if hideParticles {
		applied = applied.WithoutParticles()
	}
	for _, target := range players {
		target.AddEffect(applied)
	}
	out.Printf("Applied %s to %d player(s).", c.Effect, len(players))
}

type effectClearCommand struct {
	Clear   SubCommand `cmd:"clear"`
	Targets []Target
	Effect  Optional[effectName]
}

func (effectClearCommand) Allow(src Source) bool { return commandOperator(src) }

func (c effectClearCommand) Run(_ Source, out *Output, _ *world.Tx) {
	players := commandEffectPlayers(c.Targets)
	if len(players) == 0 {
		out.Error("No player targets matched.")
		return
	}
	name, supplied := c.Effect.Load()
	var selected effect.Type
	if supplied && !strings.EqualFold(strings.TrimPrefix(string(name), "minecraft:"), "all") {
		var ok bool
		selected, ok = commandEffectType(name)
		if !ok {
			out.Errorf("Unknown effect: %s", name)
			return
		}
	}
	removed := 0
	for _, target := range players {
		if selected != nil {
			for _, active := range target.Effects() {
				if active.Type() == selected {
					target.RemoveEffect(selected)
					removed++
					break
				}
			}
			continue
		}
		for _, active := range target.Effects() {
			target.RemoveEffect(active.Type())
			removed++
		}
	}
	out.Printf("Cleared effects from %d player(s); %d effect instance(s) removed.", len(players), removed)
}

type xpUnit string

func (xpUnit) Type() string            { return "ExperienceUnit" }
func (xpUnit) Options(Source) []string { return []string{"points", "levels"} }

type commandExperienceTarget interface {
	Target
	Name() string
	AddExperience(int) int
	RemoveExperience(int)
	ExperienceLevel() int
	ExperienceProgress() float64
	SetExperienceLevel(int)
}

func commandExperiencePlayers(targets []Target) []commandExperienceTarget {
	players := make([]commandExperienceTarget, 0, len(targets))
	for _, target := range targets {
		if p, ok := target.(commandExperienceTarget); ok {
			players = append(players, p)
		}
	}
	return players
}

type xpAddCommand struct {
	Add     SubCommand `cmd:"add"`
	Targets []Target
	Amount  int
	Unit    Optional[xpUnit]
}

func (xpAddCommand) Allow(src Source) bool { return commandOperator(src) }

func (c xpAddCommand) Run(_ Source, out *Output, _ *world.Tx) {
	applyCommandXP(c.Targets, c.Amount, c.Unit.LoadOr(xpUnit("points")), out, true)
}

func applyCommandXP(rawTargets []Target, amount int, unit xpUnit, out *Output, add bool) {
	if amount < 1 || amount > 1_000_000_000 {
		out.Error("Experience amount must be between 1 and 1,000,000,000.")
		return
	}
	players := commandExperiencePlayers(rawTargets)
	if len(players) == 0 {
		out.Error("No player targets matched.")
		return
	}
	for _, target := range players {
		switch unit {
		case "points":
			if add {
				target.AddExperience(amount)
			} else {
				target.RemoveExperience(amount)
			}
		case "levels":
			current := target.ExperienceLevel()
			if add {
				if current > 100_000_000 || amount > 100_000_000-current {
					out.Errorf("Adding %d levels to %s would exceed the safe maximum level of 100,000,000.", amount, target.Name())
					continue
				}
				target.SetExperienceLevel(current + amount)
			} else {
				target.SetExperienceLevel(max(0, current-amount))
			}
		default:
			out.Errorf("Unknown experience unit: %s", unit)
			return
		}
	}
	verb := "Added"
	if !add {
		verb = "Removed"
	}
	out.Printf("%s %d %s for %d player(s).", verb, amount, unit, len(players))
}

type xpRemoveCommand struct {
	Remove  SubCommand `cmd:"remove"`
	Targets []Target
	Amount  int
	Unit    Optional[xpUnit]
}

func (xpRemoveCommand) Allow(src Source) bool { return commandOperator(src) }

func (c xpRemoveCommand) Run(_ Source, out *Output, _ *world.Tx) {
	applyCommandXP(c.Targets, c.Amount, c.Unit.LoadOr(xpUnit("points")), out, false)
}

type xpQueryCommand struct {
	Query   SubCommand `cmd:"query"`
	Targets []Target
}

func (xpQueryCommand) Allow(src Source) bool { return commandOperator(src) }

func (c xpQueryCommand) Run(_ Source, out *Output, _ *world.Tx) {
	players := commandExperiencePlayers(c.Targets)
	if len(players) == 0 {
		out.Error("No player targets matched.")
		return
	}
	for _, target := range players {
		out.Printf("%s has experience level %d (%.1f%% to the next level).", target.Name(), target.ExperienceLevel(), target.ExperienceProgress()*100)
	}
}

type enchantmentName string

func (enchantmentName) Type() string { return "EnchantmentName" }
func (enchantmentName) Options(Source) []string {
	return []string{
		"protection", "fire_protection", "feather_falling", "blast_protection", "projectile_protection",
		"thorns", "respiration", "depth_strider", "aqua_affinity", "sharpness", "knockback", "fire_aspect",
		"efficiency", "silk_touch", "unbreaking", "fortune", "power", "punch", "flame", "infinity",
		"mending", "vanishing_curse", "multishot", "piercing", "quick_charge", "soul_speed", "swift_sneak",
		"minecraft:protection", "minecraft:fire_protection", "minecraft:feather_falling", "minecraft:blast_protection",
		"minecraft:projectile_protection", "minecraft:thorns", "minecraft:respiration", "minecraft:depth_strider",
		"minecraft:aqua_affinity", "minecraft:sharpness", "minecraft:knockback", "minecraft:fire_aspect",
		"minecraft:efficiency", "minecraft:silk_touch", "minecraft:unbreaking", "minecraft:fortune", "minecraft:power",
		"minecraft:punch", "minecraft:flame", "minecraft:infinity", "minecraft:mending", "minecraft:vanishing_curse",
		"minecraft:multishot", "minecraft:piercing", "minecraft:quick_charge", "minecraft:soul_speed", "minecraft:swift_sneak",
	}
}

var commandEnchantmentTypes = map[string]item.EnchantmentType{
	"protection":            enchantment.Protection,
	"fire_protection":       enchantment.FireProtection,
	"feather_falling":       enchantment.FeatherFalling,
	"blast_protection":      enchantment.BlastProtection,
	"projectile_protection": enchantment.ProjectileProtection,
	"thorns":                enchantment.Thorns,
	"respiration":           enchantment.Respiration,
	"depth_strider":         enchantment.DepthStrider,
	"aqua_affinity":         enchantment.AquaAffinity,
	"sharpness":             enchantment.Sharpness,
	"knockback":             enchantment.Knockback,
	"fire_aspect":           enchantment.FireAspect,
	"efficiency":            enchantment.Efficiency,
	"silk_touch":            enchantment.SilkTouch,
	"unbreaking":            enchantment.Unbreaking,
	"fortune":               enchantment.Fortune,
	"power":                 enchantment.Power,
	"punch":                 enchantment.Punch,
	"flame":                 enchantment.Flame,
	"infinity":              enchantment.Infinity,
	"mending":               enchantment.Mending,
	"vanishing_curse":       enchantment.CurseOfVanishing,
	"multishot":             enchantment.Multishot,
	"piercing":              enchantment.Piercing,
	"quick_charge":          enchantment.QuickCharge,
	"soul_speed":            enchantment.SoulSpeed,
	"swift_sneak":           enchantment.SwiftSneak,
}

func commandEnchantmentType(name enchantmentName) (item.EnchantmentType, bool) {
	key := strings.ToLower(strings.TrimSpace(string(name)))
	key = strings.TrimPrefix(key, "minecraft:")
	t, ok := commandEnchantmentTypes[key]
	return t, ok
}

type commandEnchantTarget interface {
	Target
	Name() string
	HeldItems() (item.Stack, item.Stack)
	SetHeldItems(item.Stack, item.Stack)
}

type enchantCommand struct {
	Targets     []Target
	Enchantment enchantmentName
	Level       Optional[int]
}

func (enchantCommand) Allow(src Source) bool { return commandOperator(src) }

func (c enchantCommand) Run(_ Source, out *Output, _ *world.Tx) {
	t, ok := commandEnchantmentType(c.Enchantment)
	if !ok {
		out.Errorf("Unknown enchantment: %s", c.Enchantment)
		return
	}
	level := c.Level.LoadOr(1)
	if level < 1 || level > t.MaxLevel() {
		out.Errorf("%s level must be between 1 and %d.", c.Enchantment, t.MaxLevel())
		return
	}
	updated := 0
	for _, rawTarget := range c.Targets {
		target, ok := rawTarget.(commandEnchantTarget)
		if !ok {
			continue
		}
		mainHand, offHand := target.HeldItems()
		if mainHand.Empty() {
			out.Errorf("%s is not holding an item.", target.Name())
			continue
		}
		updatedStack := mainHand.WithEnchantments(item.NewEnchantment(t, level))
		applied, ok := updatedStack.Enchantment(t)
		if !ok || applied.Level() != level {
			out.Errorf("%s cannot be applied to the item held by %s.", c.Enchantment, target.Name())
			continue
		}
		target.SetHeldItems(updatedStack, offHand)
		updated++
	}
	if updated == 0 {
		out.Error("No target held an item that could receive this enchantment.")
		return
	}
	out.Printf("Applied %s %d to %d player(s).", c.Enchantment, level, updated)
}
