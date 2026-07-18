package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Repos) != 0 {
		t.Fatalf("expected empty repos, got %v", cfg.Repos)
	}
}

func TestAddRemoveRepo(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	cfg, _ := Load()

	if !cfg.AddRepo("owner/repo") {
		t.Fatal("AddRepo should return true for new repo")
	}
	if cfg.AddRepo("owner/repo") {
		t.Fatal("AddRepo should return false for duplicate")
	}
	if !cfg.HasRepo("owner/repo") {
		t.Fatal("HasRepo should return true")
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}

	if !cfg.RemoveRepo("owner/repo") {
		t.Fatal("RemoveRepo should return true for existing repo")
	}
	if cfg.RemoveRepo("owner/repo") {
		t.Fatal("RemoveRepo should return false for missing repo")
	}
	if cfg.HasRepo("owner/repo") {
		t.Fatal("HasRepo should return false after remove")
	}
}

func TestSaveLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	cfg := &Config{Repos: []string{"python/cpython", "golang/go"}}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists at XDG path
	path := configPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(loaded.Repos))
	}
	if loaded.Repos[0] != "python/cpython" || loaded.Repos[1] != "golang/go" {
		t.Fatalf("unexpected repos: %v", loaded.Repos)
	}
}

func TestLoadCorruptFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	// Write corrupt JSON to XDG path
	dir := configDir()
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should error on corrupt file")
	}
	if !strings.Contains(err.Error(), "invalid config file") {
		t.Fatalf("expected invalid config error, got %v", err)
	}
	if strings.Contains(err.Error(), "not json") {
		t.Fatalf("error exposes config contents: %v", err)
	}
}

func TestLoadLegacyPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	// Write to legacy path
	legacyPath := filepath.Join(tmp, ".releaseradar.json")
	os.WriteFile(legacyPath, []byte(`{"repos":["owner/legacy"]}`), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Repos) != 1 || cfg.Repos[0] != "owner/legacy" {
		t.Fatalf("expected legacy repo, got %v", cfg.Repos)
	}
}
