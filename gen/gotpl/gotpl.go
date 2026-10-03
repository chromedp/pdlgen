// Package gotpl contains the text/template based code generation templates
// used by pdlgen to generate Go code.
package gotpl

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/chromedp/pdlgen/gen/genutil"
	"github.com/chromedp/pdlgen/pdl"
)

//go:embed *.tmpl
var tmplFS embed.FS

// tmpl is the set of all templates.
var tmpl *template.Template

// init parses the templates. It is not a variable initializer, because a func in
// the func map renders a template, which a variable initializer cannot do.
func init() {
	tmpl = template.Must(template.New("gotpl").Funcs(funcMap()).ParseFS(tmplFS, "*.tmpl"))
}

// Import is an import path with an optional alias.
type Import struct {
	Alias string
	Path  string
}

// FileHeader writes the package header for pkgName to w. It writes the package
// comment only when d is not nil.
func FileHeader(w io.Writer, pkgName string, d *pdl.Domain) error {
	return execute(w, "header", struct {
		Package string
		Domain  *pdl.Domain
	}{pkgName, d})
}

// FileImports writes the import block for the imports to w, sorted by path. The
// map holds the alias of each import path. An alias equal to the path means no
// alias.
func FileImports(w io.Writer, imports map[string]string) error {
	var v []Import
	for path, alias := range imports {
		if alias == path {
			alias = ""
		}
		v = append(v, Import{alias, path})
	}
	slices.SortFunc(v, func(a, b Import) int { return strings.Compare(a.Path, b.Path) })
	return execute(w, "imports", v)
}

// Domain writes the commands and the events of domain d to w.
func Domain(w io.Writer, d *pdl.Domain, domains []*pdl.Domain) error {
	return execute(w, "domain", struct {
		Domain  *pdl.Domain
		Domains []*pdl.Domain
	}{d, domains})
}

// Type writes the type t to w.
func Type(w io.Writer, t *pdl.Type, prefix, suffix string, d *pdl.Domain, domains []*pdl.Domain, noExposeOverride, omitOnlyWhenOptional bool) error {
	return execute(w, "type", newTypeData(t, prefix, suffix, d, domains, noExposeOverride, omitOnlyWhenOptional))
}

// Errors writes the shared error types to w.
func Errors(w io.Writer) error {
	return execute(w, "errors", nil)
}

// Version writes the version funcs for the Chromium and V8 versions to w.
func Version(w io.Writer, chromium, v8 string) error {
	return execute(w, "version", struct{ Chromium, V8 string }{chromium, v8})
}

// Session writes the types and funcs of the typed API to w: Command, Event,
// Empty, Session, Call and Events.
func Session(w io.Writer) error {
	return execute(w, "session", nil)
}

// MethodType returns the additional MethodType funcs and consts.
func MethodType(domains []*pdl.Domain) string {
	return render("methodtype", domains)
}

// Message returns the additional Message funcs.
func Message(domains []*pdl.Domain) string {
	return render("message", domains)
}

// execute executes the named template.
func execute(w io.Writer, name string, data any) error {
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		return fmt.Errorf("template %s: %w", name, err)
	}
	return nil
}

// render executes the named template and returns the result as a string.
func render(name string, data any) string {
	var buf bytes.Buffer
	if err := execute(&buf, name, data); err != nil {
		panic(err)
	}
	return buf.String()
}

// typeData is the data passed to the type template.
type typeData struct {
	T                    *pdl.Type
	Prefix, Suffix       string
	D                    *pdl.Domain
	Domains              []*pdl.Domain
	NoExposeOverride     bool
	OmitOnlyWhenOptional bool
}

func newTypeData(t *pdl.Type, prefix, suffix string, d *pdl.Domain, domains []*pdl.Domain, noExposeOverride, omitOnlyWhenOptional bool) *typeData {
	return &typeData{t, prefix, suffix, d, domains, noExposeOverride, omitOnlyWhenOptional}
}

// commandData is the data passed to the command templates.
type commandData struct {
	C       *pdl.Type
	D       *pdl.Domain
	Domains []*pdl.Domain
}

// funcMap returns the template func map.
func funcMap() template.FuncMap {
	return template.FuncMap{
		// data
		"typeData": newTypeData,
		"command": func(c *pdl.Type, d *pdl.Domain, domains []*pdl.Domain) *commandData {
			return &commandData{c, d, domains}
		},
		"paramsType": func(c *pdl.Type) *pdl.Type {
			p := *c
			p.Description = "are the parameters of the command " + c.RawName + "."
			return &p
		},
		"returnsType": returnsType,
		// names
		"packageName":          genutil.PackageName,
		"protoName":            ProtoName,
		"camelName":            CamelName,
		"goName":               GoName,
		"eventMethodType":      EventMethodType,
		"commandMethodType":    CommandMethodType,
		"eventType":            EventType,
		"commandType":          CommandType,
		"commandReturnsType":   CommandReturnsType,
		"enumValueName":        EnumValueName,
		"commandTypePrefix":    func() string { return CommandTypePrefix },
		"commandTypeSuffix":    func() string { return CommandTypeSuffix },
		"commandReturnsPrefix": func() string { return CommandReturnsPrefix },
		"commandReturnsSuffix": func() string { return CommandReturnsSuffix },
		// types
		"goType":     GoType,
		"goTypeDef":  GoTypeDef,
		"goEnumType": func(te pdl.TypeEnum) string { return GoEnumType(te) },
		"isNil":      func(v []*pdl.Type) bool { return v == nil },
		// comments
		"comment":    func(s, prefix string) string { return genutil.FormatComment(s, "", prefix) },
		"docRefLink": DocRefLink,
		// enums
		"enumValue":    EnumValue,
		"exportedName": ExportedName,
	}
}

// ExportedName returns the exported name of a Go type name. It removes any
// package qualifier, so "time.Time" becomes "Time".
func ExportedName(s string) string {
	if _, after, ok := strings.Cut(s, "."); ok {
		s = after
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// EnumValue returns the Go value of the i'th enum value e of the type.
func EnumValue(t *pdl.Type, i int, e string) string {
	if t.Type == pdl.TypeInteger {
		return strconv.Itoa(i + 1)
	}
	return `"` + e + `"`
}

// returnsType returns the type of the result of the command c.
//
// A result can hold a text value that the browser sends as base64 when a flag
// next to it says so, as Network.getResponseBody does with body and
// base64Encoded. The value of such a result is a []byte, and the result decodes
// it by the flag, as the old Do method did.
func returnsType(c *pdl.Type) *pdl.Type {
	t := &pdl.Type{
		RawType:     "returns",
		RawName:     c.RawName,
		Name:        c.Name,
		Type:        pdl.TypeObject,
		Description: "is the result of the command " + c.RawName + ".",
		Properties:  c.Returns,
	}
	field, flag := base64Pair(c.Returns)
	if field == nil {
		return t
	}
	t.Properties = make([]*pdl.Type, len(c.Returns))
	for i, p := range c.Returns {
		if p == field {
			bin := *p
			bin.Type = pdl.TypeBinary
			p = &bin
		}
		t.Properties[i] = p
	}
	t.Extra = render("base64result", struct{ Type, Field, FieldJSON, Flag string }{
		Type:      CommandReturnsType(c),
		Field:     GoName(field, false),
		FieldJSON: field.Name,
		Flag:      GoName(flag, false),
	})
	return t
}

// base64Pair returns the text value and the flag that says it is base64, when
// the properties have a boolean base64Encoded and a string value next to it.
// The value is the property before the flag, or the one after it.
func base64Pair(props []*pdl.Type) (field, flag *pdl.Type) {
	for i, p := range props {
		if p.Name != "base64Encoded" || p.Type != pdl.TypeBoolean {
			continue
		}
		for _, j := range []int{i - 1, i + 1} {
			if j >= 0 && j < len(props) && props[j].Type == pdl.TypeString && props[j].Ref == "" && props[j].Enum == nil {
				return props[j], p
			}
		}
	}
	return nil, nil
}
