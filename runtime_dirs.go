package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var runtimeDirectories = [...]string{"plugins", "mods"}

// ensureRuntimeDirectories creates the standard runtime directories if needed.
// Their contents are intentionally not scanned, imported, or executed.
func ensureRuntimeDirectories(root string) error {
	for _, name := range runtimeDirectories {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create runtime directory %q: %w", name, err)
		}
	}
	return nil
}
