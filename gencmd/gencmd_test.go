package gencmd_test

import (
	"bytes"
	"context"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chromedp/pdlgen/gencmd"
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
  # A WebSocket message.
  type WebSocketFrame extends object
    properties
      # WebSocket message opcode.
      number opcode
      # WebSocket message payload data. If the opcode is 1, it is text. If not, it is base64.
      string payloadData

# HAR domain.
domain HAR
  # The content of a response.
  type Content extends object
    properties
      # The text, plain or encoded.
      optional string text
      # The encoding of the text, for example "base64".
      optional string encoding

# Page domain.
domain Page
  # Navigates the current page.
  command navigate
    parameters
      # URL to navigate to.
      string url
      optional boolean replace
      optional enum format
        jpeg
        png
    returns
      string frameId
      optional binary data

  # Closes the page.
  command close

  # Returns the body of the page.
  command getBody
    returns
      # The body, as text or as base64.
      string body
      # True, if the body is base64.
      boolean base64Encoded

  # Fired when a body arrives, as text or as base64.
  event bodyReceived
    parameters
      # The body.
      string body
      # True, if the body is base64.
      boolean base64Encoded

  # Fired when the load event fires.
  event loadEventFired
    parameters
      number timestamp
`

func TestExec(t *testing.T) {
	dir := t.TempDir()
	pdlFile := filepath.Join(dir, "test.pdl")
	if err := os.WriteFile(pdlFile, []byte(testPDL), 0o644); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := filepath.Join(dir, "out")
	// a file that the generator does not write, and one that it keeps
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, name := range []string{"stale.go", "go.mod"} {
		if err := os.WriteFile(filepath.Join(out, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	}
	args := gencmd.New()
	args.PDL = pdlFile
	args.Chromium = "1.2.3.4"
	args.V8 = "5.6.7"
	args.Cache = filepath.Join(dir, "cache")
	args.Out = out
	if err := args.Exec(context.Background(), new(bytes.Buffer), nil); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, name := range []string{"cdproto.go", "version.go", "cdp/types.go", "page/page.go", "page/events.go"} {
		buf, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatalf("expected %s to be written: %v", name, err)
		}
		if _, err := format.Source(buf); err != nil {
			t.Errorf("%s is not valid Go: %v", name, err)
		}
	}
	buf, err := os.ReadFile(filepath.Join(out, "version.go"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(string(buf), `"1.2.3.4"`) || !strings.Contains(string(buf), `"5.6.7"`) {
		t.Errorf("expected version.go to hold the versions, got:\n%s", buf)
	}
	if _, err := os.Stat(filepath.Join(out, "stale.go")); err == nil {
		t.Errorf("expected stale.go to be removed")
	}
	if _, err := os.Stat(filepath.Join(out, "go.mod")); err != nil {
		t.Errorf("expected go.mod to be kept: %v", err)
	}
}

// TestGeneratedCodeWorks generates a package from a small protocol, and runs a
// test of the typed API against it with the go command.
func TestGeneratedCodeWorks(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("the go command is not on the path")
	}
	dir := t.TempDir()
	pdlFile := filepath.Join(dir, "test.pdl")
	if err := os.WriteFile(pdlFile, []byte(testPDL), 0o644); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "go.mod"), []byte("module github.com/chromedp/cdproto\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	args := gencmd.New()
	args.PDL, args.Chromium, args.V8, args.Cache, args.Out = pdlFile, "1.2.3.4", "5.6.7", filepath.Join(dir, "cache"), out
	if err := args.Exec(context.Background(), new(bytes.Buffer), nil); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	buf, err := os.ReadFile(filepath.Join("testdata", "generated_test.go.txt"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "generated_test.go"), buf, 0o644); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	cmd := exec.Command(goTool, "test", "./...")
	cmd.Dir = out
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("expected the tests of the generated code to pass, got: %v\n%s", err, b)
	}
}

func TestExecErrors(t *testing.T) {
	tests := []struct {
		name string
		set  func(*gencmd.Args)
		cli  []string
	}{
		{"arguments", func(*gencmd.Args) {}, []string{"extra"}},
		{"ttl", func(a *gencmd.Args) { a.TTL = "soon" }, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := gencmd.New()
			test.set(args)
			if err := args.Exec(context.Background(), new(bytes.Buffer), test.cli); err == nil {
				t.Errorf("expected an error")
			}
		})
	}
}
