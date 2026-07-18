package github

import (
	"net/http"
	"net/http/httptest"
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
