package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wdm0006/releaseradar/internal/config"
)

var removeCmd = &cobra.Command{
	Use:   "remove <owner/repo>",
	Short: "Remove a tracked repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.RemoveRepo(repo) {
			fmt.Printf("Not tracking: %s\n", repo)
			return nil
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Removed: %s\n", repo)
		return nil
	},
}
