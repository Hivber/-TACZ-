package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// SendGameRules sends changed game rules to this client.
func (s *Session) SendGameRules(rules []protocol.GameRule) {
	if len(rules) == 0 {
		return
	}
	s.writePacket(&packet.GameRulesChanged{GameRules: rules})
}

// SyncGameRules sends the stored values of all Bedrock game rules when a
// player joins, including the values used by the Settings screen.
func (s *Session) SyncGameRules(w *world.World) {
	if w == nil {
		return
	}
	values := w.GameRules()
	rules := make([]protocol.GameRule, 0, len(values))
	for _, definition := range world.GameRuleDefinitions() {
		value, ok := values[definition.Name]
		if !ok {
			continue
		}
		var packetValue any
		switch v := value.(type) {
		case bool:
			packetValue = v
		case uint32:
			packetValue = v
		case string:
			switch v {
			case "everyone":
				packetValue = uint32(0)
			case "off":
				packetValue = uint32(1)
			}
		}
		if packetValue == nil {
			continue
		}
		rules = append(rules, protocol.GameRule{
			Name:                  definition.Name,
			CanBeModifiedByPlayer: definition.CanBeModifiedByPlayer,
			Value:                 packetValue,
		})
	}
	s.SendGameRules(rules)
}
