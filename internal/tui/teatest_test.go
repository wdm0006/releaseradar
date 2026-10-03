package tui

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/wdm0006/releaseradar/internal/github"
)

func startTeaTest(t *testing.T, repos []string, releases []github.Release, fetch fetchFunc) *teatest.TestModel {
	t.Helper()
	isolateConfig(t)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "cache"))
	t.Setenv("RELEASERADAR_START_TAB", "")
	t.Setenv("RELEASERADAR_NO_ALTSCREEN", "1")
	t.Setenv("COLUMNS", "150")
	t.Setenv("LINES", "40")
	m := newTestModel(repos, releases)
	m.fetch = fetch
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(150, 40))
	t.Cleanup(func() {
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
		tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	})
	return tm
}

// Output accumulates frames: wait for appearances, never disappearance.
func waitTeaText(t *testing.T, tm *teatest.TestModel, text string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains([]byte(ansi.Strip(string(b))), []byte(text))
	}, teatest.WithDuration(5*time.Second))
}

func finishTeaTest(t *testing.T, tm *teatest.TestModel) Model {
	t.Helper()
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	m, ok := tm.FinalModel(t).(Model)
	if !ok {
		t.Fatalf("FinalModel returned %T, want Model", tm.FinalModel(t))
	}
	return m
}

func TestTeaRefreshMixedResults(t *testing.T) {
	release := github.Release{Repo: "owner/alpha", Name: "Alpha refreshed", TagName: "v1", PublishedAt: "2026-01-01T00:00:00Z"}
	var calls atomic.Int32
	var alphaCalls atomic.Int32
	tm := startTeaTest(t, []string{"owner/alpha", "owner/beta"}, nil, func(repo string) ([]github.Release, error) {
		calls.Add(1)
		if repo == "owner/beta" {
			return nil, errors.New("unavailable")
		}
		if alphaCalls.Add(1) == 1 {
			initial := release
			initial.Name = "Alpha initial"
			return []github.Release{initial}, nil
		}
		return []github.Release{release}, nil
	})
	waitTeaText(t, tm, "Loaded 1 releases from 2 repos")
	tm.Type("r")
	waitTeaText(t, tm, "Alpha refreshed")
	m := finishTeaTest(t, tm)
	if m.status != "Loaded 1 releases from 2 repos (1 errors)" {
		t.Fatalf("status = %q", m.status)
	}
	if m.refreshing || m.loading {
		t.Fatal("fetch did not finish")
	}
	if !reflect.DeepEqual(m.allReleases, []github.Release{release}) {
		t.Fatalf("releases = %#v", m.allReleases)
	}
	if calls.Load() != 4 {
		t.Fatalf("fetch calls = %d, want 4", calls.Load())
	}
}

func TestTeaLoadingScreen(t *testing.T) {
	unblock := make(chan struct{})
	var once sync.Once
	release := github.Release{Repo: "owner/alpha", Name: "Alpha ready", TagName: "v1"}
	tm := startTeaTest(t, []string{"owner/alpha"}, nil, func(string) ([]github.Release, error) {
		<-unblock
		return []github.Release{release}, nil
	})
	// Registered after program cleanup so a failed assertion releases the worker first.
	t.Cleanup(func() { once.Do(func() { close(unblock) }) })
	waitTeaText(t, tm, "fetching owner/alpha")
	once.Do(func() { close(unblock) })
	waitTeaText(t, tm, "Alpha ready")
	m := finishTeaTest(t, tm)
	if m.loading {
		t.Fatal("loading screen remained active")
	}
	if !reflect.DeepEqual(m.releases.filteredReleases, []github.Release{release}) {
		t.Fatalf("release list = %#v", m.releases.filteredReleases)
	}
	if m.width != 150 || m.height != 40 {
		t.Fatalf("size = %dx%d", m.width, m.height)
	}
}

func TestTeaFilterJourney(t *testing.T) {
	releases := []github.Release{
		{Repo: "owner/alpha", Name: "Alpha", TagName: "v1", PublishedAt: "2026-01-02T00:00:00Z"},
		{Repo: "owner/beta", Name: "Beta", TagName: "v2", PublishedAt: "2026-01-01T00:00:00Z"},
	}
	tm := startTeaTest(t, []string{"owner/alpha", "owner/beta"}, releases, func(string) ([]github.Release, error) { return nil, nil })
	waitTeaText(t, tm, "Loaded 2 releases from 2 repos")
	tm.Type("/alpha")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	m := finishTeaTest(t, tm)
	if m.releases.filterText != "alpha" || m.releases.filtering {
		t.Fatalf("filter = %q, editing = %v", m.releases.filterText, m.releases.filtering)
	}
	if !reflect.DeepEqual(m.releases.filteredReleases, releases[:1]) {
		t.Fatalf("filtered releases = %#v", m.releases.filteredReleases)
	}
}

func TestTeaTabSwitching(t *testing.T) {
	tm := startTeaTest(t, []string{"owner/alpha"}, nil, func(string) ([]github.Release, error) { return nil, nil })
	waitTeaText(t, tm, "Loaded 0 releases from 1 repos")
	tm.Type("2")
	m := finishTeaTest(t, tm)
	if m.activeTab != tabRepos {
		t.Fatalf("active tab = %v, want Repositories", m.activeTab)
	}
}
