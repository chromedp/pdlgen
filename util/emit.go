package util

import (
	"flag"

	"github.com/chromedp/pdlgen/util/pdl"
	"github.com/spf13/afero"
)

// Generator provides common functionality to emitters.
type Generator interface {
	// Logf is a log func.
	Logf(string, ...interface{})
	// OutFs provides an fs where generated files should be written to.
	OutFs() afero.Fs
	// TmpFs provides an fs where temporary files should be written to.
	TmpFs() afero.Fs
}

// NewEmitterFunc is the func that builds an emitter.
type NewEmitterFunc func() (*flag.FlagSet, Emitter, error)

// Emitter is the shared interface for code emitters.
type Emitter interface {
	// Emit emits the code to the generator for the specified browser and v8
	// version. Returns true to indicate an "overlay"-ed file system, otherwise
	// the out directory will be completely removed prior to emitting the
	// generated files.
	Emit(g Generator, def *pdl.PDL) (bool, error)
}

// PostEmitter is the interface for code emitters that have a post generation
// step, code formatting or other late stage steps.
type PostEmitter interface {
	PostEmit(g Generator, def *pdl.PDL) (bool, error)
}
