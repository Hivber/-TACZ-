package cmd

import (
	"strings"

	"github.com/df-mc/dragonfly/server/world"
)

func init() {
	Register(New("difficulty", "Sets or queries the world difficulty.", nil, difficultyQuery{}, difficultySet{}))
	Register(New("gamemode", "Sets or queries a player's game mode.", []string{"gm"}, gameModeQuery{}, gameModeSet{}))
	Register(New("defaultgamemode", "Sets or queries the world's default game mode.", []string{"defaultgm"}, defaultGameModeQuery{}, defaultGameModeSet{}))
}

type difficultyName string

func (difficultyName) Type() string { return "Difficulty" }
func (difficultyName) Options(Source) []string {
	return []string{"peaceful", "easy", "normal", "hard", "0", "1", "2", "3", "p", "e", "n", "h"}
}

func parseDifficultyName(name string) (world.Difficulty, bool) {
	switch strings.ToLower(name) {
	case "0", "peaceful", "p":
		return world.DifficultyPeaceful, true
	case "1", "easy", "e":
		return world.DifficultyEasy, true
	case "2", "normal", "n":
		return world.DifficultyNormal, true
	case "3", "hard", "h":
		return world.DifficultyHard, true
	default:
		return nil, false
	}
}

type difficultyQuery struct{}

func (difficultyQuery) Allow(src Source) bool { return commandOperator(src) }

func (difficultyQuery) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("difficulty requires a world.")
		return
	}
	id, _ := world.DifficultyID(tx.World().Difficulty())
	names := [...]string{"peaceful", "easy", "normal", "hard"}
	if id < 0 || id >= len(names) {
		out.Errorf("Current world difficulty has an invalid ID: %d.", id)
		return
	}
	out.Printf("Difficulty: %s (%d)", names[id], id)
}

type difficultySet struct {
	Difficulty difficultyName
}

func (difficultySet) Allow(src Source) bool { return commandOperator(src) }

func (c difficultySet) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("difficulty requires a world.")
		return
	}
	difficulty, ok := parseDifficultyName(string(c.Difficulty))
	if !ok {
		out.Errorf("Unknown difficulty: %s", c.Difficulty)
		return
	}
	tx.World().SetDifficulty(difficulty)
	for entity := range tx.Players() {
		if player, ok := entity.(interface{ SendDifficulty(world.Difficulty) }); ok {
			player.SendDifficulty(difficulty)
		}
	}
	id, _ := world.DifficultyID(difficulty)
	out.Printf("The world difficulty has been updated to %s (%d).", c.Difficulty, id)
}

type gameModeName string

func (gameModeName) Type() string { return "GameMode" }
func (gameModeName) Options(Source) []string {
	return []string{"survival", "creative", "adventure", "spectator", "default", "0", "1", "2", "5", "s", "c", "a", "d", "sp"}
}

func parseGameModeName(name string) (world.GameMode, bool) {
	switch strings.ToLower(name) {
	case "0", "survival", "s":
		return world.GameModeSurvival, true
	case "1", "creative", "c":
		return world.GameModeCreative, true
	case "2", "adventure", "a":
		return world.GameModeAdventure, true
	case "spectator", "sp", "view":
		return world.GameModeSpectator, true
	default:
		return nil, false
	}
}

type gameModeTarget interface {
	Target
	Name() string
	GameMode() world.GameMode
	SetGameMode(world.GameMode)
}

type gameModeQuery struct{}

func (gameModeQuery) Allow(src Source) bool { return commandOperator(src) }

func (gameModeQuery) Run(src Source, out *Output, tx *world.Tx) {
	var mode world.GameMode
	if target, ok := src.(gameModeTarget); ok {
		mode = target.GameMode()
	} else if tx != nil {
		mode = tx.World().DefaultGameMode()
	} else {
		out.Error("gamemode requires a player or a world.")
		return
	}
	id, ok := world.GameModeID(mode)
	if !ok {
		out.Error("The current game mode is not registered.")
		return
	}
	names := [...]string{"survival", "creative", "adventure", "spectator"}
	if id < 0 || id >= len(names) {
		out.Errorf("Current game mode has an invalid ID: %d.", id)
		return
	}
	out.Printf("Game mode: %s (%d)", names[id], id)
}

type gameModeSet struct {
	Mode    gameModeName
	Targets Optional[[]Target]
}

func (gameModeSet) Allow(src Source) bool { return commandOperator(src) }

func (c gameModeSet) Run(src Source, out *Output, tx *world.Tx) {
	var mode world.GameMode
	var ok bool
	switch strings.ToLower(string(c.Mode)) {
	case "default", "d", "5":
		if tx == nil {
			out.Error("gamemode default requires a world.")
			return
		}
		mode, ok = tx.World().DefaultGameMode(), true
	default:
		mode, ok = parseGameModeName(string(c.Mode))
	}
	if !ok {
		out.Errorf("Unknown game mode: %s", c.Mode)
		return
	}
	targets, supplied := c.Targets.Load()
	if !supplied {
		target, ok := src.(gameModeTarget)
		if !ok {
			out.Error("A player target is required when gamemode is run from the console.")
			return
		}
		targets = []Target{target}
	}
	changed := 0
	for _, raw := range targets {
		target, ok := raw.(gameModeTarget)
		if !ok {
			continue
		}
		target.SetGameMode(mode)
		changed++
	}
	if changed == 0 {
		out.Error("No player targets matched.")
		return
	}
	id, _ := world.GameModeID(mode)
	out.Printf("Set game mode to %s (%d) for %d player(s).", c.Mode, id, changed)
}

type defaultGameModeQuery struct{}

func (defaultGameModeQuery) Allow(src Source) bool { return commandOperator(src) }

func (defaultGameModeQuery) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("defaultgamemode requires a world.")
		return
	}
	id, _ := world.GameModeID(tx.World().DefaultGameMode())
	names := [...]string{"survival", "creative", "adventure", "spectator"}
	if id < 0 || id >= len(names) {
		out.Errorf("Default game mode has an invalid ID: %d.", id)
		return
	}
	out.Printf("Default game mode: %s (%d)", names[id], id)
}

type defaultGameModeSet struct {
	Mode gameModeName
}

func (defaultGameModeSet) Allow(src Source) bool { return commandOperator(src) }

func (c defaultGameModeSet) Run(_ Source, out *Output, tx *world.Tx) {
	if tx == nil {
		out.Error("defaultgamemode requires a world.")
		return
	}
	mode, ok := parseGameModeName(string(c.Mode))
	if !ok {
		out.Errorf("Unknown game mode: %s", c.Mode)
		return
	}
	tx.World().SetDefaultGameMode(mode)
	id, _ := world.GameModeID(mode)
	out.Printf("Default game mode has been updated to %s (%d).", c.Mode, id)
}
