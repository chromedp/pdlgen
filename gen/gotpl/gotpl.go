// Package gotpl contains the text/template based code generation templates
// used by cdproto-gen to generate Go code.
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

	"github.com/chromedp/cdproto-gen/gen/genutil"
	"github.com/chromedp/cdproto-gen/pdl"
)

//go:embed *.tmpl
var tmplFS embed.FS

// tmpl is the set of all templates.
var tmpl = template.Must(template.New("gotpl").Funcs(funcMap()).ParseFS(tmplFS, "*.tmpl"))

// Import is an import path with an optional alias.
type Import struct {
	Alias string
	Path  string
}

// FileHeader writes the package header for pkgName to w. The package comment
// is only written when d is not nil.
func FileHeader(w io.Writer, pkgName string, d *pdl.Domain) error {
	return execute(w, "header", struct {
		Package string
		Domain  *pdl.Domain
	}{pkgName, d})
}

// FileImports writes the import block for the imports (a map of import path to
// alias, where an alias equal to the path means no alias) to w, sorted by path.
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

// Domain writes the commands of domain d to w.
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

// Executor writes the shared executor types and funcs to w.
func Executor(w io.Writer) error {
	return execute(w, "executor", nil)
}

// Version writes the version funcs for the Chromium and V8 versions to w.
func Version(w io.Writer, chromium, v8 string) error {
	return execute(w, "version", struct{ Chromium, V8 string }{chromium, v8})
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

// render executes the named template, returning the result as a string.
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

// optionData is the data passed to the command option func template.
type optionData struct {
	T, C    *pdl.Type
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
		"option": func(t, c *pdl.Type, d *pdl.Domain, domains []*pdl.Domain) *optionData {
			return &optionData{t, c, d, domains}
		},
		"returnsType": func(c *pdl.Type) *pdl.Type {
			return &pdl.Type{
				RawType:     "returns",
				RawName:     c.RawName,
				Name:        c.Name,
				Type:        pdl.TypeObject,
				Description: "Return values.",
				Properties:  c.Returns,
			}
		},
		// names
		"packageName":            genutil.PackageName,
		"protoName":              ProtoName,
		"camelName":              CamelName,
		"goName":                 GoName,
		"eventMethodType":        EventMethodType,
		"commandMethodType":      CommandMethodType,
		"eventType":              EventType,
		"commandType":            CommandType,
		"commandReturnsType":     CommandReturnsType,
		"enumValueName":          EnumValueName,
		"commandTypePrefix":      func() string { return CommandTypePrefix },
		"commandTypeSuffix":      func() string { return CommandTypeSuffix },
		"commandReturnsPrefix":   func() string { return CommandReturnsPrefix },
		"commandReturnsSuffix":   func() string { return CommandReturnsSuffix },
		"optionFuncPrefix":       func() string { return OptionFuncPrefix },
		"optionFuncSuffix":       func() string { return OptionFuncSuffix },
		"base64EncodedParamName": func() string { return Base64EncodedParamName },
		// types
		"goType":     GoType,
		"goTypeDef":  GoTypeDef,
		"goEnumType": func(te pdl.TypeEnum) string { return GoEnumType(te) },
		"isBoolean":  func(t *pdl.Type) bool { return t.Type == pdl.TypeBoolean },
		"isNil":      func(v []*pdl.Type) bool { return v == nil },
		// comments
		"comment":    func(s, prefix string) string { return genutil.FormatComment(s, "", prefix) },
		"docRefLink": DocRefLink,
		"paramDesc":  ParamDesc,
		// command parameters and return values
		"paramList":             ParamList,
		"retTypeList":           RetTypeList,
		"emptyRetList":          EmptyRetList,
		"retNameList":           RetNameList,
		"base64EncodedRetParam": Base64EncodedRetParam,
		"hasBase64EncodedParam": HasBase64EncodedParam,
		"commandParamValue":     CommandParamValue,
		// enums
		"enumValue":    EnumValue,
		"exportedName": ExportedName,
	}
}

// ExportedName returns the exported name of a Go type name, removing any
// package qualifier ("time.Time" becomes "Time").
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

// HasBase64EncodedParam returns true when the command's return values include
// a parameter indicating whether or not the value is base64 encoded.
func HasBase64EncodedParam(c *pdl.Type) bool {
	for _, p := range c.Returns {
		if p.Name == Base64EncodedParamName {
			return true
		}
	}
	return false
}

// CommandParamValue returns the Go value to use for the parameter in a
// command's constructor func.
func CommandParamValue(t *pdl.Type) string {
	if t.Type == pdl.TypeBoolean && t.Optional {
		if defaultTrueRE.MatchString(t.Description) {
			return "true"
		}
		return "false"
	}
	return GoName(t, true)
}
