package ai

import (
	"fmt"
	"strings"

	"github.com/wdm0006/releaseradar/internal/github"
)

const chatSystemPrompt = `You are a knowledgeable assistant helping a developer understand GitHub releases from repositories they track. You have access to recent release notes and can answer questions about features, changes, breaking changes, version compatibility, migration paths, and trends across the projects. Be concise, technical, and helpful. If you don't have enough information, say so.

Format your responses using markdown: use **bold** for emphasis, headings (## and ###) to organize sections, ` + "`" + `inline code` + "`" + ` for package names and versions, code blocks for examples, and bullet lists for enumerating items. Keep formatting clean and readable in a terminal.`

// maxChatReleases is how many releases the chat prompt covers.
const maxChatReleases = 20

func buildChatPrompt(question string, releases []github.Release) string {
	repos, contextTexts := releaseContext(releases, maxChatReleases)

	return fmt.Sprintf(`Context: The user is tracking %d repositories: %s

Here are the recent releases:

%s

User question: %s

Provide a helpful, concise answer based on the release notes above.`,
		len(repos), strings.Join(repos, ", "), strings.Join(contextTexts, "\n"), question)
}

func Chat(question string, releases []github.Release) (string, error) {
	if len(releases) == 0 {
		return "No releases available to query. Add some repositories first!", nil
	}

	return callOpenAI(chatSystemPrompt, buildChatPrompt(question, releases), nil)
}
