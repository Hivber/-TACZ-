package cmd

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

func TestParseDifficultyName(t *testing.T) {
	tests := []struct {
		input string
		want  world.Difficulty
	}{
		{"0", world.DifficultyPeaceful},
		{"peaceful", world.DifficultyPeaceful},
		{"1", world.DifficultyEasy},
		{"easy", world.DifficultyEasy},
		{"2", world.DifficultyNormal},
		{"normal", world.DifficultyNormal},
		{"3", world.DifficultyHard},
		{"hard", world.DifficultyHard},
	}
	for _, test := range tests {
		got, ok := parseDifficultyName(test.input)
		if !ok || got != test.want {
			t.Errorf("parseDifficultyName(%q) = (%v, %t), want (%v, true)", test.input, got, ok, test.want)
		}
	}
	if _, ok := parseDifficultyName("impossible"); ok {
		t.Fatal("parseDifficultyName accepted an invalid difficulty")
	}
}

func TestParseGameModeName(t *testing.T) {
	tests := []struct {
		input string
		want  world.GameMode
	}{
		{"0", world.GameModeSurvival},
		{"survival", world.GameModeSurvival},
		{"1", world.GameModeCreative},
		{"creative", world.GameModeCreative},
		{"2", world.GameModeAdventure},
		{"adventure", world.GameModeAdventure},
		{"3", world.GameModeSpectator},
		{"spectator", world.GameModeSpectator},
	}
	for _, test := range tests {
		got, ok := parseGameModeName(test.input)
		if !ok || got != test.want {
			t.Errorf("parseGameModeName(%q) = (%v, %t), want (%v, true)", test.input, got, ok, test.want)
		}
	}
	if _, ok := parseGameModeName("unknown"); ok {
		t.Fatal("parseGameModeName accepted an invalid game mode")
	}
}

func TestVanillaSettingsCommandsRegistered(t *testing.T) {
	for _, name := range []string{"difficulty", "gamemode", "defaultgamemode", "gm"} {
		if _, ok := ByAlias(name); !ok {
			t.Errorf("command %q was not registered", name)
		}
	}
}
