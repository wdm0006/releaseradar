package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "releaseradar",
	Short: "Track GitHub releases across repositories",
	Long: `ReleaseRadar is a TUI for monitoring GitHub releases across multiple
repositories. It fetches recent releases, parses changelogs, and provides
AI-powered summaries using OpenAI.

Prerequisites:
  - GitHub CLI (gh) installed and authenticated
  - OPENAI_API_KEY env var set (optional, for AI features)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(screenshotCmd)
}

func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
