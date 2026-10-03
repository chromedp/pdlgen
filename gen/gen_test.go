package gen_test

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/chromedp/pdlgen/fixup"
	"github.com/chromedp/pdlgen/gen"
	"github.com/chromedp/pdlgen/pdl"
)

const testPDL = `# Test protocol.
version
  major 1
  minor 3

# Target domain.
domain Target
  # Unique session identifier.
  type SessionID extends string

# Network domain.
domain Network
  # The partition key of a cookie.
  type CookiePartitionKey extends object
    properties
      string topLevelSite
      boolean hasCrossSiteAncestor

# Input domain.
domain Input
  # A point of touch.
  type TouchPoint extends object
    properties
      number x
      # Identifier of the touch source. Zero is an identifier.
      optional number id
      optional number rotationAngle

# Page domain.
domain Page
  # Unique frame identifier.
  type FrameId extends string

  # Page load result.
  type PageResult extends object
    properties
      FrameId frameId
      optional string errorText

  # Navigates the current page.
  command navigate
    parameters
      # URL to navigate to.
      string url
      optional string referrer
      optional boolean replace
      optional Headers headers
      optional enum format
        jpeg
        png
    returns
      FrameId frameId
      optional string errorText
      optional binary data

  # Request headers as keys and values of a JSON object.
  type Headers extends object

  # A stored token.
  type Token extends object
    properties
      # Base64-encoded serialized token.
      string data

  # Captures a screenshot.
  command captureScreenshot
    parameters
      # Compression quality. Zero is a quality.
      optional integer quality
      # Compression effort (default: 0).
      optional integer effort
    returns
      optional integer quality

  # Closes the page.
  command close

  # Fired when the load event fires.
  event loadEventFired
    parameters
      number timestamp

  # Fired when the page is closed.
  event closed
`

func TestGoGenerator(t *testing.T) {
	p, err := pdl.Parse([]byte(testPDL))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, d := range p.Domains {
		for _, typs := range [][]*pdl.Type{d.Events, d.Commands} {
			for _, typ := range typs {
				if typ.Parameters == nil {
					typ.Parameters = []*pdl.Type{}
				}
			}
		}
	}
	if err := fixup.FixDomains(p.Domains); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	e, err := gen.NewGoGenerator(p.Domains, "github.com/chromedp/cdproto", gen.Versions{Chromium: "1.2.3.4", V8: "5.6.7"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	files := e.Emit()
	for _, name := range []string{"cdproto.go", "version.go", "cdp/types.go", "page/page.go", "page/types.go", "page/events.go"} {
		buf, ok := files[name]
		if !ok {
			t.Fatalf("expected file %s to be generated", name)
		}
		if _, err := format.Source(buf.Bytes()); err != nil {
			t.Errorf("%s is not valid Go: %v\n%s", name, err, buf)
		}
		// tools find a generated file by the comment before the package clause
		f, err := parser.ParseFile(token.NewFileSet(), name, buf.Bytes(), parser.ParseComments|parser.PackageClauseOnly)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		} else if !ast.IsGenerated(f) {
			t.Errorf("expected %s to be marked as generated before the package clause", name)
		}
	}
	for name, want := range map[string][]string{
		"version.go":     {`chromiumVersion = "1.2.3.4"`, `v8Version       = "5.6.7"`},
		"page/page.go":   {"var Navigate = cdp.Command[NavigateParams, NavigateResult]{Method: CommandNavigate}", "type NavigateResult struct", "var Close = cdp.Command[cdp.Empty, cdp.Empty]{Method: CommandClose}", "Replace *bool", "Data []byte", "var LoadEventFired = cdp.Event[EventLoadEventFired]{Method: \"Page.loadEventFired\"}", "var Closed = cdp.Event[EventClosed]{Method: \"Page.closed\"}", "CommandNavigate = \"Page.navigate\""},
		"page/types.go":  {"type Result struct", "type Headers map[string]any", "Data []byte", "type NavigateFormat string", "NavigateFormatJpeg NavigateFormat = \"jpeg\""},
		"page/events.go": {"type EventLoadEventFired struct", "type EventClosed struct"},
	} {
		got := files[name].String()
		for _, s := range want {
			if !strings.Contains(got, s) {
				t.Errorf("%s: expected output to contain %q", name, s)
			}
		}
	}
	// an optional number that the generator lists is a pointer in a type and in
	// the parameters of a command, so that a pointer to zero is sent and nil is
	// left out. A number that is not listed, and a result, stay plain.
	if got := files["input/types.go"].String(); !strings.Contains(got, "ID *float64 `json:\"id,omitempty,omitzero\"`") || !strings.Contains(got, "RotationAngle float64 `json:\"rotationAngle,omitempty,omitzero\"`") {
		t.Errorf("expected a pointer id and a plain rotation angle, got:\n%s", got)
	}
	got := files["page/page.go"].String()
	for _, want := range []string{"Quality *int64 `json:\"quality,omitempty,omitzero\"`", "Effort int64 `json:\"effort,omitempty,omitzero\"`", "Quality int64 `json:\"quality,omitempty,omitzero\"`"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, got)
		}
	}
	// the typed API does not hide the connection in the context, so the package
	// cdp has no error for a bad context
	if got := files["cdp/types.go"].String(); strings.Contains(got, "ErrInvalidContext") {
		t.Errorf("expected cdp to have no ErrInvalidContext, got:\n%s", got)
	}
	// the partition key of a cookie decodes the old string form too
	if got := files["network/types.go"].String(); !strings.Contains(got, "func (t *CookiePartitionKey) UnmarshalJSON(buf []byte) error") {
		t.Errorf("expected CookiePartitionKey to have an UnmarshalJSON method, got:\n%s", got)
	}
	// an object type without properties is a map, and a map is not a pointer
	if got := files["page/page.go"].String(); strings.Contains(got, "*Headers") {
		t.Errorf("expected a map parameter to be no pointer, got:\n%s", got)
	}
}
