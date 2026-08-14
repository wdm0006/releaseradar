package github

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// withTestServer points the package client at srv with a short per-request
// timeout, bypassing gh, and resets every package global on cleanup so tests
// stay isolated.
func withTestServer(t *testing.T, srv *httptest.Server, timeout time.Duration) {
	t.Helper()
	httpClient = &http.Client{Timeout: timeout}
	authToken = "test-token"
	apiBaseURL = srv.URL + "/"
	t.Cleanup(func() {
		clientOnce = sync.Once{}
		httpClient = nil
		authToken = ""
		clientErr = nil
		apiBaseURL = "https://api.github.com/"
	})
}

// withTokenSources drives the real ensureClient path against srv: GITHUB_TOKEN
// resolves to env and `gh auth token` to gh. It leaves httpClient nil so
// ensureClient actually selects a token source, and resets every package global
// on cleanup so tests stay isolated.
func withTokenSources(t *testing.T, srv *httptest.Server, env string, gh func() ([]byte, error)) {
	t.Helper()
	clientOnce = sync.Once{}
	httpClient = nil
	authToken = ""
	clientErr = nil
	if srv != nil {
		apiBaseURL = srv.URL + "/"
	}
	lookupEnv = func(key string) string {
		if key == "GITHUB_TOKEN" {
			return env
		}
		return ""
	}
	ghAuthToken = gh
	t.Cleanup(func() {
		clientOnce = sync.Once{}
		httpClient = nil
		authToken = ""
		clientErr = nil
		apiBaseURL = "https://api.github.com/"
		lookupEnv = os.Getenv
		ghAuthToken = runGHAuthToken
	})
}

// tokenEchoServer records the Authorization header of every request it serves.
func tokenEchoServer(t *testing.T, gotAuth *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEnsureClientPrefersGitHubToken(t *testing.T) {
	var gotAuth string
	srv := tokenEchoServer(t, &gotAuth)

	ghCalls := 0
	withTokenSources(t, srv, "  env-token\n", func() ([]byte, error) {
		ghCalls++
		return []byte("gh-token\n"), nil
	})

	if _, _, err := apiGet("repos/owner/repo/releases"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer env-token" {
		t.Fatalf("expected GITHUB_TOKEN to reach the Authorization header, got %q", gotAuth)
	}
	if ghCalls != 0 {
		t.Fatalf("expected gh auth token not to run, ran %d time(s)", ghCalls)
	}
}

func TestEnsureClientFallsBackToGH(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"unset", ""},
		{"whitespace only", "   \n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotAuth string
			srv := tokenEchoServer(t, &gotAuth)

			ghCalls := 0
			withTokenSources(t, srv, tc.env, func() ([]byte, error) {
				ghCalls++
				return []byte("gh-token\n"), nil
			})

			if _, _, err := apiGet("repos/owner/repo/releases"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotAuth != "Bearer gh-token" {
				t.Fatalf("expected gh token to reach the Authorization header, got %q", gotAuth)
			}
			if ghCalls != 1 {
				t.Fatalf("expected gh auth token to run once, ran %d time(s)", ghCalls)
			}
		})
	}
}

func TestEnsureClientGHErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantSub string
	}{
		{"gh missing", &exec.Error{Name: "gh", Err: exec.ErrNotFound}, "GitHub CLI (gh) not found"},
		{"gh unauthenticated", errors.New("exit status 1: gh-token-abc123"), "not authenticated with GitHub CLI"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withTokenSources(t, nil, "", func() ([]byte, error) {
				return nil, tc.err
			})

			_, _, err := apiGet("repos/owner/repo/releases")
			if err == nil {
				t.Fatal("expected an error when gh auth token fails")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("expected error containing %q, got %q", tc.wantSub, err.Error())
			}
			if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
				t.Fatalf("expected error to mention the GITHUB_TOKEN alternative, got %q", err.Error())
			}
			if strings.Contains(err.Error(), "gh-token-abc123") {
				t.Fatalf("error leaked gh output: %q", err.Error())
			}
		})
	}
}

func TestEnsureClientDoesNotLeakTokenInErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	withTokenSources(t, srv, "secret-env-token", func() ([]byte, error) {
		t.Error("gh auth token should not run when GITHUB_TOKEN is set")
		return nil, nil
	})

	_, _, err := apiGet("repos/owner/repo/releases")
	if err == nil {
		t.Fatal("expected an error for a 401 response")
	}
	if strings.Contains(err.Error(), "secret-env-token") {
		t.Fatalf("error leaked the token: %q", err.Error())
	}
}

func TestAPIGetTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	withTestServer(t, srv, 20*time.Millisecond)

	start := time.Now()
	_, _, err := apiGet("repos/owner/repo/releases")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from delayed response, got nil")
	}
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("expected request to be canceled before the response, waited %s", elapsed)
	}
}

func TestAPIGetSendsBearerToken(t *testing.T) {
	var gotAuth, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()
	withTestServer(t, srv, time.Second)

	if _, _, err := apiGet("repos/owner/repo/releases"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected bearer token header, got %q", gotAuth)
	}
	if gotAccept != "application/vnd.github+json" {
		t.Fatalf("unexpected Accept header %q", gotAccept)
	}
}

func TestAPIGetErrorMapping(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantSub string
	}{
		{"not found", http.StatusNotFound, "not found"},
		{"unauthorized", http.StatusUnauthorized, "authentication failed"},
		{"forbidden", http.StatusForbidden, "authentication failed"},
		{"server error", http.StatusInternalServerError, "GitHub API error 500"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()
			withTestServer(t, srv, time.Second)

			_, _, err := apiGet("repos/owner/repo/releases")
			if err == nil {
				t.Fatalf("expected error for status %d", tc.status)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("status %d: expected error containing %q, got %q", tc.status, tc.wantSub, err.Error())
			}
		})
	}
}

func TestFetchReleasesFollowsPagination(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/repos/owner/repo/releases" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", `<`+srv.URL+`/repos/owner/repo/releases?per_page=100&page=2>; rel="next"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"tag_name":"v1.0.0"}]`))
		case "2":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"tag_name":"v2.0.0"}]`))
		default:
			t.Errorf("unexpected page request: %s", r.URL.String())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	withTestServer(t, srv, time.Second)

	releases, err := FetchReleases("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases across pages, got %d", len(releases))
	}
	if releases[0].TagName != "v1.0.0" || releases[1].TagName != "v2.0.0" {
		t.Fatalf("unexpected release tags: %+v", releases)
	}
	if releases[0].Repo != "owner/repo" {
		t.Fatalf("expected repo to be set, got %q", releases[0].Repo)
	}
}
