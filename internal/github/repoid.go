package github

import (
	"fmt"
	"strings"
)

// ParseRepo validates and normalizes a tracked repository identifier, returning
// the canonical "owner/name" form. Common paste forms (a github.com URL, a
// trailing ".git" or "/") are normalized; anything else that is not exactly two
// segments of GitHub's allowed identifier characters is rejected.
func ParseRepo(input string) (string, error) {
	s := strings.TrimSpace(input)

	for _, scheme := range []string{"https://", "http://"} {
		if len(s) >= len(scheme) && strings.EqualFold(s[:len(scheme)], scheme) {
			s = s[len(scheme):]
			break
		}
	}
	s = trimPrefixFold(s, "www.")
	s = trimPrefixFold(s, "github.com/")
	s = strings.TrimSuffix(s, "/")
	if trimmed := strings.TrimSuffix(s, ".git"); trimmed != "" {
		s = trimmed
	}

	owner, name, found := strings.Cut(s, "/")
	if !found || !validRepoSegment(owner) || !validRepoSegment(name) {
		return "", fmt.Errorf("invalid repository %q: use owner/repo (letters, digits, '.', '_' and '-' only)", input)
	}

	return owner + "/" + name, nil
}

func trimPrefixFold(s, prefix string) string {
	if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
		return s[len(prefix):]
	}
	return s
}

func validRepoSegment(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}
