package github

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const testChangelog = `# Changelog

## [1.3.0] - 2024-05-01
- adds a thing

## 1.2.0
- fixes a thing
`

const testCommitDate = "2024-04-01T12:00:00Z"

// fallbackServer serves the three endpoints FetchReleasesWithFallback can
// touch and records every path it is asked for, so a test can assert that the
// changelog probes did or did not happen.
type fallbackServer struct {
	mu             sync.Mutex
	paths          []string
	releasesStatus int
	releasesBody   string
	changelog      string // served as CHANGELOG.md when non-empty
}

func (f *fallbackServer) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.paths = append(f.paths, r.URL.Path)
		f.mu.Unlock()

		switch {
		case strings.HasSuffix(r.URL.Path, "/releases"):
			w.WriteHeader(f.releasesStatus)
			_, _ = w.Write([]byte(f.releasesBody))
		case strings.HasSuffix(r.URL.Path, "/contents/CHANGELOG.md") && f.changelog != "":
			encoded := base64.StdEncoding.EncodeToString([]byte(f.changelog))
			_, _ = fmt.Fprintf(w, `{"content":%q}`, encoded)
		case strings.HasSuffix(r.URL.Path, "/commits"):
			_, _ = fmt.Fprintf(w, `[{"commit":{"committer":{"date":%q}}}]`, testCommitDate)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	withTestServer(t, srv, 5*time.Second)
}

func (f *fallbackServer) requested(substr string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.paths {
		if strings.Contains(p, substr) {
			return true
		}
	}
	return false
}

// assertChangelogReleases pins the exact records the test changelog parses to,
// so a fallback that fires but returns nothing cannot pass.
func assertChangelogReleases(t *testing.T, got []Release) {
	t.Helper()
	want := []Release{
		{
			Repo:        "owner/repo",
			Name:        "1.3.0",
			TagName:     "1.3.0",
			Body:        "- adds a thing",
			PublishedAt: "2024-05-01T00:00:00Z",
			Author:      Author{Login: "changelog"},
		},
		{
			Repo:        "owner/repo",
			Name:        "1.2.0",
			TagName:     "1.2.0",
			Body:        "- fixes a thing",
			PublishedAt: testCommitDate,
			Author:      Author{Login: "changelog"},
		},
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d changelog releases, got %d (%+v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("release %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestFetchReleasesWithFallbackEmptyReleasesUsesChangelog(t *testing.T) {
	srv := &fallbackServer{releasesStatus: http.StatusOK, releasesBody: "[]", changelog: testChangelog}
	srv.start(t)

	releases, err := FetchReleasesWithFallback("owner/repo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertChangelogReleases(t, releases)
}

func TestFetchReleasesWithFallbackEmptyReleasesNoChangelog(t *testing.T) {
	srv := &fallbackServer{releasesStatus: http.StatusOK, releasesBody: "[]"}
	srv.start(t)

	releases, err := FetchReleasesWithFallback("owner/repo")
	if err != nil {
		t.Fatalf("a repository with no releases and no changelog must stay successful, got %v", err)
	}
	if len(releases) != 0 {
		t.Fatalf("expected no releases, got %+v", releases)
	}
	if !srv.requested("/contents/") {
		t.Error("expected the changelog fallback to be attempted")
	}
}

func TestFetchReleasesWithFallbackRealReleasesSkipsChangelog(t *testing.T) {
	body := `[{"name":"v2.0.0","tag_name":"v2.0.0","body":"real notes","published_at":"2024-06-01T10:00:00Z","author":{"login":"octocat"}}]`
	srv := &fallbackServer{releasesStatus: http.StatusOK, releasesBody: body, changelog: testChangelog}
	srv.start(t)

	releases, err := FetchReleasesWithFallback("owner/repo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := Release{
		Repo:        "owner/repo",
		Name:        "v2.0.0",
		TagName:     "v2.0.0",
		Body:        "real notes",
		PublishedAt: "2024-06-01T10:00:00Z",
		Author:      Author{Login: "octocat"},
	}
	if len(releases) != 1 || releases[0] != want {
		t.Fatalf("got %+v, want [%+v]", releases, want)
	}
	if srv.requested("/contents/") || srv.requested("/commits") {
		t.Errorf("changelog fallback must not run when real releases exist; requested %v", srv.paths)
	}
}

func TestFetchReleasesWithFallbackAPIErrorUsesChangelog(t *testing.T) {
	srv := &fallbackServer{releasesStatus: http.StatusInternalServerError, changelog: testChangelog}
	srv.start(t)

	releases, err := FetchReleasesWithFallback("owner/repo")
	if err != nil {
		t.Fatalf("expected the changelog to mask the API error, got %v", err)
	}
	assertChangelogReleases(t, releases)
}

func TestFetchReleasesWithFallbackAPIErrorNoChangelog(t *testing.T) {
	srv := &fallbackServer{releasesStatus: http.StatusInternalServerError}
	srv.start(t)

	releases, err := FetchReleasesWithFallback("owner/repo")
	if err == nil {
		t.Fatal("expected the Releases API error to be reported")
	}
	if !strings.Contains(err.Error(), "GitHub API error 500") {
		t.Errorf("expected the original API error, got %v", err)
	}
	if len(releases) != 0 {
		t.Errorf("expected no releases alongside the error, got %+v", releases)
	}
}
