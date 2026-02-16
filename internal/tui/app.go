package tui

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wdm0006/releaseradar/internal/ai"
	"github.com/wdm0006/releaseradar/internal/cache"
	"github.com/wdm0006/releaseradar/internal/config"
	"github.com/wdm0006/releaseradar/internal/github"
)

type tabID int

const (
	tabReleases tabID = iota
	tabRepos
	tabSummary
	tabChat
)

var tabNames = []string{"Releases", "Repositories", "AI Summary", "Chat"}

// Messages
type releasesLoadedMsg struct {
	releases []github.Release
	errors   []string
}

type repoInfoLoadedMsg struct {
	repo string
	info github.RepoInfo
	err  error
}

type summaryLoadedMsg struct {
	content string
	err     error
}

type chatResponseMsg struct {
	response string
	err      error
}

type repoAddedMsg struct {
	repo string
}

type repoRemovedMsg struct {
	repo string
}

type statusMsg string

type errMsg struct{ error }

type cacheSavedMsg struct {
	err         error
	lastFetched time.Time
}

// tickMsg drives the loading screen animation
type tickMsg struct{}

// loadingProgress is shared between the fetch goroutine and the TUI
type loadingProgress struct {
	done    atomic.Int32
	total   int32
	current atomic.Value // stores string
}

// Model is the top-level Bubble Tea model
type Model struct {
	cfg       *config.Config
	activeTab tabID
	width     int
	height    int
	status    string

	releases  releasesModel
	repos     reposModel
	summary   summaryModel
	chat      chatModel
	addModal  addModalModel
	showModal bool

	allReleases  []github.Release
	trackedRepos []string
	lastFetched  time.Time
	lastViewed   time.Time

	// Loading state
	loading       bool
	refreshing    bool
	progress      *loadingProgress
	confirmRemove string
	noAltScreen   bool
}

func NewModel(cfg *config.Config, releaseCache *cache.Cache) Model {
	p := &loadingProgress{total: int32(len(cfg.Repos))}
	p.current.Store("")

	hasCached := len(releaseCache.Releases) > 0
	lastViewed := releaseCache.LastFetched

	startTab := tabReleases
	switch os.Getenv("RELEASERADAR_START_TAB") {
	case "2", "repos":
		startTab = tabRepos
	case "3", "summary":
		startTab = tabSummary
	case "4", "chat":
		startTab = tabChat
	}

	m := Model{
		cfg:          cfg,
		activeTab:    startTab,
		status:       "Loading...",
		releases:     newReleasesModel(),
		repos:        newReposModel(cfg.Repos),
		summary:      newSummaryModel(),
		chat:         newChatModel(),
		addModal:     newAddModalModel(),
		loading:      len(cfg.Repos) > 0 && !hasCached,
		progress:     p,
		trackedRepos: cfg.Repos,
		lastViewed:   lastViewed,
		lastFetched:  lastViewed,
		noAltScreen:  os.Getenv("RELEASERADAR_NO_ALTSCREEN") != "",
	}

	m.releases = m.releases.setLastViewed(lastViewed)

	if hasCached {
		sortReleases(releaseCache.Releases)
		m.allReleases = releaseCache.Releases
		m.releases = m.releases.setReleases(releaseCache.Releases)
		m.repos = m.repos.setReleases(releaseCache.Releases)
		m.chat = m.chat.setReleases(releaseCache.Releases)
		m.status = "Loaded from cache. Refreshing..."
	}

	// Focus the chat input now (Init is a value receiver so focusTab there is lost).
	if startTab == tabChat {
		m.chat.input.Focus()
	}

	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		fetchReleasesCmd(m.cfg.Repos, m.progress),
		tickCmd(),
		m.chat.Init(),
	}
	if cmd := m.focusTab(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	// Without alt-screen (e.g. VHS recording), force a window size since
	// Bubble Tea won't detect it automatically.
	if m.noAltScreen {
		w, h := 170, 44 // sensible defaults for 1400x800 @ 14pt
		if cols, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && cols > 0 {
			w = cols
		}
		if lines, err := strconv.Atoi(os.Getenv("LINES")); err == nil && lines > 0 {
			h = lines
		}
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: w, Height: h}
		})
	}
	return tea.Batch(cmds...)
}

func tickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentHeight := m.height - 5
		contentWidth := m.width

		m.releases = m.releases.setSize(contentWidth, contentHeight)
		m.repos = m.repos.setSize(contentWidth, contentHeight)
		m.summary = m.summary.setSize(contentWidth, contentHeight)
		m.chat = m.chat.setSize(contentWidth, contentHeight)
		m.addModal = m.addModal.setSize(contentWidth, contentHeight)
		return m, nil

	case tickMsg:
		if m.loading {
			return m, tickCmd()
		}
		return m, nil

	case releasesLoadedMsg:
		m.loading = false
		m.refreshing = false
		merged := cache.Merge(m.allReleases, msg.releases)
		sortReleases(merged)
		m.allReleases = merged
		m.releases = m.releases.setReleases(merged)
		m.repos = m.repos.setReleases(merged)
		m.chat = m.chat.setReleases(merged)
		status := fmt.Sprintf("Loaded %d releases from %d repos", len(merged), len(m.cfg.Repos))
		if len(msg.errors) > 0 {
			status += fmt.Sprintf(" (%d errors)", len(msg.errors))
		}
		m.status = status
		m.trackedRepos = m.cfg.Repos
		return m, saveCacheCmd(merged, m.trackedRepos)

	case cacheSavedMsg:
		if msg.err != nil {
			m.status += " (cache save failed)"
		} else {
			m.lastFetched = msg.lastFetched
		}
		return m, nil

	case repoInfoLoadedMsg:
		m.repos = m.repos.setRepoInfo(msg.repo, msg.info, msg.err)
		return m, nil

	case summaryLoadedMsg:
		if msg.err != nil {
			m.summary = m.summary.setContent(fmt.Sprintf("Error: %v", msg.err))
		} else {
			m.summary = m.summary.setContent(msg.content)
		}
		return m, nil

	case chatResponseMsg:
		if msg.err != nil {
			m.chat = m.chat.addMessage("Error", msg.err.Error())
		} else {
			m.chat = m.chat.addMessage("AI", msg.response)
		}
		return m, nil

	case repoAddedMsg:
		m.showModal = false
		m.cfg.AddRepo(msg.repo)
		if err := m.cfg.Save(); err != nil {
			m.status = fmt.Sprintf("Error saving config: %v", err)
			return m, nil
		}
		m.repos = m.repos.setRepoList(m.cfg.Repos)
		p := &loadingProgress{total: int32(len(m.cfg.Repos))}
		p.current.Store("")
		m.progress = p
		m.refreshing = true
		m.status = fmt.Sprintf("Added %s, refreshing...", msg.repo)
		return m, fetchReleasesCmd(m.cfg.Repos, m.progress)

	case repoRemovedMsg:
		m.cfg.RemoveRepo(msg.repo)
		if err := m.cfg.Save(); err != nil {
			m.status = fmt.Sprintf("Error saving config: %v", err)
			return m, nil
		}
		m.repos = m.repos.setRepoList(m.cfg.Repos)
		// Remove releases for this repo immediately
		var kept []github.Release
		for _, r := range m.allReleases {
			if r.Repo != msg.repo {
				kept = append(kept, r)
			}
		}
		m.allReleases = kept
		m.releases = m.releases.setReleases(kept)
		m.repos = m.repos.setReleases(kept)
		m.chat = m.chat.setReleases(kept)
		m.trackedRepos = m.cfg.Repos
		m.status = fmt.Sprintf("Removed %s", msg.repo)
		return m, saveCacheCmd(kept, m.cfg.Repos)

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case tea.KeyMsg:
		// Confirm removal takes priority
		if m.confirmRemove != "" {
			if msg.String() == "y" {
				repo := m.confirmRemove
				m.confirmRemove = ""
				return m, func() tea.Msg { return repoRemovedMsg{repo: repo} }
			}
			m.confirmRemove = ""
			m.status = "Removal cancelled"
			return m, nil
		}

		// Loading screen only allows quit
		if m.loading {
			if key.Matches(msg, keys.Quit) {
				return m, tea.Quit
			}
			return m, nil
		}

		// Modal gets priority
		if m.showModal {
			var cmd tea.Cmd
			m.addModal, cmd = m.addModal.Update(msg)
			if m.addModal.submitted {
				m.showModal = false
				if m.addModal.value != "" {
					return m, func() tea.Msg { return repoAddedMsg{repo: m.addModal.value} }
				}
			} else if m.addModal.cancelled {
				m.showModal = false
			}
			return m, cmd
		}

		// Filter input in releases tab gets all key input (except quit/nav)
		if m.activeTab == tabReleases && m.releases.filtering &&
			!key.Matches(msg, keys.Quit) &&
			!key.Matches(msg, keys.PrevTab) && !key.Matches(msg, keys.NextTab) &&
			!key.Matches(msg, keys.Tab1) && !key.Matches(msg, keys.Tab2) &&
			!key.Matches(msg, keys.Tab3) && !key.Matches(msg, keys.Tab4) {
			var cmd tea.Cmd
			m.releases, cmd = m.releases.Update(msg)
			return m, cmd
		}

		// Chat tab gets all key input when focused (except global/nav keys)
		if m.activeTab == tabChat && !key.Matches(msg, keys.Quit) &&
			!key.Matches(msg, keys.PrevTab) && !key.Matches(msg, keys.NextTab) &&
			!key.Matches(msg, keys.Tab1) && !key.Matches(msg, keys.Tab2) &&
			!key.Matches(msg, keys.Tab3) && !key.Matches(msg, keys.Tab4) {
			var cmd tea.Cmd
			m.chat, cmd = m.chat.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Refresh):
			p := &loadingProgress{total: int32(len(m.cfg.Repos))}
			p.current.Store("")
			m.progress = p
			m.refreshing = true
			m.status = "Refreshing..."
			return m, fetchReleasesCmd(m.cfg.Repos, m.progress)

		case key.Matches(msg, keys.AddRepo):
			m.addModal = newAddModalModel()
			m.addModal = m.addModal.setSize(m.width, m.height)
			m.showModal = true
			return m, m.addModal.Init()

		case key.Matches(msg, keys.RemoveRepo):
			var repo string
			if m.activeTab == tabRepos {
				repo = m.repos.selectedRepo()
			} else if m.activeTab == tabReleases {
				if rel := m.releases.selectedRelease(); rel != nil {
					repo = rel.Repo
				}
			}
			if repo != "" {
				m.confirmRemove = repo
				m.status = fmt.Sprintf("Remove %s? Press y to confirm, any other key to cancel", repo)
				return m, nil
			}

		case key.Matches(msg, keys.Summary):
			if m.activeTab == tabSummary {
				m.summary = m.summary.setContent("Generating summary...")
				return m, generateSummaryCmd(m.allReleases)
			}

		case key.Matches(msg, keys.PrevTab):
			m.activeTab = (m.activeTab - 1 + tabID(len(tabNames))) % tabID(len(tabNames))
			return m, m.focusTab()

		case key.Matches(msg, keys.NextTab):
			m.activeTab = (m.activeTab + 1) % tabID(len(tabNames))
			return m, m.focusTab()

		case key.Matches(msg, keys.Tab1):
			m.activeTab = tabReleases
			return m, m.focusTab()
		case key.Matches(msg, keys.Tab2):
			m.activeTab = tabRepos
			return m, m.focusTab()
		case key.Matches(msg, keys.Tab3):
			m.activeTab = tabSummary
			return m, m.focusTab()
		case key.Matches(msg, keys.Tab4):
			m.activeTab = tabChat
			return m, m.focusTab()

		case key.Matches(msg, keys.Tab):
			switch m.activeTab {
			case tabReleases:
				m.releases = m.releases.toggleFocus()
			case tabRepos:
				m.repos = m.repos.toggleFocus()
			}
			return m, nil
		}

		// Pass through to active tab
		switch m.activeTab {
		case tabReleases:
			var cmd tea.Cmd
			m.releases, cmd = m.releases.Update(msg)
			cmds = append(cmds, cmd)
		case tabRepos:
			var cmd tea.Cmd
			m.repos, cmd = m.repos.Update(msg)
			cmds = append(cmds, cmd)
		case tabSummary:
			var cmd tea.Cmd
			m.summary, cmd = m.summary.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	// Show loading screen (works even before we know terminal size)
	if m.loading {
		return m.viewLoading()
	}

	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	// Tab bar
	b.WriteString(m.viewTabBar())
	b.WriteString("\n")

	// Active tab content
	switch m.activeTab {
	case tabReleases:
		b.WriteString(m.releases.View())
	case tabRepos:
		b.WriteString(m.repos.View())
	case tabSummary:
		b.WriteString(m.summary.View())
	case tabChat:
		b.WriteString(m.chat.View())
	}
	b.WriteString("\n")

	// Status bar
	b.WriteString(m.viewStatusBar())

	// Modal overlay
	if m.showModal {
		return m.renderWithModal(b.String())
	}

	return b.String()
}

func (m Model) viewTabBar() string {
	var tabs []string
	for i, name := range tabNames {
		num := fmt.Sprintf("%d", i+1)
		if tabID(i) == m.activeTab {
			label := " " + num + " " + name + " "
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			label := " " + mutedStyle.Render(num) + " " + name + " "
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}

	tabLine := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)

	// Fill the rest of the line with a border
	tabLineWidth := lipgloss.Width(tabLine)
	remaining := m.width - tabLineWidth
	if remaining > 0 {
		filler := lipgloss.NewStyle().
			Foreground(colorDimBorder).
			Render(strings.Repeat("─", remaining))
		tabLine += filler
	}

	return tabLine
}

func (m Model) viewStatusBar() string {
	// Left side: status text
	statusText := m.status
	if !m.lastFetched.IsZero() {
		statusText += mutedStyle.Render(" | ") + statusAccentStyle.Render(formatTimeAgo(m.lastFetched))
	}
	if m.refreshing {
		statusText += " " + progressRepoStyle.Render("refreshing...")
	}
	left := statusBarStyle.Render(statusText)

	// Right side: help keys
	help := helpKeyStyle.Render("q") + helpDescStyle.Render(" quit ") +
		helpKeyStyle.Render("r") + helpDescStyle.Render(" refresh ") +
		helpKeyStyle.Render("a") + helpDescStyle.Render(" add ") +
		helpKeyStyle.Render("d") + helpDescStyle.Render(" remove ") +
		helpKeyStyle.Render("s") + helpDescStyle.Render(" summary ") +
		helpKeyStyle.Render("/") + helpDescStyle.Render(" filter ") +
		helpKeyStyle.Render("tab") + helpDescStyle.Render(" focus")

	// Lay out left and right
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(help)
	gap := m.width - leftW - rightW
	if gap < 2 {
		gap = 2
	}

	return left + strings.Repeat(" ", gap) + help
}

var logo = `
   ╦═╗┌─┐┬  ┌─┐┌─┐┌─┐┌─┐
   ╠╦╝├┤ │  ├┤ ├─┤└─┐├┤
   ╩╚═└─┘┴─┘└─┘┴ ┴└─┘└─┘
      ╦═╗┌─┐┌┬┐┌─┐┬─┐
      ╠╦╝├─┤ ││├─┤├┬┘
      ╩╚═┴ ┴─┴┘┴ ┴┴└─`

func (m Model) viewLoading() string {
	renderedLogo := logoStyle.Render(logo)

	// Subtitle
	subtitle := logoSubtitleStyle.Render("Track releases across your favorite repos")

	// Progress bar with smooth characters
	barWidth := 36
	done := int(m.progress.done.Load())
	total := int(m.progress.total)
	var filled int
	if total > 0 {
		filled = barWidth * done / total
	}
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	bar := progressBarFilledStyle.Render(strings.Repeat("━", filled)) +
		progressBarEmptyStyle.Render(strings.Repeat("─", empty))

	countText := progressCountStyle.Render(fmt.Sprintf("%d/%d", done, total))
	progressLine := "   " + bar + "  " + countText

	// Current repo being fetched
	current, _ := m.progress.current.Load().(string)
	var currentLine string
	if current != "" {
		currentLine = progressRepoStyle.Render("   fetching " + current)
	} else {
		currentLine = progressTextStyle.Render("   connecting to GitHub...")
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		renderedLogo,
		"",
		subtitle,
		"",
		progressLine,
		currentLine,
	)

	// Center if we know the terminal size, otherwise just return
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			content,
		)
	}
	return "\n" + content + "\n"
}

func (m Model) renderWithModal(base string) string {
	modalContent := m.addModal.View()
	modal := modalStyle.Render(modalContent)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
}

// focusTab ensures the right sub-component is focused for the active tab.
func (m *Model) focusTab() tea.Cmd {
	if m.activeTab == tabChat {
		m.chat.input.Focus()
		return textinput.Blink
	}
	m.chat.input.Blur()
	return nil
}

// Commands

const maxConcurrentFetches = 8

func fetchReleasesCmd(repos []string, progress *loadingProgress) tea.Cmd {
	return func() tea.Msg {
		if len(repos) == 0 {
			return releasesLoadedMsg{}
		}

		type result struct {
			releases []github.Release
			err      string
		}

		sem := make(chan struct{}, maxConcurrentFetches)
		var mu sync.Mutex
		var wg sync.WaitGroup
		results := make([]result, len(repos))

		for i, repo := range repos {
			wg.Add(1)
			go func(idx int, r string) {
				defer wg.Done()
				sem <- struct{}{}        // acquire
				defer func() { <-sem }() // release

				progress.current.Store(r)

				releases, err := github.FetchReleases(r)
				if err != nil {
					// Changelog fallback only when releases API fails
					changelog, clErr := github.FetchChangelog(r)
					if clErr == nil && changelog != "" {
						fallbackDate, _ := github.FetchChangelogCommitDate(r)
						entries := github.ParseChangelogEntries(changelog, r, fallbackDate)
						if len(entries) > 0 {
							mu.Lock()
							results[idx] = result{releases: entries}
							mu.Unlock()
							progress.done.Add(1)
							return
						}
					}
					mu.Lock()
					results[idx] = result{err: err.Error()}
					mu.Unlock()
					progress.done.Add(1)
					return
				}

				mu.Lock()
				results[idx] = result{releases: releases}
				mu.Unlock()
				progress.done.Add(1)
			}(i, repo)
		}

		wg.Wait()

		var allReleases []github.Release
		var errors []string
		for _, r := range results {
			if r.err != "" {
				errors = append(errors, r.err)
			} else {
				allReleases = append(allReleases, r.releases...)
			}
		}

		sortReleases(allReleases)
		return releasesLoadedMsg{releases: allReleases, errors: errors}
	}
}

func sortReleases(releases []github.Release) {
	slices.SortFunc(releases, func(a, b github.Release) int {
		return cmp.Compare(b.PublishedAt, a.PublishedAt)
	})
}

func saveCacheCmd(releases []github.Release, trackedRepos []string) tea.Cmd {
	// Copy slices to avoid races with the TUI goroutine
	releasesCopy := make([]github.Release, len(releases))
	copy(releasesCopy, releases)
	trackedCopy := make([]string, len(trackedRepos))
	copy(trackedCopy, trackedRepos)
	return func() tea.Msg {
		c := &cache.Cache{
			Releases:    releasesCopy,
			LastFetched: time.Now().UTC(),
		}
		err := c.Save(trackedCopy)
		return cacheSavedMsg{err: err, lastFetched: c.LastFetched}
	}
}

func generateSummaryCmd(releases []github.Release) tea.Cmd {
	return func() tea.Msg {
		content, err := ai.SummarizeReleases(releases)
		if err != nil {
			return summaryLoadedMsg{err: err}
		}
		return summaryLoadedMsg{content: content}
	}
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
