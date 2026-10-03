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
	if t.RawType != "command" {
		name := TypeName(t, prefix, suffix)
		t = withBase64(t, name)
		t = withUnmarshaler(t, name)
	}
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
func returnsType(c *pdl.Type) *pdl.Type {
	return &pdl.Type{
		RawType:     "returns",
		RawName:     c.RawName,
		Name:        c.Name,
		Type:        pdl.TypeObject,
		Description: "is the result of the command " + c.RawName + ".",
		Properties:  c.Returns,
	}
}

// unmarshalers lists the types that have a form in the protocol that the struct
// cannot decode alone. The value is the name of the template that writes the
// UnmarshalJSON method of the type. The key is the protocol name of the type.
var unmarshalers = map[string]string{
	// Older versions of the browser send the partition key as a string, and newer
	// versions send an object.
	"Network.CookiePartitionKey": "cookiepartitionkey",
}

// withUnmarshaler returns t with the UnmarshalJSON method that unmarshalers
// names for it, or t itself when it has none. typeName is the Go name of t.
func withUnmarshaler(t *pdl.Type, typeName string) *pdl.Type {
	name, ok := unmarshalers[t.RawName]
	if !ok || t.RawType != "type" {
		return t
	}
	c := *t
	if c.Extra != "" {
		c.Extra += "\n\n"
	}
	c.Extra += render(name, struct{ Type string }{typeName})
	return &c
}

// base64Rule says how a struct tells that its text value is base64.
type base64Rule struct {
	// field is the name of the text value, and sibling is the name of the
	// property that tells whether the value is base64.
	field, sibling string
	// cond returns the Go condition on the Go name of the sibling.
	cond func(sibling string) string
}

// base64Rules are the structs whose text value is base64 under a condition on
// another property, which a name alone does not give. The protocol describes
// the condition in a description or in the HAR specification.
var base64Rules = map[string]base64Rule{
	// If the opcode is 1, payloadData is a UTF-8 string. If it is not 1, then
	// payloadData is a base64 encoded string of binary data.
	"Network.WebSocketFrame": {"payloadData", "opcode", func(s string) string { return "r." + s + " != 1" }},
	// The text is encoded as the encoding says, for example "base64".
	"HAR.Content": {"text", "encoding", func(s string) string { return "r." + s + ` == "base64"` }},
}

// withBase64 returns t with the base64 text value of its struct fixed, or t
// itself when it has none.
//
// A struct can hold a text value that the browser sends as base64 when another
// property says so. Network.getResponseBody does it with body and the flag
// base64Encoded. The value of such a struct is a []byte, and the struct decodes
// it by the property in its UnmarshalJSON, as the old Do method did. typeName is
// the Go name of the struct.
func withBase64(t *pdl.Type, typeName string) *pdl.Type {
	props := t.Properties
	if t.RawType == "event" {
		props = t.Parameters
	}
	field, sibling, cond := base64Pair(t.RawName, props)
	if field == nil {
		return t
	}
	c := *t
	fixed := make([]*pdl.Type, len(props))
	for i, p := range props {
		if p == field {
			bin := *p
			bin.Type = pdl.TypeBinary
			p = &bin
		}
		fixed[i] = p
	}
	if t.RawType == "event" {
		c.Parameters = fixed
	} else {
		c.Properties = fixed
	}
	c.Extra = t.Extra + render("base64result", struct{ Type, Field, FieldJSON, Cond string }{
		Type:      typeName,
		Field:     GoName(field, false),
		FieldJSON: field.Name,
		Cond:      cond(GoName(sibling, false)),
	})
	return &c
}

// base64Pair returns the text value of the properties that can be base64, the
// property that tells it, and the Go condition that says that the value is
// base64.
//
// The value is next to a boolean base64Encoded, before the flag or after it, or
// it is the value of a rule in base64Rules for the struct rawName.
func base64Pair(rawName string, props []*pdl.Type) (field, sibling *pdl.Type, cond func(string) string) {
	find := func(name string) *pdl.Type {
		for _, p := range props {
			if p.Name == name {
				return p
			}
		}
		return nil
	}
	if r, ok := base64Rules[rawName]; ok {
		if field, sibling = find(r.field), find(r.sibling); field != nil && sibling != nil {
			return field, sibling, r.cond
		}
	}
	for i, p := range props {
		if p.Name != "base64Encoded" || p.Type != pdl.TypeBoolean {
			continue
		}
		for _, j := range []int{i - 1, i + 1} {
			if j >= 0 && j < len(props) && props[j].Type == pdl.TypeString && props[j].Ref == "" && props[j].Enum == nil {
				return props[j], p, func(s string) string { return "r." + s }
			}
		}
	}
	return nil, nil, nil
}
