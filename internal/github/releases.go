package github

import (
	"encoding/json"
	"fmt"
)

type Release struct {
	Repo        string `json:"repo"`
	Name        string `json:"name"`
	TagName     string `json:"tag_name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Author      Author `json:"author"`
}

type Author struct {
	Login string `json:"login"`
}

const maxPages = 3 // fetch up to 300 releases per repo

func FetchReleases(repo string) ([]Release, error) {
	var allReleases []Release

	pathOrURL := fmt.Sprintf("repos/%s/releases?per_page=100", repo)

	for page := 0; pathOrURL != "" && page < maxPages; page++ {
		body, header, err := apiGet(pathOrURL)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", repo, err)
		}

		var releases []Release
		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, fmt.Errorf("failed to parse releases for %s: %w", repo, err)
		}

		for i := range releases {
			releases[i].Repo = repo
		}
		allReleases = append(allReleases, releases...)

		pathOrURL = nextPageURL(header)
	}

	return allReleases, nil
}

// FetchReleasesWithFallback fetches a repository's published releases and falls
// back to its changelog when there are none to show. GitHub answers a repository
// with no published releases with 200 and an empty array, so an empty successful
// response is as much a reason to look for a CHANGELOG as a failed request is.
//
// A parseable changelog wins in both cases. Otherwise a genuinely empty
// repository stays a successful zero-release fetch, and a failed Releases
// request keeps reporting its own error.
func FetchReleasesWithFallback(repo string) ([]Release, error) {
	releases, err := FetchReleases(repo)
	if err == nil && len(releases) > 0 {
		return releases, nil
	}

	if entries := fetchChangelogReleases(repo); len(entries) > 0 {
		return entries, nil
	}

	return releases, err
}

func fetchChangelogReleases(repo string) []Release {
	changelog, err := FetchChangelog(repo)
	if err != nil || changelog == "" {
		return nil
	}
	fallbackDate, _ := FetchChangelogCommitDate(repo)
	return ParseChangelogEntries(changelog, repo, fallbackDate)
}
