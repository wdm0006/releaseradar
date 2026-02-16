package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wdm0006/releaseradar/internal/github"
)

func TestLoadEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(c.Releases) != 0 {
		t.Fatalf("expected empty releases, got %d", len(c.Releases))
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	c := &Cache{
		Releases: []github.Release{
			{Repo: "owner/repo", TagName: "v1.0.0", Name: "Release 1"},
			{Repo: "owner/repo", TagName: "v2.0.0", Name: "Release 2"},
		},
	}

	if err := c.Save([]string{"owner/repo"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists at XDG path
	path := cachePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("cache file not created")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded.Releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(loaded.Releases))
	}
	if loaded.Releases[0].TagName != "v1.0.0" {
		t.Fatalf("unexpected tag: %s", loaded.Releases[0].TagName)
	}
}

func TestSavePrunesUntrackedRepos(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	c := &Cache{
		Releases: []github.Release{
			{Repo: "owner/keep", TagName: "v1.0.0"},
			{Repo: "owner/remove", TagName: "v1.0.0"},
			{Repo: "owner/keep", TagName: "v2.0.0"},
		},
	}

	if err := c.Save([]string{"owner/keep"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if len(c.Releases) != 2 {
		t.Fatalf("expected 2 releases after prune, got %d", len(c.Releases))
	}
	for _, rel := range c.Releases {
		if rel.Repo != "owner/keep" {
			t.Fatalf("unexpected repo after prune: %s", rel.Repo)
		}
	}
}

func TestLoadCorruptFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir := cacheDir()
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "cache.json")
	os.WriteFile(path, []byte("not json"), 0644)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error on corrupt file: %v", err)
	}
	if len(c.Releases) != 0 {
		t.Fatalf("expected empty releases for corrupt file, got %d", len(c.Releases))
	}
}

func TestLoadLegacyPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Write to legacy path
	legacyPath := filepath.Join(tmp, ".releaseradar-cache.json")
	os.WriteFile(legacyPath, []byte(`{"releases":[{"repo":"owner/old","tag_name":"v1.0.0"}]}`), 0644)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(c.Releases) != 1 || c.Releases[0].Repo != "owner/old" {
		t.Fatalf("expected legacy release, got %v", c.Releases)
	}
}

func TestMerge(t *testing.T) {
	cached := []github.Release{
		{Repo: "owner/a", TagName: "v1.0.0", Body: "old body"},
		{Repo: "owner/a", TagName: "v0.9.0", Body: "historical"},
	}
	fresh := []github.Release{
		{Repo: "owner/a", TagName: "v1.0.0", Body: "new body"},
		{Repo: "owner/a", TagName: "v1.1.0", Body: "brand new"},
	}

	merged := Merge(cached, fresh)

	if len(merged) != 3 {
		t.Fatalf("expected 3 merged releases, got %d", len(merged))
	}

	byTag := make(map[string]github.Release)
	for _, r := range merged {
		byTag[r.TagName] = r
	}

	if byTag["v1.0.0"].Body != "new body" {
		t.Fatal("expected fresh to win on conflict")
	}
	if byTag["v0.9.0"].Body != "historical" {
		t.Fatal("expected historical release to be preserved")
	}
	if byTag["v1.1.0"].Body != "brand new" {
		t.Fatal("expected new release to be added")
	}
}

func TestMergeEmpty(t *testing.T) {
	fresh := []github.Release{
		{Repo: "owner/a", TagName: "v1.0.0"},
	}

	merged := Merge(nil, fresh)
	if len(merged) != 1 {
		t.Fatalf("expected 1 release, got %d", len(merged))
	}

	merged = Merge(fresh, nil)
	if len(merged) != 1 {
		t.Fatalf("expected 1 release, got %d", len(merged))
	}
}
