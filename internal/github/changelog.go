package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type fileContent struct {
	Content string `json:"content"`
}

var changelogNames = []string{"CHANGELOG.md", "CHANGELOG", "HISTORY.md", "RELEASES.md"}

// versionPattern matches changelog headers like:
//
//	## [1.2.3] - 2024-01-15
//	## 1.2.3 (2024-01-15)
//	## 2.0.69
//	## v1.2.3
var versionPattern = regexp.MustCompile(`^##\s+(?:\[v?([^\]]+)\]|v?(\S+))(?:\s*[-–—]\s*)?\(?(\d{4}-\d{2}-\d{2})?\)?`)

func FetchChangelog(repo string) (string, error) {
	for _, name := range changelogNames {
		body, _, err := apiGet(fmt.Sprintf("repos/%s/contents/%s", repo, name))
		if err != nil {
			continue
		}

		var fc fileContent
		if err := json.Unmarshal(body, &fc); err != nil {
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(fc.Content, "\n", ""))
		if err != nil {
			continue
		}

		return string(decoded), nil
	}

	return "", fmt.Errorf("no changelog file found for %s", repo)
}

func FetchChangelogCommitDate(repo string) (string, error) {
	for _, name := range changelogNames {
		body, _, err := apiGet(fmt.Sprintf("repos/%s/commits?path=%s&per_page=1", repo, name))
		if err != nil {
			continue
		}

		var commits []struct {
			Commit struct {
				Committer struct {
					Date string `json:"date"`
				} `json:"committer"`
			} `json:"commit"`
		}
		if err := json.Unmarshal(body, &commits); err != nil || len(commits) == 0 {
			continue
		}

		return commits[0].Commit.Committer.Date, nil
	}

	return "", fmt.Errorf("no changelog commit date found for %s", repo)
}

func ParseChangelogEntries(changelog, repo, fallbackDate string) []Release {
	lines := strings.Split(changelog, "\n")
	var entries []Release
	var currentVersion string
	var currentDate string
	var currentBody []string

	for _, line := range lines {
		match := versionPattern.FindStringSubmatch(line)
		if match != nil {
			// Save previous entry
			if currentVersion != "" {
				entries = append(entries, Release{
					Repo:        repo,
					Name:        currentVersion,
					TagName:     currentVersion,
					Body:        strings.TrimSpace(strings.Join(currentBody, "\n")),
					PublishedAt: currentDate,
					Author:      Author{Login: "changelog"},
				})
			}

			// Start new entry — version from group 1 (bracketed) or group 2 (unbracketed)
			currentVersion = match[1]
			if currentVersion == "" {
				currentVersion = match[2]
			}
			currentVersion = strings.TrimSpace(currentVersion)
			currentBody = nil

			dateStr := match[3]
			if dateStr != "" {
				currentDate = dateStr + "T00:00:00Z"
			} else if fallbackDate != "" {
				currentDate = fallbackDate
			} else {
				currentDate = time.Now().UTC().Format(time.RFC3339)
			}
		} else if currentVersion != "" {
			currentBody = append(currentBody, line)
		}
	}

	// Last entry
	if currentVersion != "" {
		entries = append(entries, Release{
			Repo:        repo,
			Name:        currentVersion,
			TagName:     currentVersion,
			Body:        strings.TrimSpace(strings.Join(currentBody, "\n")),
			PublishedAt: currentDate,
			Author:      Author{Login: "changelog"},
		})
	}

	return entries
}
