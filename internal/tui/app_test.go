package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wdm0006/releaseradar/internal/cache"
	"github.com/wdm0006/releaseradar/internal/config"
	"github.com/wdm0006/releaseradar/internal/github"
)

// isolateConfig points config.Save/config.Load at a temporary directory.
func isolateConfig(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
}

// configFilePath mirrors the path config.Save writes to.
func configFilePath(t *testing.T) string {
	t.Helper()
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir() error: %v", err)
	}
	return filepath.Join(dir, "releaseradar", "config.json")
}

// breakConfigSave makes config.Save fail by putting a directory where the
// config file belongs, so the write cannot succeed. The returned function
// restores a writable config path.
func breakConfigSave(t *testing.T) func() {
	t.Helper()
	path := configFilePath(t)
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error: %v", path, err)
	}
	return func() {
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("RemoveAll(%q) error: %v", path, err)
		}
	}
}

func newTestModel(repos []string, releases []github.Release) Model {
	cfg := &config.Config{Repos: repos}
	return NewModel(cfg, &cache.Cache{Releases: releases})
}

func updateModel(t *testing.T, m Model, msg interface{}) Model {
	t.Helper()
	next, _ := m.Update(msg)
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	return updated
}

func loadPersistedRepos(t *testing.T) []string {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error: %v", err)
	}
	return cfg.Repos
}

func TestRepoAddedSaveFailureLeavesConfigUnchanged(t *testing.T) {
	isolateConfig(t)
	defer breakConfigSave(t)()

	m := newTestModel([]string{"owner/one"}, nil)
	m.showModal = true

	m = updateModel(t, m, repoAddedMsg{repo: "owner/two"})

	if got := m.cfg.Repos; len(got) != 1 || got[0] != "owner/one" {
		t.Fatalf("live repos changed after failed save: %v", got)
	}
	if !strings.HasPrefix(m.status, "Error saving config:") {
		t.Fatalf("status = %q, want an error saving config", m.status)
	}
	if m.refreshing {
		t.Fatal("failed add should not start a refresh")
	}
}

func TestRepoRemovedSaveFailureLeavesConfigUnchanged(t *testing.T) {
	isolateConfig(t)
	defer breakConfigSave(t)()

	releases := []github.Release{{Repo: "owner/one", TagName: "v1"}}
	m := newTestModel([]string{"owner/one", "owner/two"}, releases)

	m = updateModel(t, m, repoRemovedMsg{repo: "owner/two"})

	want := []string{"owner/one", "owner/two"}
	if got := m.cfg.Repos; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("live repos changed after failed save: %v", got)
	}
	if !strings.HasPrefix(m.status, "Error saving config:") {
		t.Fatalf("status = %q, want an error saving config", m.status)
	}
	if len(m.allReleases) != 1 {
		t.Fatalf("failed removal pruned releases: %v", m.allReleases)
	}
}

func TestFailedMutationDoesNotLeakIntoLaterSave(t *testing.T) {
	isolateConfig(t)
	restore := breakConfigSave(t)

	m := newTestModel([]string{"owner/one", "owner/two"}, nil)
	m = updateModel(t, m, repoAddedMsg{repo: "owner/rejected"})
	if !strings.HasPrefix(m.status, "Error saving config:") {
		t.Fatalf("status = %q, want an error saving config", m.status)
	}

	restore()

	m = updateModel(t, m, repoRemovedMsg{repo: "owner/two"})
	if m.status != "Removed owner/two" {
		t.Fatalf("status = %q, want removal confirmation", m.status)
	}
	if got := m.cfg.Repos; len(got) != 1 || got[0] != "owner/one" {
		t.Fatalf("live repos = %v, want only owner/one", got)
	}
	if got := loadPersistedRepos(t); len(got) != 1 || got[0] != "owner/one" {
		t.Fatalf("persisted repos = %v, want only owner/one", got)
	}
}

func TestRepoAddedSuccessPersistsAndRefreshes(t *testing.T) {
	isolateConfig(t)

	m := newTestModel([]string{"owner/one"}, nil)
	m.showModal = true

	next, cmd := m.Update(repoAddedMsg{repo: "owner/two"})
	m, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}

	if got := m.cfg.Repos; len(got) != 2 || got[1] != "owner/two" {
		t.Fatalf("live repos = %v, want owner/one and owner/two", got)
	}
	if got := loadPersistedRepos(t); len(got) != 2 || got[1] != "owner/two" {
		t.Fatalf("persisted repos = %v, want owner/one and owner/two", got)
	}
	if m.showModal {
		t.Fatal("modal should be closed after a successful add")
	}
	if !m.refreshing {
		t.Fatal("successful add should start a refresh")
	}
	if m.status != "Added owner/two, refreshing..." {
		t.Fatalf("status = %q, want the add confirmation", m.status)
	}
	if cmd == nil {
		t.Fatal("successful add should return a fetch command")
	}
}

func TestRepoRemovedSuccessPrunesReleases(t *testing.T) {
	isolateConfig(t)

	releases := []github.Release{
		{Repo: "owner/one", TagName: "v1"},
		{Repo: "owner/two", TagName: "v2"},
	}
	m := newTestModel([]string{"owner/one", "owner/two"}, releases)

	next, cmd := m.Update(repoRemovedMsg{repo: "owner/two"})
	m, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}

	if got := m.cfg.Repos; len(got) != 1 || got[0] != "owner/one" {
		t.Fatalf("live repos = %v, want only owner/one", got)
	}
	if got := loadPersistedRepos(t); len(got) != 1 || got[0] != "owner/one" {
		t.Fatalf("persisted repos = %v, want only owner/one", got)
	}
	if len(m.allReleases) != 1 || m.allReleases[0].Repo != "owner/one" {
		t.Fatalf("allReleases = %v, want only owner/one records", m.allReleases)
	}
	if len(m.trackedRepos) != 1 || m.trackedRepos[0] != "owner/one" {
		t.Fatalf("trackedRepos = %v, want only owner/one", m.trackedRepos)
	}
	if m.status != "Removed owner/two" {
		t.Fatalf("status = %q, want the removal confirmation", m.status)
	}
	if cmd == nil {
		t.Fatal("successful removal should return a cache-save command")
	}
}
