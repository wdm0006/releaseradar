package ai

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wdm0006/releaseradar/internal/github"
)

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		limit int
		want  string
	}{
		{"empty", "", 10, ""},
		{"short", "hello", 10, "hello"},
		{"exactly at limit", "helloworld", 10, "helloworld"},
		{"ascii over limit", "helloworld!", 10, "helloworld" + truncationMarker},
		// "é" is two bytes, so a limit of 5 lands inside the third one.
		{"multi-byte boundary", "ééé", 5, "éé" + truncationMarker},
		// Every rune boundary is beyond the limit, so nothing survives.
		{"single rune wider than limit", "日", 2, truncationMarker},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateBody(tt.body, tt.limit)
			if got != tt.want {
				t.Errorf("truncateBody(%q, %d) = %q, want %q", tt.body, tt.limit, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("truncateBody(%q, %d) = %q, which is not valid UTF-8", tt.body, tt.limit, got)
			}
		})
	}
}

// oversizedMultiByteBody returns a body whose byte at index maxBodyBytes falls
// inside a multi-byte rune, plus the exact text a correct truncation keeps.
func oversizedMultiByteBody() (body, kept string) {
	const accent = "é" // 2 bytes
	// One leading ASCII byte makes every subsequent rune start at an odd offset,
	// so maxBodyBytes (even) lands mid-rune.
	accents := maxBodyBytes // more than enough to exceed the limit
	body = "a" + strings.Repeat(accent, accents)
	kept = "a" + strings.Repeat(accent, (maxBodyBytes-1)/2)
	return body, kept
}

func TestBuildSummaryPromptBoundsOversizedASCIIBody(t *testing.T) {
	body := strings.Repeat("A", maxBodyBytes) + "TAIL_SENTINEL"
	prompt := buildSummaryPrompt([]github.Release{
		{Repo: "owner/repo", TagName: "v1.0.0", PublishedAt: "2026-01-02T00:00:00Z", Body: body},
	})

	want := strings.Repeat("A", maxBodyBytes) + truncationMarker
	if !strings.Contains(prompt, want) {
		t.Errorf("summary prompt does not contain the bounded body + truncation marker")
	}
	if strings.Contains(prompt, "TAIL_SENTINEL") {
		t.Error("summary prompt contains release-note text beyond the per-release limit")
	}
	if strings.Count(prompt, "A") > maxBodyBytes {
		t.Errorf("summary prompt carries %d body bytes, want at most %d", strings.Count(prompt, "A"), maxBodyBytes)
	}
}

func TestBuildSummaryPromptPreservesUTF8(t *testing.T) {
	body, kept := oversizedMultiByteBody()
	prompt := buildSummaryPrompt([]github.Release{
		{Repo: "owner/repo", TagName: "v1.0.0", PublishedAt: "2026-01-02T00:00:00Z", Body: body},
	})

	if !utf8.ValidString(prompt) {
		t.Fatal("summary prompt is not valid UTF-8")
	}
	if !strings.Contains(prompt, kept+truncationMarker) {
		t.Error("summary prompt does not end the release body on the expected rune boundary")
	}
	assertJSONRoundTrip(t, prompt)
}

func TestBuildChatPromptBoundsOversizedASCIIBody(t *testing.T) {
	body := strings.Repeat("A", maxBodyBytes) + "TAIL_SENTINEL"
	prompt := buildChatPrompt("what changed?", []github.Release{
		{Repo: "owner/repo", TagName: "v1.0.0", PublishedAt: "2026-01-02T00:00:00Z", Body: body},
	})

	want := strings.Repeat("A", maxBodyBytes) + truncationMarker
	if !strings.Contains(prompt, want) {
		t.Error("chat prompt does not contain the bounded body + truncation marker")
	}
	if strings.Contains(prompt, "TAIL_SENTINEL") {
		t.Error("chat prompt contains release-note text beyond the per-release limit")
	}
	if !strings.Contains(prompt, "what changed?") {
		t.Error("chat prompt does not contain the user question")
	}
}

func TestBuildChatPromptPreservesUTF8(t *testing.T) {
	body, kept := oversizedMultiByteBody()
	prompt := buildChatPrompt("what changed?", []github.Release{
		{Repo: "owner/repo", TagName: "v1.0.0", PublishedAt: "2026-01-02T00:00:00Z", Body: body},
	})

	if !utf8.ValidString(prompt) {
		t.Fatal("chat prompt is not valid UTF-8")
	}
	if !strings.Contains(prompt, kept+truncationMarker) {
		t.Error("chat prompt does not end the release body on the expected rune boundary")
	}
	assertJSONRoundTrip(t, prompt)
}

// assertJSONRoundTrip checks that the prompt survives the JSON encoding the
// OpenAI request goes through: encoding/json silently replaces invalid UTF-8
// with U+FFFD, so a prompt cut mid-rune would come back corrupted.
func assertJSONRoundTrip(t *testing.T, prompt string) {
	t.Helper()
	encoded, err := json.Marshal(prompt)
	if err != nil {
		t.Fatalf("failed to marshal prompt: %v", err)
	}
	var decoded string
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("failed to unmarshal prompt: %v", err)
	}
	if decoded != prompt {
		t.Error("prompt was corrupted by JSON encoding (invalid UTF-8 replaced)")
	}
	if strings.ContainsRune(decoded, utf8.RuneError) {
		t.Error("JSON-encoded prompt contains the Unicode replacement character")
	}
}

func TestBuildPromptsKeepShortAndEmptyBodies(t *testing.T) {
	releases := []github.Release{
		{Repo: "owner/short", TagName: "v1.0.0", PublishedAt: "2026-01-02T00:00:00Z", Body: "Small note."},
		{Repo: "owner/empty", TagName: "v2.0.0", PublishedAt: "2026-01-03T00:00:00Z", Body: ""},
	}

	for name, prompt := range map[string]string{
		"summary": buildSummaryPrompt(releases),
		"chat":    buildChatPrompt("q", releases),
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(prompt, "Small note.") {
				t.Error("prompt dropped a body that is under the limit")
			}
			if !strings.Contains(prompt, "No description") {
				t.Error("prompt lost the empty-body placeholder")
			}
			if strings.Contains(prompt, truncationMarker) {
				t.Error("prompt marked an under-limit body as truncated")
			}
		})
	}
}

func TestBuildPromptsKeepReleaseCountLimits(t *testing.T) {
	makeReleases := func(n int) []github.Release {
		releases := make([]github.Release, 0, n)
		for i := 0; i < n; i++ {
			releases = append(releases, github.Release{
				Repo:        "owner/repo",
				TagName:     "v1.0.0",
				PublishedAt: "2026-01-02T00:00:00Z",
				Body:        "BODY_MARKER",
			})
		}
		return releases
	}

	summary := buildSummaryPrompt(makeReleases(maxSummaryReleases + 5))
	if got := strings.Count(summary, "BODY_MARKER"); got != maxSummaryReleases {
		t.Errorf("summary prompt included %d releases, want %d", got, maxSummaryReleases)
	}

	chat := buildChatPrompt("q", makeReleases(maxChatReleases+5))
	if got := strings.Count(chat, "BODY_MARKER"); got != maxChatReleases {
		t.Errorf("chat prompt included %d releases, want %d", got, maxChatReleases)
	}
}
