// Package gen takes the Chrome protocol domain definitions and applies the
// necessary code generation templates.
package gen

import (
	"bytes"
	"path"

	"github.com/gedex/inflector"
	"github.com/knq/snaker"
	qtpl "github.com/valyala/quicktemplate"

	"github.com/chromedp/chromedp-gen/templates"
	"github.com/chromedp/chromedp-gen/types"
)

// GenerateDomains generates domains for the Chrome Debugging Protocol domain
// definitions, returning a set of file buffers as a map of the file name ->
// content.
func GenerateDomains(domains []*types.Domain, sharedFunc func(string, string) bool, basePkg string) map[string]*bytes.Buffer {
	fb := make(fileBuffers)

	var w *qtpl.Writer

	// determine base (also used for the domains manager type name)
	pkgBase := path.Base(basePkg)
	types.DomainTypeSuffix = inflector.Singularize(snaker.ForceCamelIdentifier(pkgBase))

	// generate internal types
	fb.generateCDPTypes(domains, sharedFunc, basePkg)

	// generate util package
	fb.generateUtilPackage(domains, basePkg)

	// do individual domain templates
	for _, d := range domains {
		pkgName := d.PackageName()
		pkgOut := pkgName + "/" + pkgName + ".go"

		// do command template
		w = fb.get(pkgOut, pkgName, d)
		templates.StreamDomainTemplate(w, d, domains, sharedFunc, basePkg)
		fb.release(w)

		// generate domain types
		if len(d.Types) != 0 {
			fb.generateTypes(
				pkgName+"/types.go",
				d.Types, types.TypePrefix, types.TypeSuffix,
				d, domains, sharedFunc,
				"", "", "", "", "",
				basePkg,
			)
		}

		// generate domain event types
		if len(d.Events) != 0 {
			fb.generateTypes(
				pkgName+"/events.go",
				d.Events, types.EventTypePrefix, types.EventTypeSuffix,
				d, domains, sharedFunc,
				"EventTypes", "cdp.MethodType", "cdp."+types.EventMethodPrefix+d.String(), types.EventMethodSuffix,
				"All event types in the domain.",
				basePkg,
			)
		}
	}

	return map[string]*bytes.Buffer(fb)
}

// fileBuffers is a type to manage buffers for file data.
type fileBuffers map[string]*bytes.Buffer

// generateCDPTypes generates the common shared types for domain d.
//
// Because there are circular package dependencies, some types need to be moved
// to eliminate the circular dependencies. Please see the fixup package for a
// list of the "shared" CDP types.
func (fb fileBuffers) generateCDPTypes(domains []*types.Domain, sharedFunc func(string, string) bool, basePkg string) {
	var typs []*types.Type
	for _, d := range domains {
		// process internal types
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

	n := path.Base(basePkg)
	w := fb.get(n+".go", n, nil)
	for _, t := range typs {
		templates.StreamTypeTemplate(
			w, t, types.TypePrefix, types.TypeSuffix,
			cdpDomain, doms, sharedFunc,
			nil, false, true,
		)
	}
	fb.release(w)
}

// generateUtilPackage generates the util package.
//
// Currently only contains the low-level message unmarshaler -- if this wasn't
// in a separate package, then there would be circular dependencies.
func (fb fileBuffers) generateUtilPackage(domains []*types.Domain, basePkg string) {
	// generate import map data
	importMap := map[string]string{
		basePkg: "cdp",
	}
	for _, d := range domains {
		importMap[basePkg+"/"+d.PackageName()] = d.PackageImportAlias()
	}

	w := fb.get("cdputil/cdputil.go", "cdputil", nil)
	templates.StreamFileImportTemplate(w, importMap)
	templates.StreamExtraUtilTemplate(w, domains)
	fb.release(w)
}

// generateTypes generates the types for a domain.
func (fb fileBuffers) generateTypes(
	path string,
	types []*types.Type, prefix, suffix string,
	d *types.Domain, domains []*types.Domain, sharedFunc func(string, string) bool,
	emit, emitType, emitPrefix, emitSuffix, emitDesc string,
	basePkg string,
) {
	w := fb.get(path, d.PackageName(), d)

	// add internal import
	templates.StreamFileImportTemplate(w, map[string]string{basePkg: "cdp"})

	// process type list
	var names []string
	for _, t := range types {
		if sharedFunc(d.Domain.String(), t.IDorName()) {
			continue
		}
		templates.StreamTypeTemplate(
			w, t, prefix, suffix,
			d, domains, sharedFunc,
			nil, false, true,
		)
		names = append(names, t.TypeName(emitPrefix, emitSuffix))
	}

	// emit var
	if emit != "" {
		s := "[]" + emitType + "{"
		for _, n := range names {
			s += "\n" + n + ","
		}
		s += "\n}"
		templates.StreamFileVarTemplate(w, emit, s, emitDesc)
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
