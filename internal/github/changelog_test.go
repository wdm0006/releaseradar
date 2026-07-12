package github

import (
	"strings"
	"testing"
	"time"
)

func TestParseChangelogEntries_BracketedVersion(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	changelog := "# Changelog\n\n## [1.2.3] - " + today + "\n\n- Fixed a bug\n- Added feature\n\n## [1.2.2] - 2020-01-01\n\n- Old stuff\n"

	entries := ParseChangelogEntries(changelog, "owner/repo", "")

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", entries[0].Name)
	}
	if entries[0].Repo != "owner/repo" {
		t.Errorf("expected repo owner/repo, got %s", entries[0].Repo)
	}
	if !strings.Contains(entries[0].Body, "Fixed a bug") {
		t.Errorf("body missing expected content: %s", entries[0].Body)
	}
}

func TestParseChangelogEntries_UnbracketedVersion(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	changelog := "## 2.0.69 - " + today + "\n\n- Some change\n"

	entries := ParseChangelogEntries(changelog, "test/repo", "")

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "2.0.69" {
		t.Errorf("expected version 2.0.69, got %s", entries[0].Name)
	}
}

func TestParseChangelogEntries_ParenDate(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	changelog := "## 1.0.0 (" + today + ")\n\n- Initial release\n"

	entries := ParseChangelogEntries(changelog, "test/repo", "")

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", entries[0].Name)
	}
}

func TestParseChangelogEntries_NoDate_UsesFallback(t *testing.T) {
	changelog := "## v3.0.0\n\n- Big release\n"
	fallback := time.Now().UTC().Format(time.RFC3339)

	entries := ParseChangelogEntries(changelog, "test/repo", fallback)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "3.0.0" {
		t.Errorf("expected version 3.0.0, got %s", entries[0].Name)
	}
	if entries[0].PublishedAt != fallback {
		t.Errorf("expected fallback date %s, got %s", fallback, entries[0].PublishedAt)
	}
}

func TestParseChangelogEntries_Empty(t *testing.T) {
	entries := ParseChangelogEntries("", "test/repo", "")
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseChangelogEntries_MultipleRecent(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	changelog := "## [2.0.0] - " + today + "\n\n- Feature A\n\n## [1.9.0] - " + yesterday + "\n\n- Feature B\n"

	entries := ParseChangelogEntries(changelog, "test/repo", "")

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestParseChangelogEntries_SkipsBracketedUnreleased(t *testing.T) {
	changelog := "# Changelog\n\n## [Unreleased]\n\n- Work in progress\n- Not shipped yet\n\n## [1.2.3] - 2024-01-15\n\n- Fixed a bug\n\n## [1.2.2] - 2020-01-01\n\n- Old stuff\n"

	entries := ParseChangelogEntries(changelog, "owner/repo", "")

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name, "Unreleased") {
			t.Fatalf("did not expect an Unreleased entry, got %q", e.Name)
		}
	}
	if entries[0].Name != "1.2.3" {
		t.Errorf("expected first version 1.2.3, got %s", entries[0].Name)
	}
	if entries[0].PublishedAt != "2024-01-15T00:00:00Z" {
		t.Errorf("expected date 2024-01-15T00:00:00Z, got %s", entries[0].PublishedAt)
	}
	if !strings.Contains(entries[0].Body, "Fixed a bug") {
		t.Errorf("body missing expected content: %s", entries[0].Body)
	}
	if strings.Contains(entries[0].Body, "Work in progress") {
		t.Errorf("Unreleased body leaked into entry: %s", entries[0].Body)
	}
	if entries[1].Name != "1.2.2" {
		t.Errorf("expected second version 1.2.2, got %s", entries[1].Name)
	}
}

func TestParseChangelogEntries_SkipsUnbracketedUnreleased(t *testing.T) {
	changelog := "## Unreleased\n\n- Pending change\n\n## 1.0.0 - 2024-02-01\n\n- Initial release\n"

	entries := ParseChangelogEntries(changelog, "test/repo", "")

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", entries[0].Name)
	}
	if strings.Contains(entries[0].Body, "Pending change") {
		t.Errorf("Unreleased body leaked into entry: %s", entries[0].Body)
	}
}

func TestParseChangelogEntries_AuthorIsChangelog(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	changelog := "## [1.0.0] - " + today + "\n\n- Stuff\n"

	entries := ParseChangelogEntries(changelog, "test/repo", "")
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if entries[0].Author.Login != "changelog" {
		t.Errorf("expected author 'changelog', got %s", entries[0].Author.Login)
	}
}
