// Package gen takes the Chrome protocol domain definitions and applies the
// necessary code generation templates.
package gen

import (
	"bytes"
	"path"
	"path/filepath"

	"github.com/knq/snaker"
	qtpl "github.com/valyala/quicktemplate"

	"github.com/chromedp/chromedp-gen/templates"
	"github.com/chromedp/chromedp-gen/types"
)

// GenerateDomains generates domains for the Chrome Debugging Protocol domain
// definitions, returning a set of file buffers as a map of the file name ->
// content.
func GenerateDomains(domains []*types.Domain, basePkg string, redirect bool) map[string]*bytes.Buffer {
	// setup shared types
	sharedTypes := map[string]bool{
		"DOM.BackendNodeId":      true,
		"DOM.BackendNode":        true,
		"DOM.NodeId":             true,
		"DOM.Node":               true,
		"DOM.NodeType":           true,
		"DOM.PseudoType":         true,
		"DOM.RGBA":               true,
		"DOM.ShadowRootType":     true,
		"Network.LoaderId":       true,
		"Network.MonotonicTime":  true,
		"Network.TimeSinceEpoch": true,
		"Page.FrameId":           true,
		"Page.Frame":             true,
		"Network.Cookie":         redirect,
		"Network.CookieSameSite": redirect,
		"Page.ResourceType":      redirect,
	}
	sharedFunc := func(dtyp string, typ string) bool {
		return sharedTypes[dtyp+"."+typ]
	}

	fb := make(fileBuffers)

	var w *qtpl.Writer

	// generate shared types
	fb.generateSharedTypes(domains, sharedFunc, basePkg)

	// generate util package
	fb.generateRootPackage(domains, basePkg)

	// generate individual domains
	for _, d := range domains {
		pkgName := d.PackageName()
		pkgOut := filepath.Join(pkgName, pkgName+".go")

		// do command template
		w = fb.get(pkgOut, pkgName, d)
		templates.StreamDomainTemplate(w, d, domains, sharedFunc, map[string]string{
			basePkg + "/cdp": "cdp",
		})
		fb.release(w)

		// generate domain types
		if len(d.Types) != 0 {
			fb.generateTypes(
				filepath.Join(pkgName, "types.go"),
				d.Types, types.TypePrefix, types.TypeSuffix,
				d, domains, sharedFunc,
				basePkg,
			)
		}

		// generate domain event types
		if len(d.Events) != 0 {
			fb.generateTypes(
				filepath.Join(pkgName, "events.go"),
				d.Events, types.EventTypePrefix, types.EventTypeSuffix,
				d, domains, sharedFunc,
				basePkg,
			)
		}
	}

	return map[string]*bytes.Buffer(fb)
}

// fileBuffers is a type to manage buffers for file data.
type fileBuffers map[string]*bytes.Buffer

// generateSharedTypes generates the common shared types for domains.
//
// Because there are circular package dependencies, some types need to be moved
// to eliminate circular dependencies.
func (fb fileBuffers) generateSharedTypes(domains []*types.Domain, sharedFunc func(string, string) bool, basePkg string) {
	// determine shared types
	var typs []*types.Type
	for _, d := range domains {
		for _, t := range d.Types {
			if sharedFunc(d.Domain.String(), t.IDorName()) {
				typs = append(typs, t)
			}
		}
	}

	cdpDomain := &types.Domain{
		Domain: types.DomainType("cdp"),
		Types:  typs,
	}
	doms := append(domains, cdpDomain)

	w := fb.get("cdp/types.go", "cdp", &types.Domain{
		Domain:      types.DomainType("CDP"),
		Description: "Shared Chrome Debugging Protocol Domain types.",
	})
	for _, t := range typs {
		templates.StreamTypeTemplate(
			w, t, types.TypePrefix, types.TypeSuffix,
			cdpDomain, doms, sharedFunc,
			nil, false, true,
		)
	}
	fb.release(w)
}

// generateRootPackage generates the util package.
//
// Currently only contains the low-level message unmarshaler -- if this wasn't
// in a separate package, then there would be circular dependencies.
func (fb fileBuffers) generateRootPackage(domains []*types.Domain, basePkg string) {
	// generate import map data
	importMap := map[string]string{
		basePkg + "/cdp": "cdp",
	}
	for _, d := range domains {
		importMap[basePkg+"/"+d.PackageName()] = d.PackageImportAlias()
	}

	n := path.Base(basePkg)

	d := &types.Domain{
		Domain:      types.DomainType(n),
		Description: "Chrome Debugging Protocol types.",
	}
	w := fb.get(n+".go", n, d)
	templates.StreamFileImportTemplate(w, importMap)

	typs := rootPackageTypes(domains)
	for _, t := range typs {
		templates.StreamTypeTemplate(
			w, t, "", "",
			d, domains, func(string, string) bool { return false },
			nil, false, true,
		)
	}
	fb.release(w)
}

// generateTypes generates the types for a domain.
func (fb fileBuffers) generateTypes(
	path string,
	types []*types.Type, prefix, suffix string,
	d *types.Domain, domains []*types.Domain, sharedFunc func(string, string) bool,
	basePkg string,
) {
	w := fb.get(path, d.PackageName(), d)

	// add internal import
	templates.StreamFileImportTemplate(w, map[string]string{
		basePkg + "/cdp": "cdp",
	})

	// process type list
	for _, t := range types {
		if sharedFunc(d.Domain.String(), t.IDorName()) {
			continue
		}
		templates.StreamTypeTemplate(
			w, t, prefix, suffix,
			d, domains, sharedFunc,
			nil, false, true,
		)
	}

	fb.release(w)
}

// get retrieves the file buffer for s, or creates it if it is not yet available.
func (fb fileBuffers) get(s string, pkgName string, d *types.Domain) *qtpl.Writer {
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
	templates.StreamFileHeader(w, pkgName, v)

	return w
}

// release releases a template writer.
func (fb fileBuffers) release(w *qtpl.Writer) {
	qtpl.ReleaseWriter(w)
}

// rootPackageTypes returns the root package types.
func rootPackageTypes(domains []*types.Domain) []*types.Type {
	errorValues := []string{"channel closed", "invalid result", "unknown result"}
	errorValueNameMap := make(map[string]string)
	for _, e := range errorValues {
		errorValueNameMap[e] = "Err" + snaker.ForceCamelIdentifier(e)
	}

	return []*types.Type{{
		ID:               "MethodType",
		Type:             types.TypeString,
		Description:      "Chrome Debugging Protocol method type (ie, event and command names).",
		EnumValueNameMap: make(map[string]string),
		Extra:            templates.ExtraMethodTypeTemplate(domains),
	}, {
		ID:          "Error",
		Type:        types.TypeObject,
		Description: "Error type.",
		Properties: []*types.Type{{
			Name:        "code",
			Type:        types.TypeInteger,
			Description: "Error code.",
		}, {
			Name:        "message",
			Type:        types.TypeString,
			Description: "Error message.",
		}},
		Extra: `// Error satisfies error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}`,
	}, {
		ID:          "Message",
		Type:        types.TypeObject,
		Description: "Chrome Debugging Protocol message sent to/read over websocket connection.",
		Properties: []*types.Type{{
			Name:        "id",
			Type:        types.TypeInteger,
			Description: "Unique message identifier.",
			Optional:    true,
		}, {
			Name:        "method",
			Ref:         "MethodType",
			Description: "Event or command type.",
			Optional:    true,
			NoResolve:   true,
		}, {
			Name:        "params",
			Type:        types.TypeAny,
			Description: "Event or command parameters.",
			Optional:    true,
		}, {
			Name:        "result",
			Type:        types.TypeAny,
			Description: "Command return values.",
			Optional:    true,
		}, {
			Name:        "error",
			Ref:         "*Error",
			Description: "Error message.",
			Optional:    true,
			NoResolve:   true,
		}},
		Extra: templates.ExtraMessageTemplate(domains),
	}}
}
