// Package gen provides various template-based source code generators for the
// Chrome Debugging Protocol domain definitions.
package gen

import (
	"bytes"

	"github.com/chromedp/cdproto-gen/pdl"
)

// Generator is the common interface for code generators.
type Generator func([]*pdl.Domain, string) (Emitter, error)

// Emitter is the shared interface for code emitters.
type Emitter interface {
	Emit() map[string]*bytes.Buffer
}

// Generators returns all the various Chrome Debugging Protocol generators.
func Generators() map[string]Generator {
	return map[string]Generator{
		"go": NewGoGenerator,
	}
}
