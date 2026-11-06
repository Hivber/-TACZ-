package player

import "github.com/sandertv/gophertunnel/minecraft/protocol"

// SendGameRules forwards changed game rules to the player's session.
func (p *Player) SendGameRules(rules []protocol.GameRule) {
	p.session().SendGameRules(rules)
}
