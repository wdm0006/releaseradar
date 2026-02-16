package tui

import (
	"cmp"
	"slices"

	"github.com/wdm0006/releaseradar/internal/cache"
	"github.com/wdm0006/releaseradar/internal/config"
	"github.com/wdm0006/releaseradar/internal/github"
)

// RenderScreenshot creates a Model with the given config and cache, sizes it,
// and returns the rendered ANSI output for the specified tab. This is used
// to generate static screenshots without running the full TUI event loop.
func RenderScreenshot(cfg *config.Config, c *cache.Cache, tab string, width, height int) string {
	p := &loadingProgress{total: int32(len(cfg.Repos))}
	p.current.Store("")

	m := Model{
		cfg:          cfg,
		activeTab:    tabReleases,
		status:       "Loaded from cache",
		releases:     newReleasesModel(),
		repos:        newReposModel(cfg.Repos),
		summary:      newSummaryModel(),
		chat:         newChatModel(),
		addModal:     newAddModalModel(),
		loading:      false,
		progress:     p,
		trackedRepos: cfg.Repos,
		lastViewed:   c.LastFetched,
		lastFetched:  c.LastFetched,
	}

	switch tab {
	case "repos":
		m.activeTab = tabRepos
	case "loading":
		m.loading = true
		// Simulate mid-load state
		m.progress.total = int32(len(cfg.Repos))
		m.progress.done.Store(0)
		m.progress.current.Store("")
		m.width = width
		m.height = height
		return m.View()
	default:
		m.activeTab = tabReleases
	}

	// Populate with cached data
	if len(c.Releases) > 0 {
		releases := make([]github.Release, len(c.Releases))
		copy(releases, c.Releases)
		slices.SortFunc(releases, func(a, b github.Release) int {
			return cmp.Compare(b.PublishedAt, a.PublishedAt)
		})
		m.allReleases = releases
		m.releases = m.releases.setReleases(releases)
		m.repos = m.repos.setReleases(releases)
	}

	// Set the size (triggers layout in sub-models)
	m.width = width
	m.height = height
	contentHeight := height - 5
	m.releases = m.releases.setSize(width, contentHeight)
	m.repos = m.repos.setSize(width, contentHeight)
	m.summary = m.summary.setSize(width, contentHeight)
	m.chat = m.chat.setSize(width, contentHeight)

	return m.View()
}
