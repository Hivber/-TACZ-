package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRuntimeDirectories(t *testing.T) {
	root := t.TempDir()
	if err := ensureRuntimeDirectories(root); err != nil {
		t.Fatalf("ensureRuntimeDirectories() error = %v", err)
	}

	for _, name := range runtimeDirectories {
		info, err := os.Stat(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s exists but is not a directory", name)
		}
	}
}

func TestEnsureRuntimeDirectoriesIsIdempotentAndLeavesContentsUntouched(t *testing.T) {
	root := t.TempDir()
	pluginFile := filepath.Join(root, "plugins", "example.txt")
	if err := os.MkdirAll(filepath.Dir(pluginFile), 0o755); err != nil {
		t.Fatal(err)
	}
	const content = "not loaded or executed"
	if err := os.WriteFile(pluginFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ensureRuntimeDirectories(root); err != nil {
		t.Fatalf("first ensureRuntimeDirectories() error = %v", err)
	}
	if err := ensureRuntimeDirectories(root); err != nil {
		t.Fatalf("second ensureRuntimeDirectories() error = %v", err)
	}
	got, err := os.ReadFile(pluginFile)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if string(got) != content {
		t.Fatalf("existing file content = %q, want %q", got, content)
	}
}
