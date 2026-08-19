package github

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// requestTimeout bounds every GitHub API request so a stalled connection or
// response cannot block the refresh fan-out (and cache warm) indefinitely.
const requestTimeout = 30 * time.Second

var (
	// clientMu guards httpClient, authToken and clientErr. Every read and write
	// of those three goes through it, so a caller that finds the client already
	// built is ordered against the goroutine that built it.
	clientMu   sync.Mutex
	httpClient *http.Client
	authToken  string
	clientErr  error
	// apiBaseURL is the production API origin; overridable in tests only.
	apiBaseURL = "https://api.github.com/"
	// lookupEnv and ghAuthToken are the two token sources; overridable in tests
	// only, so token selection can be exercised without a real gh binary.
	lookupEnv   = os.Getenv
	ghAuthToken = runGHAuthToken
)

func runGHAuthToken() ([]byte, error) {
	return exec.Command("gh", "auth", "token").Output()
}

// ensureClient resolves the bearer token and HTTP client once and returns them
// to the caller. Callers must use the returned values rather than reading the
// package globals, which are only valid while clientMu is held.
func ensureClient() (*http.Client, string, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if httpClient != nil || clientErr != nil {
		return httpClient, authToken, clientErr
	}

	if token := strings.TrimSpace(lookupEnv("GITHUB_TOKEN")); token != "" {
		authToken = token
		httpClient = &http.Client{Timeout: requestTimeout}
		return httpClient, authToken, nil
	}

	output, err := ghAuthToken()
	if err != nil {
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			clientErr = fmt.Errorf("GitHub CLI (gh) not found\n\nInstall it from: https://cli.github.com, or set GITHUB_TOKEN")
		} else {
			clientErr = fmt.Errorf("not authenticated with GitHub CLI\n\nRun: gh auth login, or set GITHUB_TOKEN")
		}
		return nil, "", clientErr
	}

	authToken = strings.TrimSpace(string(output))
	httpClient = &http.Client{Timeout: requestTimeout}
	return httpClient, authToken, nil
}

func apiGet(pathOrURL string) ([]byte, http.Header, error) {
	client, token, err := ensureClient()
	if err != nil {
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
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
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
