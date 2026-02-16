package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type addModalModel struct {
	input     textinput.Model
	submitted bool
	cancelled bool
	value     string
	width     int
	height    int
}

func newAddModalModel() addModalModel {
	ti := textinput.New()
	ti.Placeholder = "owner/repo"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	return addModalModel{
		input: ti,
	}
}

func (m addModalModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m addModalModel) setSize(w, h int) addModalModel {
	m.width = w
	m.height = h
	return m
}

func (m addModalModel) Update(msg tea.Msg) (addModalModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			val := strings.TrimSpace(m.input.Value())
			if val != "" && strings.Contains(val, "/") {
				m.submitted = true
				m.value = val
			}
			return m, nil
		case tea.KeyEsc:
			m.cancelled = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m addModalModel) View() string {
	return sectionHeaderStyle.Render("Add Repository") + "\n\n" +
		mutedStyle.Render("Format: owner/repo") + "\n\n" +
		m.input.View() + "\n\n" +
		helpKeyStyle.Render("enter") + helpDescStyle.Render(" add  ") +
		helpKeyStyle.Render("esc") + helpDescStyle.Render(" cancel")
}
