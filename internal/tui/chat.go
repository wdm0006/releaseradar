package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wdm0006/releaseradar/internal/ai"
	"github.com/wdm0006/releaseradar/internal/github"
)

type chatMessage struct {
	role    string
	content string
}

type chatModel struct {
	viewport viewport.Model
	input    textinput.Model
	messages []chatMessage
	releases []github.Release
	width    int
	height   int
}

func newChatModel() chatModel {
	ti := textinput.New()
	ti.Placeholder = "Ask about your releases..."
	ti.Prompt = ""
	ti.CharLimit = 500

	vp := viewport.New(80, 20)

	return chatModel{
		viewport: vp,
		input:    ti,
	}
}

func (m chatModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m chatModel) setSize(w, h int) chatModel {
	m.width = w
	m.height = h
	m.viewport.Width = w - 4
	m.viewport.Height = h - 5
	m.input.Width = w - 6
	return m
}

func (m chatModel) setReleases(releases []github.Release) chatModel {
	m.releases = releases
	return m
}

func (m chatModel) addMessage(role, content string) chatModel {
	// Replace "Thinking..." placeholder if present
	if len(m.messages) > 0 && m.messages[len(m.messages)-1].content == "Thinking..." {
		m.messages[len(m.messages)-1] = chatMessage{role: role, content: content}
	} else {
		m.messages = append(m.messages, chatMessage{role: role, content: content})
	}
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m
}

func (m chatModel) renderMessages() string {
	if len(m.messages) == 0 {
		return mutedStyle.Render("Ask a question about your tracked releases...")
	}

	renderer := newMarkdownRenderer(m.viewport.Width - 4)

	var b strings.Builder
	for _, msg := range m.messages {
		switch msg.role {
		case "You":
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render("You: "))
			b.WriteString(msg.content)
		case "AI":
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorSuccess).Render("AI:"))
			if renderer != nil {
				rendered, err := renderer.Render(msg.content)
				if err == nil {
					b.WriteString(strings.TrimRight(rendered, "\n"))
				} else {
					b.WriteString(" " + msg.content)
				}
			} else {
				b.WriteString(" " + msg.content)
			}
		default:
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorError).Render(msg.role + ": "))
			b.WriteString(msg.content)
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			question := strings.TrimSpace(m.input.Value())
			if question == "" {
				return m, nil
			}
			m.input.SetValue("")
			m = m.addMessage("You", question)
			m.messages = append(m.messages, chatMessage{role: "AI", content: "Thinking..."})
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, sendChatCmd(question, m.releases)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m chatModel) View() string {
	vpView := panelStyle.Width(m.width - 3).Height(m.height - 3).Render(m.viewport.View())
	inputLine := fmt.Sprintf(" %s %s",
		lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(">"),
		m.input.View(),
	)
	return lipgloss.JoinVertical(lipgloss.Left, vpView, inputLine)
}

func sendChatCmd(question string, releases []github.Release) tea.Cmd {
	return func() tea.Msg {
		resp, err := ai.Chat(question, releases)
		if err != nil {
			return chatResponseMsg{err: err}
		}
		return chatResponseMsg{response: resp}
	}
}
