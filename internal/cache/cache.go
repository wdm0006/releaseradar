package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/wdm0006/releaseradar/internal/github"
)

type Cache struct {
	Releases    []github.Release `json:"releases"`
	LastFetched time.Time        `json:"last_fetched"`
}

func cacheDir() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "releaseradar")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "releaseradar")
}

func cachePath() string {
	return filepath.Join(cacheDir(), "cache.json")
}

func legacyCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".releaseradar-cache.json"
	}
	return filepath.Join(home, ".releaseradar-cache.json")
}

func Load() (*Cache, error) {
	// Try XDG path first
	path := cachePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Fall back to legacy path
		data, err = os.ReadFile(legacyCachePath())
	}
	if err != nil {
		if os.IsNotExist(err) {
			return &Cache{}, nil
		}
		return nil, err
	}

	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return &Cache{}, nil
	}
	return &c, nil
}

// Save writes the cache to disk, pruning releases for repos no longer tracked.
// This is safe to call from a goroutine as long as the Cache value is not shared.
func (c *Cache) Save(trackedRepos []string) error {
	tracked := make(map[string]bool, len(trackedRepos))
	for _, r := range trackedRepos {
		tracked[r] = true
	}
	var pruned []github.Release
	for _, rel := range c.Releases {
		if tracked[rel.Repo] {
			pruned = append(pruned, rel)
		}
	}
	c.Releases = pruned

	dir := cacheDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath(), data, 0644)
}

// Clear removes the cache file from disk.
func Clear() error {
	path := cachePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Also remove legacy path if present
	legacy := legacyCachePath()
	if err := os.Remove(legacy); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Merge combines cached and fresh releases. Fresh wins on conflict (same repo+tag_name).
func Merge(cached, fresh []github.Release) []github.Release {
	seen := make(map[string]int) // key -> index in result
	result := make([]github.Release, 0, len(cached)+len(fresh))

	// Add cached first
	for _, rel := range cached {
		key := rel.Repo + "\x00" + rel.TagName
		seen[key] = len(result)
		result = append(result, rel)
	}

	// Fresh wins on conflict
	for _, rel := range fresh {
		key := rel.Repo + "\x00" + rel.TagName
		if idx, exists := seen[key]; exists {
			result[idx] = rel
		} else {
			seen[key] = len(result)
			result = append(result, rel)
		}
	}

	return result
}
