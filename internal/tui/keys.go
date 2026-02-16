package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit       key.Binding
	Refresh    key.Binding
	AddRepo    key.Binding
	RemoveRepo key.Binding
	Summary    key.Binding
	PrevTab    key.Binding
	NextTab    key.Binding
	Tab        key.Binding
	Enter      key.Binding
	Filter     key.Binding
	Tab1       key.Binding
	Tab2       key.Binding
	Tab3       key.Binding
	Tab4       key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	AddRepo: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add repo"),
	),
	RemoveRepo: key.NewBinding(
		key.WithKeys("d", "x"),
		key.WithHelp("d/x", "remove repo"),
	),
	Summary: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "AI summary"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("[", "shift+tab"),
		key.WithHelp("[", "prev tab"),
	),
	NextTab: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "next tab"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "focus next"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "action"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Tab1: key.NewBinding(
		key.WithKeys("1"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
	),
	Tab4: key.NewBinding(
		key.WithKeys("4"),
	),
}
