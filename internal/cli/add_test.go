package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolateConfig points config.Load/Save at a temp directory and returns the
// config file path. macOS ignores XDG_CONFIG_HOME, so both vars are set.
func isolateConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	configHome, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error: %v", err)
	}
	return filepath.Join(configHome, "releaseradar", "config.json")
}

func storedRepos(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	var cfg struct {
		Repos []string `json:"repos"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	return cfg.Repos
}

func TestAddStoresCanonicalIdentifier(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{"canonical", "owner/repo", "owner/repo"},
		{"surrounding whitespace", "  owner/repo  ", "owner/repo"},
		{"github url", "https://github.com/owner/repo", "owner/repo"},
		{"clone url", "https://github.com/owner/repo.git", "owner/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := isolateConfig(t)

			if err := addCmd.RunE(addCmd, []string{tt.arg}); err != nil {
				t.Fatalf("add %q error: %v", tt.arg, err)
			}

			got := storedRepos(t, path)
			if len(got) != 1 || got[0] != tt.want {
				t.Fatalf("add %q stored %q, want [%q]", tt.arg, got, tt.want)
			}
		})
	}
}

func TestAddRejectsInvalidIdentifiers(t *testing.T) {
	for _, arg := range []string{
		"/repo",
		"owner/",
		"owner/repo/tree/main",
		"owner/repo?per_page=1&foo=",
		"../../users/octocat",
	} {
		t.Run(arg, func(t *testing.T) {
			path := isolateConfig(t)

			err := addCmd.RunE(addCmd, []string{arg})
			if err == nil {
				t.Fatalf("add %q should fail", arg)
			}
			if !strings.Contains(err.Error(), "owner/repo") {
				t.Fatalf("add %q error %q does not name the expected format", arg, err)
			}
			if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
				t.Fatalf("add %q wrote a config file (stat err: %v, repos: %q)", arg, statErr, storedRepos(t, path))
			}
		})
	}
}

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
