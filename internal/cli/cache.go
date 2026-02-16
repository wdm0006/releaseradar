package cli

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
	"github.com/wdm0006/releaseradar/internal/cache"
	"github.com/wdm0006/releaseradar/internal/config"
	"github.com/wdm0006/releaseradar/internal/github"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the local release cache",
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete the local cache file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cache.Clear(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
		fmt.Println("Cache cleared.")
		return nil
	},
}

var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache info",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cache.Load()
		if err != nil {
			return fmt.Errorf("failed to load cache: %w", err)
		}
		if len(c.Releases) == 0 {
			fmt.Println("Cache is empty.")
			return nil
		}
		repos := make(map[string]int)
		for _, r := range c.Releases {
			repos[r.Repo]++
		}
		fmt.Printf("Releases: %d\n", len(c.Releases))
		fmt.Printf("Repos:    %d\n", len(repos))
		if !c.LastFetched.IsZero() {
			fmt.Printf("Fetched:  %s\n", c.LastFetched.Format(time.RFC3339))
		}
		for repo, count := range repos {
			fmt.Printf("  %-40s %d releases\n", repo, count)
		}
		return nil
	},
}

var cacheWarmCmd = &cobra.Command{
	Use:   "warm",
	Short: "Fetch releases for all tracked repos and populate the cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if len(cfg.Repos) == 0 {
			fmt.Println("No repos configured. Use 'releaseradar add owner/repo' first.")
			return nil
		}

		existing, err := cache.Load()
		if err != nil {
			return fmt.Errorf("failed to load cache: %w", err)
		}

		fmt.Printf("Fetching releases for %d repos...\n", len(cfg.Repos))

		type result struct {
			releases []github.Release
			err      string
		}

		var (
			mu      sync.Mutex
			wg      sync.WaitGroup
			done    atomic.Int32
			results = make([]result, len(cfg.Repos))
		)

		sem := make(chan struct{}, 8)
		for i, repo := range cfg.Repos {
			wg.Add(1)
			go func(idx int, r string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				releases, err := github.FetchReleases(r)
				if err != nil {
					changelog, clErr := github.FetchChangelog(r)
					if clErr == nil && changelog != "" {
						fallbackDate, _ := github.FetchChangelogCommitDate(r)
						entries := github.ParseChangelogEntries(changelog, r, fallbackDate)
						if len(entries) > 0 {
							mu.Lock()
							results[idx] = result{releases: entries}
							mu.Unlock()
							n := done.Add(1)
							fmt.Printf("  [%d/%d] %s (%d releases)\n", n, len(cfg.Repos), r, len(entries))
							return
						}
					}
					mu.Lock()
					results[idx] = result{err: err.Error()}
					mu.Unlock()
					n := done.Add(1)
					fmt.Printf("  [%d/%d] %s (error: %s)\n", n, len(cfg.Repos), r, err)
					return
				}

				mu.Lock()
				results[idx] = result{releases: releases}
				mu.Unlock()
				n := done.Add(1)
				fmt.Printf("  [%d/%d] %s (%d releases)\n", n, len(cfg.Repos), r, len(releases))
			}(i, repo)
		}
		wg.Wait()

		var allReleases []github.Release
		var errCount int
		for _, r := range results {
			if r.err != "" {
				errCount++
			} else {
				allReleases = append(allReleases, r.releases...)
			}
		}

		merged := cache.Merge(existing.Releases, allReleases)
		c := &cache.Cache{
			Releases:    merged,
			LastFetched: time.Now().UTC(),
		}
		if err := c.Save(cfg.Repos); err != nil {
			return fmt.Errorf("failed to save cache: %w", err)
		}

		fmt.Printf("Cache warmed: %d releases", len(merged))
		if errCount > 0 {
			fmt.Printf(" (%d errors)", errCount)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheWarmCmd)
}
