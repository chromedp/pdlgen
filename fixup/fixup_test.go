package fixup_test

import (
	"strings"
	"testing"

	"github.com/chromedp/pdlgen/fixup"
	"github.com/chromedp/pdlgen/pdl"
)

const testPDL = `# Test protocol.
version
  major 1
  minor 3

# Input domain.
domain Input
  # Dispatches a key event.
  command dispatchKeyEvent
    parameters
      # Type of the key event.
      enum type
        keyDown
        keyUp
      # Locations of the key.
      array of enum locations
        left
        right

  # Fired when a key was dispatched.
  event dispatched
    parameters
      enum source
        keyboard
        script

  # A key description.
  type KeyInfo extends object
    properties
      enum kind
        printable
        control
`

func TestExtractEnums(t *testing.T) {
	p, err := pdl.Parse([]byte(testPDL))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := fixup.FixDomains(p.Domains); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	d := p.Domains[0]
	got := make(map[string][]string)
	for _, typ := range d.Types {
		got[typ.Name] = typ.Enum
	}
	for name, want := range map[string]string{
		"DispatchKeyEventType":      "keyDown keyUp",
		"DispatchKeyEventLocations": "left right",
		"DispatchedSource":          "keyboard script",
		"KeyInfoKind":               "printable control",
	} {
		if v, ok := got[name]; !ok || strings.Join(v, " ") != want {
			t.Errorf("expected enum type %s with values %q, got: %v", name, want, v)
		}
	}
	// the parameters now refer to the types, and an array holds them
	params := d.Commands[0].Parameters
	if p := params[0]; p.Enum != nil || p.Ref != "DispatchKeyEventType" {
		t.Errorf("expected type to refer to DispatchKeyEventType, got: %+v", p)
	}
	if p := params[1]; p.Enum != nil || p.Items == nil || p.Items.Ref != "DispatchKeyEventLocations" {
		t.Errorf("expected locations to be an array of DispatchKeyEventLocations, got: %+v", p)
	}
}

func TestExtractEnumsRejectsACollision(t *testing.T) {
	p, err := pdl.Parse([]byte(testPDL + `
  # Takes the name that the enum of the command needs.
  type DispatchKeyEventType extends string
`))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := fixup.FixDomains(p.Domains); err == nil {
		t.Errorf("expected an error for the colliding name")
	}
}
