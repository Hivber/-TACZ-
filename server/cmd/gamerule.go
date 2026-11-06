package cmd

import (
	"sort"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func init() {
	Register(New("gamerule", "Sets or queries a Bedrock game rule.", nil, gamerule{}))
	Register(New("daylock", "Locks or unlocks the day-night cycle.", []string{"alwaysday"}, daylock{}))
}

type gamerule struct {
	Name  Optional[string]
	Value Optional[string]
}

func (g gamerule) Run(src Source, out *Output, tx *world.Tx) {
	w := commandWorld(src, tx)
	if w == nil {
		out.Error("gamerule requires a world.")
		return
	}
	nameValue, hasName := g.Name.Load()
	if !hasName {
		values := w.GameRules()
		names := make([]string, 0, len(values))
		for name := range values {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			out.Printf("%s = %v", name, values[name])
		}
		return
	}
	name, ok := world.CanonicalGameRule(nameValue)
	if !ok {
		out.Errorf("Unknown game rule: %s", nameValue)
		return
	}
	value, supplied := g.Value.Load()
	if !supplied {
		current, exists := w.GameRule(name)
		if !exists {
			out.Errorf("Game rule %s has no value.", name)
			return
		}
		out.Printf("%s = %v", name, current)
		return
	}
	parsed, err := world.ParseGameRuleValue(name, value)
	if err != nil {
		out.Error(err)
		return
	}
	if err := w.SetGameRule(name, parsed); err != nil {
		out.Error(err)
		return
	}
	syncGameRule(tx, name, parsed)
	out.Printf("Game rule %s has been updated to %v.", name, parsed)
}

type daylock struct {
	Locked Optional[bool]
}

func (daylock) Allow(src Source) bool {
	if operator, ok := src.(interface{ Operator() bool }); ok {
		return operator.Operator()
	}
	return true
}

func (d daylock) Run(src Source, out *Output, tx *world.Tx) {
	w := commandWorld(src, tx)
	if w == nil {
		out.Error("daylock requires a world.")
		return
	}
	locked := d.Locked.LoadOr(true)
	cycle := !locked
	if err := w.SetGameRule("dodaylightcycle", cycle); err != nil {
		out.Error(err)
		return
	}
	if locked {
		w.SetTime(5000)
	}
	syncGameRule(tx, "dodaylightcycle", cycle)
	out.Printf("Day lock: %t", locked)
}

func commandWorld(src Source, tx *world.Tx) *world.World {
	if tx != nil {
		return tx.World()
	}
	if provider, ok := src.(interface{ World() *world.World }); ok {
		return provider.World()
	}
	return nil
}

// syncGameRule broadcasts one changed rule to every player in the command's
// world. The value conversion follows the Bedrock GameRulesChanged packet.
func syncGameRule(tx *world.Tx, name string, value any) {
	if tx == nil {
		return
	}
	var packetValue any
	switch v := value.(type) {
	case bool:
		packetValue = v
	case uint32:
		packetValue = v
	case string:
		switch strings.ToLower(v) {
		case "everyone":
			packetValue = uint32(0)
		case "off":
			packetValue = uint32(1)
		}
	}
	if packetValue == nil {
		return
	}
	canModify := false
	for _, definition := range world.GameRuleDefinitions() {
		if definition.Name == name {
			canModify = definition.CanBeModifiedByPlayer
			break
		}
	}
	updates := []protocol.GameRule{{Name: name, CanBeModifiedByPlayer: canModify, Value: packetValue}}
	for player := range tx.Players() {
		if receiver, ok := player.(interface{ SendGameRules([]protocol.GameRule) }); ok {
			receiver.SendGameRules(updates)
		}
		switch name {
		case "showcoordinates":
			if v, ok := value.(bool); ok {
				if v {
					if p, ok := player.(interface{ ShowCoordinates() }); ok {
						p.ShowCoordinates()
					}
				} else if p, ok := player.(interface{ HideCoordinates() }); ok {
					p.HideCoordinates()
				}
			}
		case "doimmediaterespawn":
			if v, ok := value.(bool); ok {
				if v {
					if p, ok := player.(interface{ EnableInstantRespawn() }); ok {
						p.EnableInstantRespawn()
					}
				} else if p, ok := player.(interface{ DisableInstantRespawn() }); ok {
					p.DisableInstantRespawn()
				}
			}
		}
	}
}
