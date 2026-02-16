package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type summaryModel struct {
	viewport viewport.Model
	content  string
	width    int
	height   int
}

func newSummaryModel() summaryModel {
	vp := viewport.New(80, 20)
	return summaryModel{
		viewport: vp,
		content:  "Press **s** on this tab to generate an AI summary of recent releases.",
	}
}

func (m summaryModel) setSize(w, h int) summaryModel {
	m.width = w
	m.height = h
	m.viewport.Width = w - 6
	m.viewport.Height = h - 2
	m.viewport.SetContent(m.renderContent())
	return m
}

func (m summaryModel) setContent(content string) summaryModel {
	m.content = content
	m.viewport.SetContent(m.renderContent())
	m.viewport.GotoTop()
	return m
}

func (m summaryModel) renderContent() string {
	renderWidth := m.viewport.Width - 2
	renderer := newMarkdownRenderer(renderWidth)
	if renderer == nil {
		return m.content
	}
	rendered, err := renderer.Render(m.content)
	if err != nil {
		return m.content
	}
	return strings.TrimSpace(rendered)
}

func (m summaryModel) Update(msg tea.Msg) (summaryModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m summaryModel) View() string {
	return panelStyle.Width(m.width - 2).Height(m.height - 2).Render(m.viewport.View())
}
