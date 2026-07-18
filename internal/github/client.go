package github

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// requestTimeout bounds every GitHub API request so a stalled connection or
// response cannot block the refresh fan-out (and cache warm) indefinitely.
const requestTimeout = 30 * time.Second

var (
	clientOnce sync.Once
	httpClient *http.Client
	authToken  string
	clientErr  error
	// apiBaseURL is the production API origin; overridable in tests only.
	apiBaseURL = "https://api.github.com/"
)

func ensureClient() error {
	// Allow tests to inject an already-configured client, bypassing gh.
	if httpClient != nil {
		return clientErr
	}
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
		httpClient = &http.Client{Timeout: requestTimeout}
	})
	return clientErr
}

func apiGet(pathOrURL string) ([]byte, http.Header, error) {
	if err := ensureClient(); err != nil {
		return nil, nil, err
	}

	url := pathOrURL
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = apiBaseURL + pathOrURL
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
