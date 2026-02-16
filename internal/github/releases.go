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
