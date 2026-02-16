package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wdm0006/releaseradar/internal/github"
)

type repoItem struct {
	name string
}

func (i repoItem) Title() string       { return i.name }
func (i repoItem) Description() string { return "" }
func (i repoItem) FilterValue() string { return i.name }

type reposModel struct {
	list     list.Model
	detail   viewport.Model
	repos    []string
	releases []github.Release
	repoInfo map[string]github.RepoInfo
	width    int
	height   int
	focused  int // 0 = list, 1 = detail
}

func newReposModel(repos []string) reposModel {
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = repoItem{name: r}
	}

	// Use a compact delegate for tighter spacing
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(colorSelectedBg).
		Bold(true).
		BorderLeftForeground(colorAccent)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(colorDimFg)

	l := list.New(items, delegate, 30, 20)
	l.Title = "Repositories"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent).
		Background(colorHeaderBg).
		Padding(0, 1).
		MarginBottom(1)

	vp := viewport.New(40, 20)

	return reposModel{
		list:     l,
		detail:   vp,
		repos:    repos,
		repoInfo: make(map[string]github.RepoInfo),
	}
}

func (m reposModel) setSize(w, h int) reposModel {
	m.width = w
	m.height = h

	const panelOverhead = 4
	listPanel := w*35/100 - 1
	detailPanel := w - listPanel - 1

	listContent := listPanel - panelOverhead
	detailContent := detailPanel - panelOverhead
	if listContent < 10 {
		listContent = 10
	}
	if detailContent < 20 {
		detailContent = 20
	}

	m.list.SetWidth(listContent)
	m.list.SetHeight(h - 2)

	m.detail.Width = detailContent
	m.detail.Height = h - 2

	return m
}

func (m reposModel) setRepoList(repos []string) reposModel {
	m.repos = repos
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = repoItem{name: r}
	}
	m.list.SetItems(items)
	return m
}

func (m reposModel) setReleases(releases []github.Release) reposModel {
	m.releases = releases
	if sel := m.selectedRepo(); sel != "" {
		m.detail.SetContent(m.renderRepoDetail(sel))
		m.detail.GotoTop()
	}
	return m
}

func (m reposModel) setRepoInfo(repo string, info github.RepoInfo, err error) reposModel {
	if err == nil {
		m.repoInfo[repo] = info
	}
	if sel := m.selectedRepo(); sel == repo {
		m.detail.SetContent(m.renderRepoDetail(repo))
		m.detail.GotoTop()
	}
	return m
}

func (m reposModel) selectedRepo() string {
	if item, ok := m.list.SelectedItem().(repoItem); ok {
		return item.name
	}
	return ""
}

func (m reposModel) toggleFocus() reposModel {
	m.focused = (m.focused + 1) % 2
	return m
}

func (m reposModel) Update(msg tea.Msg) (reposModel, tea.Cmd) {
	var cmds []tea.Cmd

	prevIdx := m.list.Index()

	if m.focused == 0 {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.list.Index() != prevIdx {
		if sel := m.selectedRepo(); sel != "" {
			m.detail.SetContent(m.renderRepoDetail(sel))
			m.detail.GotoTop()
			if _, cached := m.repoInfo[sel]; !cached {
				cmds = append(cmds, fetchRepoInfoCmd(sel))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m reposModel) View() string {
	listStyle := panelStyle
	detailStyle := panelStyle
	if m.focused == 0 {
		listStyle = focusedPanelStyle
	} else {
		detailStyle = focusedPanelStyle
	}

	listPanel := m.width*35/100 - 1
	detailPanel := m.width - listPanel - 1

	left := listStyle.Width(listPanel).Height(m.height - 2).Render(m.list.View())
	right := detailStyle.Width(detailPanel).Height(m.height - 2).Render(m.detail.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m reposModel) renderRepoDetail(repo string) string {
	var b strings.Builder

	// Repo name as styled header
	b.WriteString(releaseNameStyle.Render(repo))
	b.WriteString("\n")

	if info, ok := m.repoInfo[repo]; ok {
		if info.Description != "" {
			b.WriteString(metaValueStyle.Render(info.Description))
			b.WriteString("\n")
		}
		b.WriteString("\n")

		// Stats as inline badges
		stats := badgeStyle.Render(fmt.Sprintf(" %d stars ", info.Stars))
		stats += " "
		stats += badgeCyanStyle.Render(fmt.Sprintf(" %d forks ", info.Forks))
		if info.Language != "" {
			stats += " "
			stats += badgeAmberStyle.Render(fmt.Sprintf(" %s ", info.Language))
		}
		b.WriteString(stats)
		b.WriteString("\n\n")

		if info.CreatedAt != "" {
			created := info.CreatedAt[:10]
			updated := info.UpdatedAt[:10]
			b.WriteString(metaLabelStyle.Render("Created ") + metaValueStyle.Render(created))
			b.WriteString(metaLabelStyle.Render("  Updated ") + metaValueStyle.Render(updated))
			b.WriteString("\n")
		}
		if info.Homepage != "" {
			b.WriteString(metaLabelStyle.Render("Web ") + lipgloss.NewStyle().Foreground(colorCyan).Underline(true).Render(info.Homepage))
			b.WriteString("\n")
		}
		if len(info.Topics) > 0 {
			b.WriteString("\n")
			for _, t := range info.Topics {
				b.WriteString(lipgloss.NewStyle().
					Foreground(colorPrimary).
					Background(lipgloss.Color("#1E1B4B")).
					Padding(0, 1).
					Render(t))
				b.WriteString(" ")
			}
			b.WriteString("\n")
		}
		if info.Archived {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("ARCHIVED"))
			b.WriteString("\n")
		}
	} else {
		b.WriteString("\n")
		b.WriteString(progressRepoStyle.Render("Loading repository info..."))
		b.WriteString("\n")
	}

	// Divider
	detailW := m.detail.Width - 4
	if detailW < 10 {
		detailW = 10
	}
	if detailW > 70 {
		detailW = 70
	}
	b.WriteString("\n")
	b.WriteString(releaseDividerStyle.Render(strings.Repeat("─", detailW)))
	b.WriteString("\n\n")

	// Releases section
	var repoReleases []github.Release
	for _, r := range m.releases {
		if r.Repo == repo {
			repoReleases = append(repoReleases, r)
		}
	}

	if len(repoReleases) > 0 {
		b.WriteString(sectionHeaderStyle.Render(fmt.Sprintf("Releases (%d)", len(repoReleases))))
		b.WriteString("\n\n")
		limit := 10
		if limit > len(repoReleases) {
			limit = len(repoReleases)
		}
		for _, r := range repoReleases[:limit] {
			name := r.Name
			if name == "" {
				name = r.TagName
			}
			published := ""
			if len(r.PublishedAt) >= 10 {
				if t, err := time.Parse(time.RFC3339, r.PublishedAt); err == nil {
					published = t.Format("Jan 2")
				} else {
					published = r.PublishedAt[:10]
				}
			}
			b.WriteString("  ")
			b.WriteString(releaseTagStyle.Render(name))
			if published != "" {
				b.WriteString(metaLabelStyle.Render("  " + published))
			}
			b.WriteString("\n")
		}
		if len(repoReleases) > limit {
			b.WriteString(mutedStyle.Render(fmt.Sprintf("  ... and %d more", len(repoReleases)-limit)))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(mutedStyle.Render("No releases found."))
		b.WriteString("\n")
	}

	return b.String()
}

func fetchRepoInfoCmd(repo string) tea.Cmd {
	return func() tea.Msg {
		info, err := github.FetchRepoInfo(repo)
		return repoInfoLoadedMsg{repo: repo, info: info, err: err}
	}
}
