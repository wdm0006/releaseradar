package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wdm0006/releaseradar/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tracked repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.Repos) == 0 {
			fmt.Println("No repositories tracked. Use 'releaseradar add <owner/repo>' to add one.")
			return nil
		}

		for _, repo := range cfg.Repos {
			fmt.Println(repo)
		}
		return nil
	},
}
