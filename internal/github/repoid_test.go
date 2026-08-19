package github

import (
	"strings"
	"testing"
)

func TestParseRepo(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"canonical", "owner/repo", "owner/repo"},
		{"surrounding whitespace", "  owner/repo  ", "owner/repo"},
		{"https url", "https://github.com/owner/repo", "owner/repo"},
		{"http url", "http://github.com/owner/repo", "owner/repo"},
		{"url with www", "https://www.github.com/owner/repo", "owner/repo"},
		{"host without scheme", "github.com/owner/repo", "owner/repo"},
		{"trailing slash", "https://github.com/owner/repo/", "owner/repo"},
		{"clone url", "https://github.com/owner/repo.git", "owner/repo"},
		{"dot repo name", "owner/.github", "owner/.github"},
		{"allowed punctuation", "my-org/my_repo.js", "my-org/my_repo.js"},
		{"uppercase host", "HTTPS://GitHub.com/owner/repo", "owner/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRepo(tt.input)
			if err != nil {
				t.Fatalf("ParseRepo(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ParseRepo(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRepoRejects(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"no slash", "repo"},
		{"missing owner", "/repo"},
		{"missing name", "owner/"},
		{"extra path segments", "owner/repo/tree/main"},
		{"query string", "owner/repo?per_page=1&foo="},
		{"traversal", "../../users/octocat"},
		{"traversal segment", "owner/.."},
		{"dot segment", "./repo"},
		{"url with extra segments", "https://github.com/owner/repo/tree/main"},
		{"space inside", "owner/my repo"},
		{"other host", "gitlab.com/owner/repo"},
		{"trailing newline injection", "owner/repo\nowner2/repo2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRepo(tt.input)
			if err == nil {
				t.Fatalf("ParseRepo(%q) = %q, want an error", tt.input, got)
			}
			if got != "" {
				t.Fatalf("ParseRepo(%q) returned %q alongside an error", tt.input, got)
			}
			if !strings.Contains(err.Error(), "owner/repo") {
				t.Fatalf("ParseRepo(%q) error %q does not name the expected format", tt.input, err)
			}
		})
	}
}
