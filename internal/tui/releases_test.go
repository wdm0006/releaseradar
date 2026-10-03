package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wdm0006/releaseradar/internal/github"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

func TestSetReleasesInvalidatesRenderedDetail(t *testing.T) {
	m := newReleasesModel().setSize(160, 40)
	release := github.Release{
		Repo:    "owner/repo",
		Name:    "Release",
		TagName: "v1.0.0",
		Body:    "old release notes",
	}
	m = m.setReleases([]github.Release{release})

	release.Body = "updated release notes"
	m = m.setReleases([]github.Release{release})

	if got := ansiEscape.ReplaceAllString(m.detail.View(), ""); !strings.Contains(got, release.Body) {
		t.Fatalf("detail pane does not contain updated body %q:\n%s", release.Body, got)
	}
}

func TestFilterKeystrokeCompletesWithinBudget(t *testing.T) {
	const releaseCount = 500
	body := strings.Repeat("- release note with **markdown** and a [link](https://example.com)\n", 18)
	releases := make([]github.Release, releaseCount)
	for i := range releases {
		releases[i] = github.Release{
			Repo:    "owner/repo",
			Name:    fmt.Sprintf("Release %d", i),
			TagName: fmt.Sprintf("v%d.0.0", i),
			Body:    body,
		}
	}

	m := newReleasesModel().setSize(160, 40).setReleases(releases)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	start := time.Now()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	elapsed := time.Since(start)
	t.Logf("500-release filter keystroke completed in %v", elapsed)

	if len(m.filteredReleases) != releaseCount {
		t.Fatalf("filtered releases = %d, want %d", len(m.filteredReleases), releaseCount)
	}
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("filter keystroke took %v, want less than 200ms", elapsed)
	}
}
