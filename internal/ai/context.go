package ai

import (
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/wdm0006/releaseradar/internal/github"
)

// maxBodyBytes bounds how much of a single release body is sent to the model.
// Release notes are unbounded upstream, so without this one large changelog can
// push a request past the model's context limit.
const maxBodyBytes = 2000

const truncationMarker = "\n\n[release notes truncated]"

// truncateBody limits body to at most limit bytes, cutting on a UTF-8 rune
// boundary so the result is always valid UTF-8, and marks that it was cut.
func truncateBody(body string, limit int) string {
	if len(body) <= limit {
		return body
	}
	truncated := body[:limit]
	for len(truncated) > 0 {
		r, size := utf8.DecodeLastRuneInString(truncated)
		if r != utf8.RuneError || size > 1 {
			break
		}
		truncated = truncated[:len(truncated)-1]
	}
	return truncated + truncationMarker
}

// releaseContext returns the sorted set of repositories represented by releases
// and one bounded markdown block per release, for at most maxReleases releases.
func releaseContext(releases []github.Release, maxReleases int) (repos []string, blocks []string) {
	repoSet := make(map[string]bool)
	for _, r := range releases {
		repoSet[r.Repo] = true
	}
	repos = make([]string, 0, len(repoSet))
	for r := range repoSet {
		repos = append(repos, r)
	}
	sort.Strings(repos)

	limit := maxReleases
	if limit > len(releases) {
		limit = len(releases)
	}
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
		body = truncateBody(body, maxBodyBytes)
		blocks = append(blocks, fmt.Sprintf("**%s - %s** (%s)\n%s\n", r.Repo, name, published, body))
	}
	return repos, blocks
}
