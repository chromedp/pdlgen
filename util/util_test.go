package util

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

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

// tagsPage is the part of the page of a gitiles project that lists the tags.
// Before October 2026 the page had no white space around the text of a tag. It
// has newlines and spaces now. Both must work.
const (
	tagsPageCompact = `<html><body><div class="RefList"><h3 class="RefList-title">Branches</h3><ul class="RefList-items"><li class="RefList-item"><a href="/x">main</a></li></ul></div>` +
		`<div class="RefList"><h3 class="RefList-title">Tags</h3><ul class="RefList-items">` +
		`<li class="RefList-item"><a href="/a">157.0.8103.22</a></li><li class="RefList-item"><a href="/b">158.0.8124.1</a></li>` +
		`<li class="RefList-item"><a href="/c">158.0.8123.0</a></li><li class="RefList-item"><a href="/d">not-a-version</a></li></ul></div></body></html>`
	tagsPageIndented = `<html><body>
    <div class="RefList">
      <h3 class="RefList-title">Branches</h3>
      <ul class="RefList-items">
        <li class="RefList-item">
          <a href="/x">main</a>
        </li>
      </ul>
    </div>
    <div class="RefList RefList--responsive">
      <h3 class="RefList-title">Tags</h3>
      <ul class="RefList-items">
        <li class="RefList-item">
          <a href="/a">157.0.8103.22</a>
        </li>
        <li class="RefList-item">
          <a href="/b">158.0.8124.1</a>
        </li>
        <li class="RefList-item">
          <a href="/c">158.0.8123.0</a>
        </li>
        <li class="RefList-item">
          <a href="/d">not-a-version</a>
        </li>
      </ul>
    </div>
  </body></html>`
)

func TestGetLatestVersion(t *testing.T) {
	for name, page := range map[string]string{"compact": tagsPageCompact, "indented": tagsPageIndented} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, page)
			}))
			defer srv.Close()
			got, err := GetLatestVersion(Cache{URL: srv.URL, Path: filepath.Join(t.TempDir(), "page.html")})
			if err != nil {
				t.Fatal(err)
			}
			if got != "158.0.8124.1" {
				t.Errorf("expected 158.0.8124.1, got: %q", got)
			}
		})
	}
	// a page without tags is an error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><body>no tags</body></html>")
	}))
	defer srv.Close()
	if _, err := GetLatestVersion(Cache{URL: srv.URL, Path: filepath.Join(t.TempDir(), "page.html")}); err == nil {
		t.Error("expected an error for a page without tags")
	}
}
