package gen

import (
	"bytes"
	"path"
	"path/filepath"

	qtpl "github.com/valyala/quicktemplate"

	"github.com/chromedp/cdproto-gen/gen/genutil"
	"github.com/chromedp/cdproto-gen/gen/gotpl"
	"github.com/chromedp/cdproto-gen/pdl"
)

// GoGenerator generates Go source code for the Chrome DevTools Protocol.
type GoGenerator struct {
	files fileBuffers
}

// NewGoGenerator creates a Go source code generator for the Chrome DevTools
// Protocol domain definitions.
func NewGoGenerator(domains []*pdl.Domain, basePkg string) (Emitter, error) {
	var w *qtpl.Writer

	fb := make(fileBuffers)

	// generate shared types
	fb.generateSharedTypes(domains, basePkg)

	// generate util package
	fb.generateRootPackage(domains, basePkg)

	// generate individual domains
	for _, d := range domains {
		pkgName := genutil.PackageName(d)
		pkgOut := filepath.Join(pkgName, pkgName+".go")

		// do command template
		w = fb.get(pkgOut, pkgName, d, domains, basePkg)
		gotpl.StreamDomainTemplate(w, d, domains)
		fb.release(w)

		// generate domain types
		if len(d.Types) != 0 {
			fb.generateTypes(
				filepath.Join(pkgName, "types.go"),
				d.Types, gotpl.TypePrefix, gotpl.TypeSuffix,
				d, domains,
				basePkg,
			)
		}

		// generate domain event types
		if len(d.Events) != 0 {
			fb.generateTypes(
				filepath.Join(pkgName, "events.go"),
				d.Events, gotpl.EventTypePrefix, gotpl.EventTypeSuffix,
				d, domains,
				basePkg,
			)
		}
	}

	return &GoGenerator{
		files: fb,
	}, nil
}

// Emit returns the generated files.
func (gg *GoGenerator) Emit() map[string]*bytes.Buffer {
	return map[string]*bytes.Buffer(gg.files)
}

// fileBuffers is a type to manage buffers for file data.
type fileBuffers map[string]*bytes.Buffer

// generateSharedTypes generates the common shared types for domains.
//
// Because there are circular package dependencies, some types need to be moved
// to eliminate circular dependencies.
func (fb fileBuffers) generateSharedTypes(domains []*pdl.Domain, basePkg string) {
	// determine shared types
	var typs []*pdl.Type
	for _, d := range domains {
		for _, t := range d.Types {
			if t.IsCircularDep {
				typs = append(typs, t)
			}
		}
	}

	d := &pdl.Domain{
		Domain:      pdl.DomainType("cdp"),
		Types:       typs,
		Description: "Shared Chrome DevTools Protocol Domain types.",
	}

	w := fb.get("cdp/types.go", "cdp", d, domains, basePkg)

	// add executor
	gotpl.StreamExtraExecutorTemplate(w)

	// add types
	for _, t := range typs {
		gotpl.StreamTypeTemplate(
			w, t, gotpl.TypePrefix, gotpl.TypeSuffix,
			d, append(domains, d),
			nil, false, true,
		)
	}

	fb.release(w)
}

// generateRootPackage generates the util package.
//
// Currently only contains the low-level message unmarshaler -- if this wasn't
// in a separate package, then there would be circular dependencies.
func (fb fileBuffers) generateRootPackage(domains []*pdl.Domain, basePkg string) {
	n := path.Base(basePkg)
	d := &pdl.Domain{
		Domain:      pdl.DomainType(n),
		Description: "Chrome DevTools Protocol types.",
	}
	w := fb.get(n+".go", n, d, domains, basePkg)
	for _, t := range rootPackageTypes(domains) {
		gotpl.StreamTypeTemplate(
			w, t, "", "",
			d, domains,
			nil, false, true,
		)
	}
	fb.release(w)
}

// generateTypes generates the types for a domain.
func (fb fileBuffers) generateTypes(
	path string,
	types []*pdl.Type, prefix, suffix string,
	d *pdl.Domain, domains []*pdl.Domain,
	basePkg string,
) {
	w := fb.get(path, genutil.PackageName(d), d, domains, basePkg)

	// process type list
	for _, t := range types {
		if t.IsCircularDep {
			continue
		}
		gotpl.StreamTypeTemplate(
			w, t, prefix, suffix,
			d, domains,
			nil, false, true,
		)
	}

	fb.release(w)
}

// get retrieves the file buffer for s, or creates it if it is not yet available.
func (fb fileBuffers) get(s string, pkgName string, d *pdl.Domain, domains []*pdl.Domain, basePkg string) *qtpl.Writer {
	// check if it already exists
	if b, ok := fb[s]; ok {
		return qtpl.AcquireWriter(b)
	}

	// create buffer
	b := new(bytes.Buffer)
	fb[s] = b
	w := qtpl.AcquireWriter(b)

	v := d
	if b := path.Base(s); b != pkgName+".go" {
		v = nil
	}

	// add package header
	gotpl.StreamFileHeader(w, pkgName, v)

	// add import map
	importMap := map[string]string{
		"encoding/json":                      "",
		basePkg + "/cdp":                     "",
		"github.com/mailru/easyjson":         "",
		"github.com/mailru/easyjson/jlexer":  "",
		"github.com/mailru/easyjson/jwriter": "",
		"github.com/chromedp/sysutil":        "",
	}
	// add io only for cdp package
	if pkgName == "cdp" {
		importMap["io"] = ""
	}
	for _, d := range domains {
		pn := genutil.PackageName(d)
		// skip adding cdproto/io package to cdp package
		if pkgName == "cdp" && pn == "io" {
			continue
		}
		importMap[basePkg+"/"+pn] = ""
	}
	gotpl.StreamFileImportTemplate(w, importMap)

	return w
}

// release releases a template writer.
func (fb fileBuffers) release(w *qtpl.Writer) {
	qtpl.ReleaseWriter(w)
}

// rootPackageTypes returns the root package types.
func rootPackageTypes(domains []*pdl.Domain) []*pdl.Type {
	return []*pdl.Type{{
		Name:             "MethodType",
		Type:             pdl.TypeString,
		Description:      "Chrome DevTools Protocol method type (ie, event and command names).",
		EnumValueNameMap: make(map[string]string),
		Extra:            gotpl.ExtraMethodTypeTemplate(domains),
	}, {
		Name:        "Error",
		Type:        pdl.TypeObject,
		Description: "Error type.",
		Properties: []*pdl.Type{{
			Name:        "code",
			Type:        pdl.TypeInteger,
			Description: "Error code.",
		}, {
			Name:        "message",
			Type:        pdl.TypeString,
			Description: "Error message.",
		}},
		Extra: `// Error satisfies the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}`,
	}, {
		Name:        "Message",
		Type:        pdl.TypeObject,
		Description: "Chrome DevTools Protocol message sent/read over websocket connection.",
		Properties: []*pdl.Type{{
			Name:        "id",
			Type:        pdl.TypeInteger,
			Description: "Unique message identifier.",
			Optional:    true,
		}, {
			Name:        "sessionId",
			Ref:         "Target.SessionID",
			Description: "Session that the message belongs to when using flat access.",
			Optional:    true,
		}, {
			Name:        "method",
			Ref:         "MethodType",
			Description: "Event or command type.",
			Optional:    true,
			NoResolve:   true,
		}, {
			Name:        "params",
			Type:        pdl.TypeAny,
			Description: "Event or command parameters.",
			Optional:    true,
		}, {
			Name:        "result",
			Type:        pdl.TypeAny,
			Description: "Command return values.",
			Optional:    true,
		}, {
			Name:        "error",
			Ref:         "*Error",
			Description: "Error message.",
			Optional:    true,
			NoResolve:   true,
		}},
		Extra: gotpl.ExtraMessageTemplate(domains),
	}}
}
