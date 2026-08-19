package tui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plainView(m addModalModel) string {
	return ansiRE.ReplaceAllString(m.View(), "")
}

func submitAddModal(value string) addModalModel {
	m := newAddModalModel()
	m.input.SetValue(value)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return m
}

func TestAddModalSubmitsCanonicalValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"canonical", "owner/repo", "owner/repo"},
		{"surrounding whitespace", "  owner/repo  ", "owner/repo"},
		{"github url", "https://github.com/owner/repo", "owner/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := submitAddModal(tt.input)
			if !m.submitted {
				t.Fatalf("input %q was not submitted (err %q)", tt.input, m.errMsg)
			}
			if m.value != tt.want {
				t.Fatalf("input %q submitted value %q, want %q", tt.input, m.value, tt.want)
			}
			if m.errMsg != "" {
				t.Fatalf("input %q left error %q", tt.input, m.errMsg)
			}
			if view := plainView(m); strings.Contains(view, "invalid repository") {
				t.Fatalf("input %q: modal view shows a rejection: %q", tt.input, view)
			}
		})
	}
}

func TestAddModalKeepsModalOpenOnRejection(t *testing.T) {
	for _, input := range []string{
		"",
		"/repo",
		"owner/",
		"owner/repo/tree/main",
		"owner/repo?per_page=1&foo=",
		"../../users/octocat",
	} {
		t.Run(input, func(t *testing.T) {
			m := submitAddModal(input)
			if m.submitted {
				t.Fatalf("input %q was submitted as %q", input, m.value)
			}
			if m.cancelled {
				t.Fatalf("input %q cancelled the modal", input)
			}
			if !strings.Contains(m.errMsg, "owner/repo") {
				t.Fatalf("input %q error %q does not name the expected format", input, m.errMsg)
			}
			if view := plainView(m); !strings.Contains(view, "invalid repository") {
				t.Fatalf("input %q: modal view does not show the rejection: %q", input, view)
			}
		})
	}
}

func TestAddModalClearsErrorOnNextKey(t *testing.T) {
	m := submitAddModal("owner/")
	if m.errMsg == "" {
		t.Fatal("expected a rejection message")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.errMsg != "" {
		t.Fatalf("error message survived further typing: %q", m.errMsg)
	}
	if got := m.input.Value(); got != "owner/x" {
		t.Fatalf("input value = %q, want %q", got, "owner/x")
	}
}
