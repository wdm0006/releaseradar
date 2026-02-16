package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wdm0006/releaseradar/internal/cache"
	"github.com/wdm0006/releaseradar/internal/config"
	"github.com/wdm0006/releaseradar/internal/tui"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot [tab]",
	Short: "Render TUI view as ANSI output (for piping to freeze)",
	Long: `Renders a single frame of the TUI and prints it to stdout.
Useful for generating screenshots with charmbracelet/freeze:

  releaseradar screenshot releases | freeze -o releases.png
  releaseradar screenshot repos | freeze -o repos.png
  releaseradar screenshot loading | freeze -o loading.png

Valid tabs: releases (default), repos, loading`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tab := "releases"
		if len(args) > 0 {
			tab = args[0]
		}

		width, _ := cmd.Flags().GetInt("width")
		height, _ := cmd.Flags().GetInt("height")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		releaseCache, err := cache.Load()
		if err != nil {
			return fmt.Errorf("failed to load cache: %w", err)
		}

		output := tui.RenderScreenshot(cfg, releaseCache, tab, width, height)
		fmt.Print(output)
		return nil
	},
}

func init() {
	screenshotCmd.Flags().Int("width", 170, "terminal width in columns")
	screenshotCmd.Flags().Int("height", 44, "terminal height in rows")
}
