package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wdm0006/releaseradar/internal/config"
	"github.com/wdm0006/releaseradar/internal/github"
)

var addCmd = &cobra.Command{
	Use:   "add <owner/repo>",
	Short: "Add a repository to track",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := github.ParseRepo(args[0])
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !cfg.AddRepo(repo) {
			fmt.Printf("Already tracking: %s\n", repo)
			return nil
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Added: %s\n", repo)
		return nil
	},
}
