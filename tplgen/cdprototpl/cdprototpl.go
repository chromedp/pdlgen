// Package cdprototpl contains the templates used to generate chromedp's
// protocol layer, cdproto.
//
// See:
//   github.com/chromedp/chromedp
//   github.com/chromedp/cdproto
package cdprototpl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/chromedp/pdlgen/util"
	"github.com/chromedp/pdlgen/util/pdl"
)

// Generator handles generating the source code for the cdproto package.
type Generator struct {
	Out          string                       `help:"out" default:"${GOPATH}/src/github.com/chromedp/cdproto"`
	Experimental bool                         `help:"enable experimental" default:"true"`
	Deprecated   bool                         `help:"enable deprecated"`
	Logf         func(string, ...interface{}) `kong:"-"`
}

// ReadVersion reads the version.
func (g Generator) ReadVersion() (string, string, error) {
	verpath := filepath.Join(g.Out, "version.json")
	fi, err := os.Stat(verpath)
	switch {
	case err != nil && os.IsNotExist(err):
		return "", "", nil
	case err != nil:
		return "", "", err
	case fi.IsDir():
		return "", "", fmt.Errorf("%s is a directory", verpath)
	}
	buf, err := ioutil.ReadFile(verpath)
	if err != nil {
		return "", "", err
	}
	var v struct {
		Browser string `json:"browser"`
		V8      string `json:"v8"`
	}
	if err := json.Unmarshal(buf, &v); err != nil {
		return "", "", err
	}
	return v.Browser, v.V8, nil
}

// Emit emits the cdproto templates for the provided domains to the passed file
// system.
func (g *Generator) Emit(gen util.Generator, def *pdl.PDL) error {
	domains := g.fixup(gen, def)
	domains = domains

	//	var w *qtpl.Writer
	//
	//	domains := def.Domains
	//
	//	fb := make(fileBuffers)
	//
	//	// generate shared types
	//	fb.generateSharedTypes(domains, g.Out)
	//
	//	// generate util package
	//	fb.generateRootPackage(domains, g.Out)
	//
	//	// generate individual domains
	//	for _, d := range domains {
	//		pkgName := util.PackageName(d)
	//		pkgOut := filepath.Join(pkgName, pkgName+".go")
	//
	//		// do command template
	//		w = fb.get(pkgOut, pkgName, d, domains, g.Out)
	//		StreamDomainTemplate(w, d, domains)
	//		fb.release(w)
	//
	//		// generate domain types
	//		if len(d.Types) != 0 {
	//			fb.generateTypes(
	//				filepath.Join(pkgName, "types.go"),
	//				d.Types, TypePrefix, TypeSuffix,
	//				d, domains,
	//				g.Out,
	//			)
	//		}
	//
	//		// generate domain event types
	//		if len(d.Events) != 0 {
	//			fb.generateTypes(
	//				filepath.Join(pkgName, "events.go"),
	//				d.Events, EventTypePrefix, EventTypeSuffix,
	//				d, domains,
	//				g.Out,
	//			)
	//		}
	//	}
	//
	//	return &Generator{
	//		files: fb,
	//	}, nil
	return nil
}

//// fileBuffers is a type to manage buffers for file data.
// type fileBuffers map[string]*bytes.Buffer
//
//// generateSharedTypes generates the common shared types for domains.
////
//// Because there are circular package dependencies, some types need to be moved
//// to eliminate circular dependencies.
// func (fb fileBuffers) generateSharedTypes(domains []*pdl.Domain) {
//	// determine shared types
//	var typs []*pdl.Type
//	for _, d := range domains {
//		for _, t := range d.Types {
//			if t.IsCircularDep {
//				typs = append(typs, t)
//			}
//		}
//	}
//
//	d := &pdl.Domain{
//		Domain:      pdl.DomainType("cdp"),
//		Types:       typs,
//		Description: "Shared Chrome DevTools Protocol Domain types.",
//	}
//
//	w := fb.get("cdp/types.go", "cdp", d, domains, g.Out)
//
//	// add executor
//	StreamExtraExecutorTemplate(w)
//
//	// add types
//	for _, t := range typs {
//		StreamTypeTemplate(
//			w, t, TypePrefix, TypeSuffix,
//			d, append(domains, d),
//			nil, false, true,
//		)
//	}
//
//	fb.release(w)
//}
//
//// generateRootPackage generates the util package.
////
//// Currently only contains the low-level message unmarshaler -- if this wasn't
//// in a separate package, then there would be circular dependencies.
// func (fb fileBuffers) generateRootPackage(domains []*pdl.Domain) {
//	n := path.Base(g.Out)
//	d := &pdl.Domain{
//		Domain:      pdl.DomainType(n),
//		Description: "Chrome DevTools Protocol types.",
//	}
//	w := fb.get(n+".go", n, d, domains, g.Out)
//	for _, t := range rootPackageTypes(domains) {
//		StreamTypeTemplate(
//			w, t, "", "",
//			d, domains,
//			nil, false, true,
//		)
//	}
//	fb.release(w)
//}
//
//// generateTypes generates the types for a domain.
// func (fb fileBuffers) generateTypes(
//	path string,
//	types []*pdl.Type, prefix, suffix string,
//	d *pdl.Domain, domains []*pdl.Domain,
//) {
//	w := fb.get(path, util.PackageName(d), d, domains, g.Out)
//
//	// process type list
//	for _, t := range types {
//		if t.IsCircularDep {
//			continue
//		}
//		StreamTypeTemplate(
//			w, t, prefix, suffix,
//			d, domains,
//			nil, false, true,
//		)
//	}
//
//	fb.release(w)
//}
//
//// get retrieves the file buffer for s, or creates it if it is not yet available.
// func (fb fileBuffers) get(s string, pkgName string, d *pdl.Domain, domains []*pdl.Domain) *qtpl.Writer {
//	// check if it already exists
//	if b, ok := fb[s]; ok {
//		return qtpl.AcquireWriter(b)
//	}
//
//	// create buffer
//	b := new(bytes.Buffer)
//	fb[s] = b
//	w := qtpl.AcquireWriter(b)
//
//	v := d
//	if b := path.Base(s); b != pkgName+".go" {
//		v = nil
//	}
//
//	// add package header
//	StreamFileHeader(w, pkgName, v)
//
//	// add import map
//	importMap := map[string]string{
//		"encoding/json":                      "",
//		g.Out + "/cdp":                       "",
//		"github.com/mailru/easyjson":         "",
//		"github.com/mailru/easyjson/jlexer":  "",
//		"github.com/mailru/easyjson/jwriter": "",
//	}
//	for _, d := range domains {
//		importMap[g.Out+"/"+util.PackageName(d)] = ""
//	}
//	StreamFileImportTemplate(w, importMap)
//
//	return w
//}
//
//// release releases a template writer.
// func (fb fileBuffers) release(w *qtpl.Writer) {
//	qtpl.ReleaseWriter(w)
//}
//
//// rootPackageTypes returns the root package types.
// func rootPackageTypes(domains []*pdl.Domain) []*pdl.Type {
//	return []*pdl.Type{{
//		Name:             "MethodType",
//		Type:             pdl.TypeString,
//		Description:      "Chrome DevTools Protocol method type (ie, event and command names).",
//		EnumValueNameMap: make(map[string]string),
//		Extra:            ExtraMethodTypeTemplate(domains),
//	}, {
//		Name:        "Error",
//		Type:        pdl.TypeObject,
//		Description: "Error type.",
//		Properties: []*pdl.Type{{
//			Name:        "code",
//			Type:        pdl.TypeInteger,
//			Description: "Error code.",
//		}, {
//			Name:        "message",
//			Type:        pdl.TypeString,
//			Description: "Error message.",
//		}},
//		Extra: `// Error satisfies the error interface.
// func (e *Error) Error() string {
//	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
//}`,
//	}, {
//		Name:        "Message",
//		Type:        pdl.TypeObject,
//		Description: "Chrome DevTools Protocol message sent/read over websocket connection.",
//		Properties: []*pdl.Type{{
//			Name:        "id",
//			Type:        pdl.TypeInteger,
//			Description: "Unique message identifier.",
//			Optional:    true,
//		}, {
//			Name:        "sessionId",
//			Ref:         "Target.SessionID",
//			Description: "Session that the message belongs to when using flat access.",
//			Optional:    true,
//		}, {
//			Name:        "method",
//			Ref:         "MethodType",
//			Description: "Event or command type.",
//			Optional:    true,
//			NoResolve:   true,
//		}, {
//			Name:        "params",
//			Type:        pdl.TypeAny,
//			Description: "Event or command parameters.",
//			Optional:    true,
//		}, {
//			Name:        "result",
//			Type:        pdl.TypeAny,
//			Description: "Command return values.",
//			Optional:    true,
//		}, {
//			Name:        "error",
//			Ref:         "*Error",
//			Description: "Error message.",
//			Optional:    true,
//			NoResolve:   true,
//		}},
//		Extra: ExtraMessageTemplate(domains),
//	}}
//}
