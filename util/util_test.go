package util

import "testing"

func TestFindTag(t *testing.T) {
	refs := map[string]Ref{
		"refs/tags/15.7.23-pgo": {Value: "abc"},
		"refs/tags/15.7.23":     {Value: "abc"},
		"refs/tags/15.7.22":     {Value: "def"},
		"refs/heads/main":       {Value: "abc"},
	}
	// map order is random, so ask many times
	for range 200 {
		if got := findTag(refs, "abc"); got != "15.7.23" {
			t.Fatalf("expected 15.7.23, got: %q", got)
		}
	}
	if got := findTag(refs, "def"); got != "15.7.22" {
		t.Errorf("expected 15.7.22, got: %q", got)
	}
	if got := findTag(refs, "none"); got != "" {
		t.Errorf("expected no tag, got: %q", got)
	}
}
