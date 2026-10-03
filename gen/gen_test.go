package gen_test

import (
	"go/format"
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
}
