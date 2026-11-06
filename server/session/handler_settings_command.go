package session

import (
	"strings"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// SettingsCommandHandler routes Bedrock's settings-originated commands through
// the same player command pipeline used by ordinary CommandRequest packets.
// This preserves Dragonfly command permissions, event hooks and OP checks.
type SettingsCommandHandler struct{}

// Handle executes a command issued by the Bedrock settings screen.
func (h *SettingsCommandHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, c Controllable) error {
	pk := p.(*packet.SettingsCommand)
	line := strings.TrimSpace(pk.CommandLine)
	if line == "" {
		return nil
	}
	if !strings.HasPrefix(line, "/") {
		line = "/" + line
	}

	// SettingsCommand has no CommandOrigin field. Supply a player origin so
	// command output, when not suppressed by the client, has a valid request ID.
	origin := s.handlers[packet.IDCommandRequest].(*CommandRequestHandler)
	previousOrigin := origin.origin
	origin.origin = protocol.CommandOrigin{Origin: protocol.CommandOriginPlayer, UUID: uuid.New()}
	defer func() { origin.origin = previousOrigin }()

	previousSuppress := s.suppressCommandOutput.Swap(pk.SuppressOutput)
	defer s.suppressCommandOutput.Store(previousSuppress)

	// ExecuteCommand dispatches through Dragonfly's normal command registry.
	// It does not grant permissions; each command still checks its own source.
	c.ExecuteCommand(line)
	return nil
}
