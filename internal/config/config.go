package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Repos []string `json:"repos"`
}

func configDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "releaseradar")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "releaseradar")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func legacyConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".releaseradar.json"
	}
	return filepath.Join(home, ".releaseradar.json")
}

func Load() (*Config, error) {
	// Try XDG path first
	path := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Fall back to legacy path
		path = legacyConfigPath()
		data, err = os.ReadFile(path)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Repos: []string{}}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file %q: %w", path, err)
	}
	if cfg.Repos == nil {
		cfg.Repos = []string{}
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

// Clone returns a deep copy so callers can stage a mutation and only adopt it
// once the corresponding Save succeeds.
func (c *Config) Clone() *Config {
	repos := make([]string, len(c.Repos))
	copy(repos, c.Repos)
	return &Config{Repos: repos}
}

func (c *Config) AddRepo(repo string) bool {
	for _, r := range c.Repos {
		if r == repo {
			return false
		}
	}
	c.Repos = append(c.Repos, repo)
	return true
}

func (c *Config) RemoveRepo(repo string) bool {
	for i, r := range c.Repos {
		if r == repo {
			c.Repos = append(c.Repos[:i], c.Repos[i+1:]...)
			return true
		}
	}
	return false
}

func (c *Config) HasRepo(repo string) bool {
	for _, r := range c.Repos {
		if r == repo {
			return true
		}
	}
	return false
}
