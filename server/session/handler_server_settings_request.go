package session

import (
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ServerSettingsRequestHandler exposes server-controlled gameplay settings in
// Bedrock's dedicated server settings tab.
type ServerSettingsRequestHandler struct{}

// Handle sends a form whose selections are executed through the normal command
// dispatcher. Command permissions and server-side state changes therefore stay
// identical to commands entered in chat.
func (ServerSettingsRequestHandler) Handle(_ packet.Packet, s *Session, tx *world.Tx, c Controllable) error {
	if tx == nil {
		return nil
	}
	modeIndex := 0
	if id, ok := world.GameModeID(c.GameMode()); ok && id >= 0 && id < 4 {
		modeIndex = id
	}
	difficultyIndex := 2
	if id, ok := world.DifficultyID(tx.World().Difficulty()); ok && id >= 0 && id < 4 {
		difficultyIndex = id
	}
	s.SendServerSettingsForm(form.New(serverGameplaySettings{
		GameMode: form.NewDropdown("游戏模式", []string{"生存", "创造", "冒险", "旁观者"}, modeIndex),
		Difficulty: form.NewDropdown("世界难度", []string{"和平", "简单", "普通", "困难"}, difficultyIndex),
	}, "服务器游戏设置"))
	return nil
}

type serverGameplaySettings struct {
	GameMode    form.Dropdown
	Difficulty form.Dropdown
}

func (settings serverGameplaySettings) Submit(submitter form.Submitter, _ *world.Tx) {
	executor, ok := submitter.(interface{ ExecuteCommand(string) })
	if !ok {
		return
	}
	gameModes := [...]string{"survival", "creative", "adventure", "spectator"}
	difficulties := [...]string{"peaceful", "easy", "normal", "hard"}
	if index := settings.GameMode.Value(); index >= 0 && index < len(gameModes) {
		executor.ExecuteCommand("/gamemode " + gameModes[index])
	}
	if index := settings.Difficulty.Value(); index >= 0 && index < len(difficulties) {
		executor.ExecuteCommand("/difficulty " + difficulties[index])
	}
}
