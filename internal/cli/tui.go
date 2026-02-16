package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wdm0006/releaseradar/internal/cache"
	"github.com/wdm0006/releaseradar/internal/config"
	"github.com/wdm0006/releaseradar/internal/tui"
)

func runTUI() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	releaseCache, err := cache.Load()
	if err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	m := tui.NewModel(cfg, releaseCache)
	opts := []tea.ProgramOption{}
	// Skip alt-screen when recording with VHS (it can't capture alt-screen content)
	if os.Getenv("RELEASERADAR_NO_ALTSCREEN") == "" {
		opts = append(opts, tea.WithAltScreen())
	}
	p := tea.NewProgram(m, opts...)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
