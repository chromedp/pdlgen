package gen

import (
	"bytes"
	"path"
	"path/filepath"

	"github.com/chromedp/pdlgen/gen/genutil"
	"github.com/chromedp/pdlgen/gen/gotpl"
	"github.com/chromedp/pdlgen/pdl"
)

// GoGenerator generates Go source code for the Chrome DevTools Protocol.
type GoGenerator struct {
	files fileBuffers
}

// NewGoGenerator creates a Go source code generator for the Chrome DevTools
// Protocol domain definitions.
func NewGoGenerator(domains []*pdl.Domain, basePkg string, versions Versions) (Emitter, error) {
	fb := make(fileBuffers)

	// generate shared types
	if err := fb.generateSharedTypes(domains, basePkg); err != nil {
		return nil, err
	}

	// generate util package
	if err := fb.generateRootPackage(domains, basePkg); err != nil {
		return nil, err
	}

	// generate version file
	if err := fb.generateVersion(basePkg, versions); err != nil {
		return nil, err
	}

	// generate individual domains
	for _, d := range domains {
		pkgName := genutil.PackageName(d)
		pkgOut := filepath.Join(pkgName, pkgName+".go")

		// do command template
		w := fb.get(pkgOut, pkgName, d, domains, basePkg)
		if err := gotpl.Domain(w, d, domains); err != nil {
			return nil, err
		}

		// generate domain types
		if len(d.Types) != 0 {
			if err := fb.generateTypes(
				filepath.Join(pkgName, "types.go"),
				d.Types, gotpl.TypePrefix, gotpl.TypeSuffix,
				d, domains,
				basePkg,
			); err != nil {
				return nil, err
			}
		}

		// generate domain event types
		if len(d.Events) != 0 {
			if err := fb.generateTypes(
				filepath.Join(pkgName, "events.go"),
				d.Events, gotpl.EventTypePrefix, gotpl.EventTypeSuffix,
				d, domains,
				basePkg,
			); err != nil {
				return nil, err
			}
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
func (fb fileBuffers) generateSharedTypes(domains []*pdl.Domain, basePkg string) error {
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
	if err := gotpl.Errors(w); err != nil {
		return err
	}
	if err := gotpl.Session(w); err != nil {
		return err
	}

	// add types
	for _, t := range typs {
		if err := gotpl.Type(
			w, t, gotpl.TypePrefix, gotpl.TypeSuffix,
			d, append(domains, d),
			false, true,
		); err != nil {
			return err
		}
	}
	return nil
}

// generateRootPackage generates the util package.
//
// The package holds only the low-level message unmarshaler. A separate package
// prevents circular dependencies.
func (fb fileBuffers) generateRootPackage(domains []*pdl.Domain, basePkg string) error {
	n := path.Base(basePkg)
	d := &pdl.Domain{
		Domain:      pdl.DomainType(n),
		Description: "Chrome DevTools Protocol types.",
	}
	w := fb.get(n+".go", n, d, domains, basePkg)
	for _, t := range rootPackageTypes(domains) {
		if err := gotpl.Type(w, t, "", "", d, domains, false, true); err != nil {
			return err
		}
	}
	return nil
}

// generateVersion generates the version file for the root package, recording
// the versions of the protocol definitions the code was generated from.
func (fb fileBuffers) generateVersion(basePkg string, versions Versions) error {
	n := path.Base(basePkg)
	w := new(bytes.Buffer)
	fb["version.go"] = w
	if err := gotpl.FileHeader(w, n, nil); err != nil {
		return err
	}
	return gotpl.Version(w, versions.Chromium, versions.V8)
}

// generateTypes generates the types for a domain.
func (fb fileBuffers) generateTypes(
	path string,
	types []*pdl.Type, prefix, suffix string,
	d *pdl.Domain, domains []*pdl.Domain,
	basePkg string,
) error {
	w := fb.get(path, genutil.PackageName(d), d, domains, basePkg)

	// process type list
	for _, t := range types {
		if t.IsCircularDep {
			continue
		}
		if err := gotpl.Type(w, t, prefix, suffix, d, domains, false, true); err != nil {
			return err
		}
	}
	return nil
}

// get retrieves the file buffer for s, or creates it if it is not yet available.
func (fb fileBuffers) get(s string, pkgName string, d *pdl.Domain, domains []*pdl.Domain, basePkg string) *bytes.Buffer {
	// check if it already exists
	if b, ok := fb[s]; ok {
		return b
	}

	// create buffer
	w := new(bytes.Buffer)
	fb[s] = w

	v := d
	if b := path.Base(s); b != pkgName+".go" {
		v = nil
	}

	// add package header
	if err := gotpl.FileHeader(w, pkgName, v); err != nil {
		panic(err)
	}

	// add import map
	importMap := map[string]string{
		"encoding/json":               "",
		"encoding/json/jsontext":      "",
		"fmt":                         "",
		"iter":                        "",
		"encoding/json/v2":            "jsonv2",
		basePkg + "/cdp":              "",
		"github.com/chromedp/sysutil": "",
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
	if err := gotpl.FileImports(w, importMap); err != nil {
		panic(err)
	}

	return w
}

// rootPackageTypes returns the root package types.
func rootPackageTypes(domains []*pdl.Domain) []*pdl.Type {
	return []*pdl.Type{{
		Name:        "MethodType",
		Type:        pdl.TypeString,
		Description: "Chrome DevTools Protocol method type (ie, event and command names).",
		Extra:       gotpl.MethodType(domains),
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
		Extra: gotpl.Message(domains),
	}}
}
