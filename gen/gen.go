// Package gen provides various template-based source code generators for the
// Chrome DevTools Protocol domain definitions.
package gen

import (
	"bytes"

	"github.com/chromedp/pdlgen/pdl"
)

// Versions are the Chromium and V8 versions of the protocol definitions code is
// generated from.
type Versions struct {
	// Chromium is the Chromium version of browser_protocol.pdl.
	Chromium string
	// V8 is the V8 version of js_protocol.pdl.
	V8 string
}

// Generator is the common interface for code generators.
type Generator func([]*pdl.Domain, string, Versions) (Emitter, error)

// Emitter is the shared interface for code emitters.
type Emitter interface {
	Emit() map[string]*bytes.Buffer
}

// Generators returns all the various Chrome DevTools Protocol generators.
func Generators() map[string]Generator {
	return map[string]Generator{
		"go": NewGoGenerator,
	}
}
