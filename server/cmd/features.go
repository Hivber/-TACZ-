package cmd

import "sync"

var (
	javaCompatibilityCommandsOnce sync.Once
	performanceDiagnosticsOnce    sync.Once
)

// RegisterJavaCompatibilityCommands registers optional Java-only command
// extensions. Vanilla Bedrock commands remain registered independently.
func RegisterJavaCompatibilityCommands() {
	javaCompatibilityCommandsOnce.Do(func() {
		Register(New("item", "Replaces an item in a player's hand, armour or inventory slot.", nil, itemCommand{}))
		Register(New("attribute", "Queries or changes supported player attributes.", nil, attributeCommand{}))
		Register(New("bossbar", "Displays a boss bar for one or more players.", nil, bossbarCommand{}))
		Register(New("bossbarclear", "Hides the current boss bar for one or more players.", nil, bossbarClearCommand{}))
		Register(New("data", "Reads and modifies NBT data on supported blocks and reads entity NBT.", nil, dataGetBlockCommand{}, dataGetEntityCommand{}, dataMergeBlockCommand{}, dataModifyBlockCommand{}, dataRemoveBlockCommand{}))
		Register(New("forceload", "Keeps selected chunks loaded while the server is running.", nil,
			forceloadAddCommand{}, forceloadRemoveCommand{}, forceloadRemoveAllCommand{}, forceloadQueryCommand{}))
		Register(New("random", "Generates a random integer, optionally announcing the result.", nil, randomCommand{}))
		Register(New("worldborder", "Queries and changes the current world's horizontal movement border.", nil,
			worldBorderGetCommand{}, worldBorderSetCommand{}, worldBorderAddCommand{}, worldBorderCenterCommand{}))
	})
}

// RegisterPerformanceDiagnostics registers the optional /perf command.
func RegisterPerformanceDiagnostics() {
	performanceDiagnosticsOnce.Do(func() {
		Register(New("perf", "Shows live server performance diagnostics.", nil, perfCommand{}))
	})
}
