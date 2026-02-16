package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/wdm0006/releaseradar/internal/github"
)

type releasesModel struct {
	table            table.Model
	detail           viewport.Model
	allReleases      []github.Release
	filteredReleases []github.Release
	renderedDetail   []string
	renderer         *glamour.TermRenderer
	width            int
	height           int
	focused          int // 0 = table, 1 = detail

	filter     textinput.Model
	filtering  bool
	filterText string
	lastViewed time.Time
}

func newReleasesModel() releasesModel {
	columns := []table.Column{
		{Title: "Repository", Width: 25},
		{Title: "Release", Width: 22},
		{Title: "Published", Width: 12},
		{Title: "Tag", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(colorBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(colorAccent).
		Background(colorHeaderBg)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(colorSelectedBg).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(colorDimFg)
	t.SetStyles(s)

	vp := viewport.New(40, 20)

	fi := textinput.New()
	fi.Placeholder = "filter repos/releases..."
	fi.CharLimit = 100

	return releasesModel{
		table:  t,
		detail: vp,
		filter: fi,
	}
}

func (m releasesModel) setLastViewed(t time.Time) releasesModel {
	m.lastViewed = t
	return m
}

func (m releasesModel) setSize(w, h int) releasesModel {
	m.width = w
	m.height = h

	const panelOverhead = 4
	// Give table 55%, detail 45% for better readability
	tablePanel := w*55/100 - 1
	detailPanel := w - tablePanel - 1

	tableContent := tablePanel - panelOverhead
	detailContent := detailPanel - panelOverhead

	if tableContent < 20 {
		tableContent = 20
	}
	if detailContent < 20 {
		detailContent = 20
	}

	// Better column proportions
	colPublished := 12
	colTag := 16
	colRelease := 22
	colRepo := tableContent - colPublished - colTag - colRelease
	if colRepo < 14 {
		colRepo = 14
	}

	m.table.SetColumns([]table.Column{
		{Title: "Repository", Width: colRepo},
		{Title: "Release", Width: colRelease},
		{Title: "Published", Width: colPublished},
		{Title: "Tag", Width: colTag},
	})
	m.table.SetWidth(tableContent)
	m.table.SetHeight(h - 2)

	m.detail.Width = detailContent
	m.detail.Height = h - 2

	m.renderer = newMarkdownRenderer(detailContent - 4)
	m.filter.Width = tableContent - 10

	m = m.rebuildTable()
	return m
}

func (m releasesModel) setReleases(releases []github.Release) releasesModel {
	m.allReleases = releases
	m = m.applyFilter()
	return m
}

func (m releasesModel) applyFilter() releasesModel {
	if m.filterText == "" {
		m.filteredReleases = m.allReleases
	} else {
		lower := strings.ToLower(m.filterText)
		var filtered []github.Release
		for _, r := range m.allReleases {
			name := r.Name
			if name == "" {
				name = r.TagName
			}
			if strings.Contains(strings.ToLower(r.Repo), lower) ||
				strings.Contains(strings.ToLower(name), lower) ||
				strings.Contains(strings.ToLower(r.TagName), lower) {
				filtered = append(filtered, r)
			}
		}
		m.filteredReleases = filtered
	}
	m = m.rebuildTable()
	return m
}

func (m releasesModel) rebuildTable() releasesModel {
	rows := make([]table.Row, len(m.filteredReleases))
	for i, r := range m.filteredReleases {
		published := r.PublishedAt
		if published != "" {
			if t, err := time.Parse(time.RFC3339, published); err == nil {
				published = t.Format("2006-01-02")
			}
		}
		name := r.Name
		if name == "" {
			name = r.TagName
		}

		repoCol := r.Repo
		if !m.lastViewed.IsZero() && r.PublishedAt != "" {
			if t, err := time.Parse(time.RFC3339, r.PublishedAt); err == nil {
				if t.After(m.lastViewed) {
					repoCol = "* " + repoCol
				}
			}
		}

		rows[i] = table.Row{repoCol, name, published, r.TagName}
	}
	m.table.SetRows(rows)

	m.renderedDetail = make([]string, len(m.filteredReleases))
	for i, r := range m.filteredReleases {
		m.renderedDetail[i] = renderReleaseDetail(r, m.renderer, m.detail.Width)
	}

	if len(m.filteredReleases) > 0 {
		cursor := m.table.Cursor()
		if cursor < 0 || cursor >= len(m.renderedDetail) {
			cursor = 0
		}
		m.detail.SetContent(m.renderedDetail[cursor])
	} else {
		m.detail.SetContent("")
	}

	return m
}

func (m releasesModel) selectedRelease() *github.Release {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filteredReleases) {
		r := m.filteredReleases[cursor]
		return &r
	}
	return nil
}

func (m releasesModel) toggleFocus() releasesModel {
	m.focused = (m.focused + 1) % 2
	m.table.Focus()
	if m.focused == 1 {
		m.table.Blur()
	}
	return m
}

func (m releasesModel) Update(msg tea.Msg) (releasesModel, tea.Cmd) {
	// When filter input is active, route keys there
	if m.filtering {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.Type {
			case tea.KeyEsc:
				m.filtering = false
				m.filterText = ""
				m.filter.SetValue("")
				m.filter.Blur()
				m = m.applyFilter()
				return m, nil
			case tea.KeyEnter:
				m.filtering = false
				m.filter.Blur()
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		newText := m.filter.Value()
		if newText != m.filterText {
			m.filterText = newText
			m = m.applyFilter()
		}
		return m, cmd
	}

	// Start filtering on /
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "/" {
		m.filtering = true
		m.filter.Focus()
		return m, textinput.Blink
	}

	var cmds []tea.Cmd
	prevCursor := m.table.Cursor()

	if m.focused == 0 {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
	}

	cursor := m.table.Cursor()
	if cursor != prevCursor && cursor >= 0 && cursor < len(m.renderedDetail) {
		m.detail.SetContent(m.renderedDetail[cursor])
		m.detail.GotoTop()
	}

	return m, tea.Batch(cmds...)
}

func (m releasesModel) View() string {
	tableStyle := panelStyle
	detailStyle := panelStyle
	if m.focused == 0 {
		tableStyle = focusedPanelStyle
	} else {
		detailStyle = focusedPanelStyle
	}

	tablePanel := m.width*55/100 - 1
	detailPanel := m.width - tablePanel - 1

	// Build left panel content with optional filter
	var leftContent string
	if m.filtering {
		filterLine := helpKeyStyle.Render("/") + " " + m.filter.View()
		leftContent = filterLine + "\n" + m.table.View()
	} else if m.filterText != "" {
		count := fmt.Sprintf("%d", len(m.filteredReleases))
		filterLine := helpKeyStyle.Render("/") + " " +
			mutedStyle.Render(m.filterText) + "  " +
			lipgloss.NewStyle().Foreground(colorCyan).Render(count+" matches") + "  " +
			helpDescStyle.Render("esc clear")
		leftContent = filterLine + "\n" + m.table.View()
	} else {
		leftContent = m.table.View()
	}

	left := tableStyle.Width(tablePanel).Height(m.height - 2).Render(leftContent)
	right := detailStyle.Width(detailPanel).Height(m.height - 2).Render(m.detail.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func renderReleaseDetail(r github.Release, renderer *glamour.TermRenderer, width int) string {
	name := r.Name
	if name == "" {
		name = r.TagName
	}
	published := r.PublishedAt
	if published != "" {
		if t, err := time.Parse(time.RFC3339, published); err == nil {
			published = t.Format("Jan 2, 2006")
		}
	}

	var b strings.Builder

	// Styled header section
	b.WriteString(releaseNameStyle.Render(name))
	b.WriteString("\n")

	// Compact metadata line
	meta := metaLabelStyle.Render("repo ") + metaValueStyle.Render(r.Repo)
	meta += metaLabelStyle.Render("  tag ") + releaseTagStyle.Render(r.TagName)
	if r.Author.Login != "" && r.Author.Login != "changelog" {
		meta += metaLabelStyle.Render("  by ") + releaseAuthorStyle.Render(r.Author.Login)
	}
	b.WriteString(meta)
	b.WriteString("\n")

	if published != "" {
		b.WriteString(releaseDateStyle.Render(published))
		b.WriteString("\n")
	}

	// Divider
	divWidth := width - 4
	if divWidth < 10 {
		divWidth = 10
	}
	if divWidth > 60 {
		divWidth = 60
	}
	b.WriteString(releaseDividerStyle.Render(strings.Repeat("─", divWidth)))
	b.WriteString("\n\n")

	// Body rendered as markdown
	body := r.Body
	if body == "" {
		body = mutedStyle.Render("No release notes.")
	} else if renderer != nil {
		rendered, err := renderer.Render(body)
		if err == nil {
			body = strings.TrimSpace(rendered)
		}
	}
	b.WriteString(body)

	return b.String()
}
