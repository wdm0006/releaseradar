package github

import (
	"encoding/json"
	"fmt"
)

type RepoInfo struct {
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	Stars       int      `json:"stargazers_count"`
	Forks       int      `json:"forks_count"`
	Language    string   `json:"language"`
	Topics      []string `json:"topics"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Homepage    string   `json:"homepage"`
	Archived    bool     `json:"archived"`
	Fork        bool     `json:"fork"`
}

func FetchRepoInfo(repo string) (RepoInfo, error) {
	body, _, err := apiGet(fmt.Sprintf("repos/%s", repo))
	if err != nil {
		return RepoInfo{}, fmt.Errorf("%s: %w", repo, err)
	}

	var info RepoInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return RepoInfo{}, fmt.Errorf("failed to parse repo info for %s: %w", repo, err)
	}

	return info, nil
}
