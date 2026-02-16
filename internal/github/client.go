package github

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

var (
	clientOnce sync.Once
	httpClient *http.Client
	authToken  string
	clientErr  error
)

func ensureClient() error {
	clientOnce.Do(func() {
		cmd := exec.Command("gh", "auth", "token")
		output, err := cmd.Output()
		if err != nil {
			if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
				clientErr = fmt.Errorf("GitHub CLI (gh) not found\n\nInstall it from: https://cli.github.com")
			} else {
				clientErr = fmt.Errorf("not authenticated with GitHub CLI\n\nRun: gh auth login")
			}
			return
		}
		authToken = strings.TrimSpace(string(output))
		httpClient = &http.Client{}
	})
	return clientErr
}

func apiGet(pathOrURL string) ([]byte, http.Header, error) {
	if err := ensureClient(); err != nil {
		return nil, nil, err
	}

	url := pathOrURL
	if !strings.HasPrefix(url, "https://") {
		url = "https://api.github.com/" + pathOrURL
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case 404:
			return nil, nil, fmt.Errorf("not found")
		case 401, 403:
			return nil, nil, fmt.Errorf("authentication failed\n\nRun: gh auth login")
		default:
			return nil, nil, fmt.Errorf("GitHub API error %d", resp.StatusCode)
		}
	}

	return body, resp.Header, nil
}

func nextPageURL(header http.Header) string {
	link := header.Get("Link")
	if link == "" {
		return ""
	}
	for _, part := range strings.Split(link, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}
