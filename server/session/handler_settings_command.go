package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// SettingsCommandHandler executes server-backed Bedrock settings commands
// through the normal player command pipeline, preserving command permissions.
type SettingsCommandHandler struct{}

// Handle ...
func (*SettingsCommandHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, c Controllable) error {
	pk := p.(*packet.SettingsCommand)
	if pk.CommandLine == "" {
		return nil
	}
	h := s.handlers[packet.IDCommandRequest].(*CommandRequestHandler)
	h.origin = protocol.CommandOrigin{
		Origin: protocol.CommandOriginPlayer,
		UUID:   uuid.New(),
	}
	s.suppressCommandOutput.Store(pk.SuppressOutput)
	defer s.suppressCommandOutput.Store(false)
	c.ExecuteCommand(pk.CommandLine)
	return nil
}
