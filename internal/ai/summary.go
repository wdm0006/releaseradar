package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wdm0006/releaseradar/internal/github"
)

type ReleaseSummary struct {
	Summary         string   `json:"summary"`
	KeyHighlights   []string `json:"key_highlights"`
	BreakingChanges []string `json:"breaking_changes"`
}

var summarySchema = &respFormat{
	Type: "json_schema",
	JSONSchema: &jsonSchema{
		Name:   "release_summary",
		Strict: true,
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Ecosystem overview of what's happening across the tracked projects",
				},
				"key_highlights": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "Key highlights from recent releases",
				},
				"breaking_changes": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "Breaking changes that users should be aware of",
				},
			},
			"required":             []string{"summary", "key_highlights", "breaking_changes"},
			"additionalProperties": false,
		},
	},
}

const summarySystemPrompt = `You are a helpful assistant that summarizes software release notes across multiple projects. Provide an ecosystem-level overview that tells the user what's happening NOW in the tools they care about. Focus on trends, major updates, and the overall state of the ecosystem. Be conversational and insightful - help them understand the big picture, not just individual changes.`

// maxSummaryReleases is how many releases the summary prompt covers.
const maxSummaryReleases = 15

func buildSummaryPrompt(releases []github.Release) string {
	repos, releaseTexts := releaseContext(releases, maxSummaryReleases)

	return fmt.Sprintf(`You're tracking releases from %d repositories: %s

Here are the most recent releases across this ecosystem:

%s

Provide an ecosystem overview: What's happening NOW in these projects? What are the major trends, themes, or significant updates?
Help the user understand what's going on in the tools they care about. Keep the summary under 250 words but be insightful.`,
		len(repos), strings.Join(repos, ", "), strings.Join(releaseTexts, "\n"))
}

func SummarizeReleases(releases []github.Release) (string, error) {
	if len(releases) == 0 {
		return "No releases to summarize.", nil
	}

	content, err := callOpenAI(summarySystemPrompt, buildSummaryPrompt(releases), summarySchema)
	if err != nil {
		return "", err
	}

	var summary ReleaseSummary
	if err := json.Unmarshal([]byte(content), &summary); err != nil {
		// If structured output fails, return raw content
		return content, nil
	}

	// Format as markdown
	var b strings.Builder
	b.WriteString("## Summary\n\n")
	b.WriteString(summary.Summary)
	b.WriteString("\n\n")

	if len(summary.KeyHighlights) > 0 {
		b.WriteString("### Key Highlights\n\n")
		for _, h := range summary.KeyHighlights {
			b.WriteString(fmt.Sprintf("- %s\n", h))
		}
		b.WriteString("\n")
	}

	if len(summary.BreakingChanges) > 0 {
		b.WriteString("### Breaking Changes\n\n")
		for _, c := range summary.BreakingChanges {
			b.WriteString(fmt.Sprintf("- %s\n", c))
		}
	}

	return b.String(), nil
}
