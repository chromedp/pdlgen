// Package fixup alters the type definitions for the Chrome DevTools Protocol
// domains prior to code generation.
//
// The only fix applied is removing name stuttering: any type whose name is
// prefixed with the name of its domain (for example, the "CSS" domain's
// "CSSStyle" type, or "AXNode" in the "Accessibility" domain) has the prefix
// removed, as the Go package name already provides it (ie, css.Style).
package fixup

import (
	"regexp"
	"strings"

	"github.com/chromedp/pdlgen/pdl"
)

var axRE = regexp.MustCompile(`^AX`)

// FixDomains removes name stuttering from the types defined in the domains.
func FixDomains(domains []*pdl.Domain) {
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
}
