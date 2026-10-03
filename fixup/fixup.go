// Package fixup changes the type definitions of the Chrome DevTools Protocol
// domains before code generation.
//
// The package makes two fixes. Neither adds anything that the protocol does not
// define.
//
// The first fix extracts each inline enum into a named type. A value of the
// enum then has a type of its own and a constant for each value. See
// extractEnums.
//
// The second fix removes name stuttering. When the name of a type starts with
// the name of its domain, the fix removes that prefix, because the Go package
// name already provides it. For example, the type "CSSStyle" of the domain
// "CSS" becomes css.Style, and "AXNode" of "Accessibility" becomes
// accessibility.Node.
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
