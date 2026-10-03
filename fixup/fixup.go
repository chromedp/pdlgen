// Package fixup alters the type definitions for the Chrome DevTools Protocol
// domains prior to code generation.
//
// Two fixes are applied. Neither adds anything that the protocol does not
// define.
//
// The first extracts each inline enum into a named type, so that a value of the
// enum has a type of its own and a constant for each value. See extractEnums.
//
// The second removes name stuttering: any type whose name is prefixed with the
// name of its domain (for example, the "CSS" domain's "CSSStyle" type, or
// "AXNode" in the "Accessibility" domain) has the prefix removed, as the Go
// package name already provides it (for example, css.Style).
package fixup

import (
	"regexp"
	"strings"

	"github.com/chromedp/pdlgen/pdl"
)

var axRE = regexp.MustCompile(`^AX`)

// FixDomains extracts the inline enums of the domains into named types, and
// removes name stuttering from the types.
func FixDomains(domains []*pdl.Domain) error {
	for _, d := range domains {
		if err := extractEnums(d); err != nil {
			return err
		}
	}
	for _, d := range domains {
		for _, t := range d.Types {
			if t.IsCircularDep {
				continue
			}
			name := strings.TrimPrefix(t.RawName, d.Domain.String()+".")
			name = strings.TrimPrefix(name, d.Domain.String())
			if after, ok := strings.CutPrefix(t.RawName, "Accessibility."); ok {
				name = axRE.ReplaceAllString(after, "")
			}
			if name != "" && t.Name != name {
				t.Name = name
			}
		}
	}
	return nil
}
