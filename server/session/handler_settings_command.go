package session

import (
	"strings"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// SettingsCommandHandler executes commands issued by Bedrock's Settings UI.
//
// Commands are delegated to Controllable.ExecuteCommand so they use the same
// command registry, source permissions, and operator restrictions as normal
// chat commands.
type SettingsCommandHandler struct{}

// Handle implements packetHandler.
func (SettingsCommandHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, c Controllable) error {
	pk := p.(*packet.SettingsCommand)
	line := strings.TrimSpace(pk.CommandLine)
	if line == "" {
		return nil
	}

	origin, ok := s.handlers[packet.IDCommandRequest].(*CommandRequestHandler)
	if !ok {
		// Never execute a settings command outside the regular command pipeline.
		return nil
	}

	previousOrigin := origin.origin
	origin.origin = protocol.CommandOrigin{
		Origin: protocol.CommandOriginPlayer,
		UUID:   c.UUID(),
	}
	s.suppressCommandOutput.Store(pk.SuppressOutput)
	defer func() {
		s.suppressCommandOutput.Store(false)
		origin.origin = previousOrigin
	}()

	c.ExecuteCommand(line)
	return nil
}
