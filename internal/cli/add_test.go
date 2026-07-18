package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAddDoesNotOverwriteMalformedConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	configHome, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error: %v", err)
	}
	path := filepath.Join(configHome, "releaseradar", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	original := []byte(`{"repos":["owner/existing"],broken}`)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	if err := addCmd.RunE(addCmd, []string{"owner/new"}); err == nil {
		t.Fatal("add should fail when the config is malformed")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("malformed config was overwritten: got %q", got)
	}
}
