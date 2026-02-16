package ai

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wdm0006/releaseradar/internal/github"
)

const chatSystemPrompt = `You are a knowledgeable assistant helping a developer understand GitHub releases from repositories they track. You have access to recent release notes and can answer questions about features, changes, breaking changes, version compatibility, migration paths, and trends across the projects. Be concise, technical, and helpful. If you don't have enough information, say so.

Format your responses using markdown: use **bold** for emphasis, headings (## and ###) to organize sections, ` + "`" + `inline code` + "`" + ` for package names and versions, code blocks for examples, and bullet lists for enumerating items. Keep formatting clean and readable in a terminal.`

func Chat(question string, releases []github.Release) (string, error) {
	if len(releases) == 0 {
		return "No releases available to query. Add some repositories first!", nil
	}

	// Get unique repos
	repoSet := make(map[string]bool)
	for _, r := range releases {
		repoSet[r.Repo] = true
	}
	repos := make([]string, 0, len(repoSet))
	for r := range repoSet {
		repos = append(repos, r)
	}
	sort.Strings(repos)

	// Prepare context (up to 20 releases)
	limit := 20
	if limit > len(releases) {
		limit = len(releases)
	}
	var contextTexts []string
	for _, r := range releases[:limit] {
		name := r.Name
		if name == "" {
			name = r.TagName
		}
		published := ""
		if len(r.PublishedAt) >= 10 {
			published = r.PublishedAt[:10]
		}
		body := r.Body
		if body == "" {
			body = "No description"
		}
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		contextTexts = append(contextTexts, fmt.Sprintf("**%s - %s** (%s)\n%s\n", r.Repo, name, published, body))
	}

	userPrompt := fmt.Sprintf(`Context: The user is tracking %d repositories: %s

Here are the recent releases:

%s

User question: %s

Provide a helpful, concise answer based on the release notes above.`,
		len(repos), strings.Join(repos, ", "), strings.Join(contextTexts, "\n"), question)

	return callOpenAI(chatSystemPrompt, userPrompt, nil)
}
