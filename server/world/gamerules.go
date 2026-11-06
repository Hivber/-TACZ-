package world

import (
	"fmt"
	"strconv"
	"strings"
)

// GameRuleDefinition describes a Bedrock game rule supported by this server.
type GameRuleDefinition struct {
	Name                  string
	Default               any
	CanBeModifiedByPlayer bool
}

var gameRuleDefinitions = []GameRuleDefinition{
	{Name: "commandblockoutput", Default: true, CanBeModifiedByPlayer: true},
	{Name: "commandblocksenabled", Default: true, CanBeModifiedByPlayer: true},
	{Name: "dodaylightcycle", Default: true, CanBeModifiedByPlayer: true},
	{Name: "doentitydrops", Default: true, CanBeModifiedByPlayer: true},
	{Name: "dofiretick", Default: true, CanBeModifiedByPlayer: true},
	{Name: "recipesunlock", Default: true, CanBeModifiedByPlayer: true},
	{Name: "dolimitedcrafting", Default: false, CanBeModifiedByPlayer: true},
	{Name: "domobloot", Default: true, CanBeModifiedByPlayer: true},
	{Name: "domobspawning", Default: true, CanBeModifiedByPlayer: true},
	{Name: "dotiledrops", Default: true, CanBeModifiedByPlayer: true},
	{Name: "doweathercycle", Default: true, CanBeModifiedByPlayer: true},
	{Name: "drowningdamage", Default: true, CanBeModifiedByPlayer: true},
	{Name: "falldamage", Default: true, CanBeModifiedByPlayer: true},
	{Name: "firedamage", Default: true, CanBeModifiedByPlayer: true},
	{Name: "keepinventory", Default: false, CanBeModifiedByPlayer: true},
	{Name: "mobgriefing", Default: true, CanBeModifiedByPlayer: true},
	{Name: "pvp", Default: true, CanBeModifiedByPlayer: true},
	{Name: "showcoordinates", Default: false, CanBeModifiedByPlayer: true},
	{Name: "showdaysplayed", Default: false, CanBeModifiedByPlayer: true},
	{Name: "naturalregeneration", Default: true, CanBeModifiedByPlayer: true},
	{Name: "tntexplodes", Default: true, CanBeModifiedByPlayer: true},
	{Name: "sendcommandfeedback", Default: true, CanBeModifiedByPlayer: true},
	{Name: "doinsomnia", Default: true, CanBeModifiedByPlayer: true},
	{Name: "doimmediaterespawn", Default: false, CanBeModifiedByPlayer: true},
	{Name: "showdeathmessages", Default: true, CanBeModifiedByPlayer: true},
	{Name: "showtags", Default: true, CanBeModifiedByPlayer: true},
	{Name: "freezedamage", Default: true, CanBeModifiedByPlayer: true},
	{Name: "respawnblocksexplode", Default: true, CanBeModifiedByPlayer: true},
	{Name: "showbordereffect", Default: true, CanBeModifiedByPlayer: true},
	{Name: "showrecipemessages", Default: true, CanBeModifiedByPlayer: true},
	{Name: "projectilescanbreakblocks", Default: true, CanBeModifiedByPlayer: true},
	{Name: "tntexplosiondropdecay", Default: true, CanBeModifiedByPlayer: true},
	{Name: "maxcommandchainlength", Default: uint32(65535), CanBeModifiedByPlayer: true},
	{Name: "randomtickspeed", Default: uint32(1), CanBeModifiedByPlayer: true},
	{Name: "functioncommandlimit", Default: uint32(10000), CanBeModifiedByPlayer: true},
	{Name: "spawnradius", Default: uint32(5), CanBeModifiedByPlayer: true},
	{Name: "playerssleepingpercentage", Default: uint32(100), CanBeModifiedByPlayer: true},
	{Name: "playerwaypoints", Default: "everyone", CanBeModifiedByPlayer: true},
}

// GameRuleDefinitions returns all known Bedrock game rules in stable order.
func GameRuleDefinitions() []GameRuleDefinition {
	return append([]GameRuleDefinition(nil), gameRuleDefinitions...)
}

// DefaultGameRules returns a new map containing every known game rule.
func DefaultGameRules() map[string]any {
	values := make(map[string]any, len(gameRuleDefinitions))
	for _, definition := range gameRuleDefinitions {
		values[definition.Name] = definition.Default
	}
	return values
}

// CanonicalGameRule returns the canonical name of a known rule.
func CanonicalGameRule(name string) (string, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, definition := range gameRuleDefinitions {
		if name == definition.Name {
			return name, true
		}
	}
	return "", false
}

// ParseGameRuleValue parses a command argument according to the rule's type.
func ParseGameRuleValue(name, raw string) (any, error) {
	name, ok := CanonicalGameRule(name)
	if !ok {
		return nil, fmt.Errorf("unknown game rule %q", name)
	}
	for _, definition := range gameRuleDefinitions {
		if definition.Name != name {
			continue
		}
		switch definition.Default.(type) {
		case bool:
			switch strings.ToLower(raw) {
			case "true":
				return true, nil
			case "false":
				return false, nil
			default:
				return nil, fmt.Errorf("game rule %q expects true or false", name)
			}
		case uint32:
			value, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("game rule %q expects a non-negative integer", name)
			}
			switch name {
			case "randomtickspeed":
				if value > 4096 { return nil, fmt.Errorf("randomtickspeed must be between 0 and 4096") }
			case "functioncommandlimit":
				if value > 10000 { return nil, fmt.Errorf("functioncommandlimit must be between 0 and 10000") }
			case "spawnradius":
				if value > 128 { return nil, fmt.Errorf("spawnradius must be between 0 and 128") }
			case "playerssleepingpercentage":
				if value > 100 { return nil, fmt.Errorf("playerssleepingpercentage must be between 0 and 100") }
			}
			return uint32(value), nil
		case string:
			switch strings.ToLower(raw) {
			case "everyone":
				return "everyone", nil
			case "off":
				return "off", nil
			default:
				return nil, fmt.Errorf("game rule %q expects everyone or off", name)
			}
		}
	}
	return nil, fmt.Errorf("unsupported game rule %q", name)
}

// GameRules returns a copy of the world's current game rules.
// GameRules returns a copy of the world's current game rules.
func (w *World) GameRules() map[string]any {
	if w == nil || w.set == nil {
		return DefaultGameRules()
	}
	return w.set.GameRuleValues()
}

// GameRule returns the current value of one game rule.
func (w *World) GameRule(name string) (any, bool) {
	name, ok := CanonicalGameRule(name)
	if !ok || w == nil || w.set == nil { return nil, false }
	w.set.Lock()
	defer w.set.Unlock()
	value, ok := w.set.GameRules[name]
	if !ok {
		for _, definition := range gameRuleDefinitions {
			if definition.Name == name { return definition.Default, true }
		}
	}
	return value, ok
}

// GameRuleBool returns a boolean rule value, or fallback if unavailable.
func (w *World) GameRuleBool(name string, fallback bool) bool {
	value, ok := w.GameRule(name)
	if !ok { return fallback }
	result, ok := value.(bool)
	if !ok { return fallback }
	return result
}

// SetGameRule validates and updates one game rule.
func (w *World) SetGameRule(name string, value any) error {
	name, ok := CanonicalGameRule(name)
	if !ok { return fmt.Errorf("unknown game rule %q", name) }
	var expected any
	for _, definition := range gameRuleDefinitions {
		if definition.Name == name { expected = definition.Default; break }
	}
	switch expected.(type) {
	case bool:
		if _, ok := value.(bool); !ok { return fmt.Errorf("game rule %q expects a boolean", name) }
	case uint32:
		if _, ok := value.(uint32); !ok { return fmt.Errorf("game rule %q expects an integer", name) }
	case string:
		v, ok := value.(string)
		if !ok || (v != "everyone" && v != "off") { return fmt.Errorf("game rule %q expects everyone or off", name) }
	default:
		return fmt.Errorf("unsupported game rule %q", name)
	}
	if w == nil || w.set == nil { return fmt.Errorf("world is unavailable") }
	switch name {
	case "dodaylightcycle":
		w.enableTimeCycle(value.(bool))
	case "doweathercycle":
		w.enableWeatherCycle(value.(bool))
	default:
		w.set.Lock()
		if w.set.GameRules == nil { w.set.GameRules = DefaultGameRules() }
		w.set.GameRules[name] = value
		w.set.Unlock()
	}
	return nil
}
