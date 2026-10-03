package relnotes_test

import (
	"strings"
	"testing"

	"github.com/chromedp/pdlgen/relnotes"
)

const sample = `Incompatible changes:
- ./audits.(*Foo).UnmarshalJSON: removed
- ./audits.(*Bar).UnmarshalJSON: removed
- ./audits.(*Baz).UnmarshalJSON: removed
- ./audits.(*Qux).UnmarshalJSON: removed
- ./css.Style: removed
- ./input.DispatchKeyEventParams.Type: changed from string to DispatchKeyEventType
Compatible changes:
- ./input.DispatchKeyEventType: added
- ./input.DispatchKeyEventTypeChar: added
`

func parse(t *testing.T, s string) *relnotes.Report {
	t.Helper()
	r, err := relnotes.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return r
}

func TestParse(t *testing.T) {
	r := parse(t, sample)
	if len(r.Changes) != 8 {
		t.Fatalf("expected 8 changes, got: %d", len(r.Changes))
	}
	if inc, comp := r.Count(); inc != 6 || comp != 2 {
		t.Errorf("expected 6 incompatible and 2 compatible, got: %d, %d", inc, comp)
	}
	c := r.Changes[0]
	if c.Package != "audits" || c.Name != "Foo.UnmarshalJSON" || c.Kind != relnotes.Removed || !c.Incompatible {
		t.Errorf("unexpected change: %+v", c)
	}
	if c := r.Changes[5]; c.Kind != relnotes.Changed || c.Detail != "string -> DispatchKeyEventType" {
		t.Errorf("unexpected change: %+v", c)
	}
	if _, err := relnotes.Parse(strings.NewReader("surprise\n")); err == nil {
		t.Errorf("expected an error for an unexpected line")
	}
}

func TestCommit(t *testing.T) {
	r := parse(t, sample)
	got := r.Commit(relnotes.Meta{
		Version: "v0.157.1", Previous: "v0.157.0",
		Chromium: "157.0.8084.4", V8: "15.7.24", PrevChromium: "157.0.8084.3", PrevV8: "15.7.23",
		AddedPackages: []string{"findinpage"},
	})
	for _, want := range []string{
		"Updating to 157.0.8084.4_15.7.24 definitions\n\n- Release: v0.157.1\n",
		"- Chromium: 157.0.8084.4 (was 157.0.8084.3)",
		"- V8: 15.7.24 (was 15.7.23)",
		"- API changes since v0.157.0: 6 incompatible, 2 compatible",
		"- Packages added: findinpage",
		"audits                      4 removed",
		"audits: UnmarshalJSON on 4 types",
		"css: Style",
		"input: DispatchKeyEventParams.Type (string -> DispatchKeyEventType)",
		"input: DispatchKeyEventType, DispatchKeyEventTypeChar",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected the message to contain %q, got:\n%s", want, got)
		}
	}
}

func TestChangelogHasNoLists(t *testing.T) {
	got := parse(t, sample).Changelog(relnotes.Meta{Version: "v0.157.1", Previous: "v0.157.0", Chromium: "1", V8: "2", Date: "2026-10-03"})
	if !strings.HasPrefix(got, "## v0.157.1 - 2026-10-03\n\n") || strings.Contains(got, "Removed:") {
		t.Errorf("unexpected changelog:\n%s", got)
	}
}

func TestFirstRelease(t *testing.T) {
	got := parse(t, "").Tag(relnotes.Meta{Version: "v0.157.0", Chromium: "157.0.8084.3", V8: "15.7.23"})
	if !strings.HasPrefix(got, "cdproto v0.157.0\n\n") || !strings.Contains(got, "- API changes: first tagged release") {
		t.Errorf("unexpected tag annotation:\n%s", got)
	}
}

func TestLongListsAreShortened(t *testing.T) {
	var b strings.Builder
	b.WriteString("Compatible changes:\n")
	for _, n := range "ABCDEFGHIJKLMNOPQRST" {
		b.WriteString("- ./page." + string(n) + ": added\n")
	}
	got := parse(t, b.String()).Commit(relnotes.Meta{Version: "v1", Previous: "v0", Chromium: "1", V8: "2"})
	if !strings.Contains(got, "and 8 more") {
		t.Errorf("expected a shortened list, got:\n%s", got)
	}
}
