package gotpl

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/chromedp/cdproto-gen/gen/genutil"
	"github.com/chromedp/cdproto-gen/pdl"
	"github.com/knq/snaker"
)

// Prefix and suffix values.
const (
	TypePrefix           = ""
	TypeSuffix           = ""
	EventMethodPrefix    = "Event"
	EventMethodSuffix    = ""
	CommandMethodPrefix  = "Command"
	CommandMethodSuffix  = ""
	EventTypePrefix      = "Event"
	EventTypeSuffix      = ""
	CommandTypePrefix    = ""
	CommandTypeSuffix    = "Params"
	CommandReturnsPrefix = ""
	CommandReturnsSuffix = "Returns"
	OptionFuncPrefix     = "With"
	OptionFuncSuffix     = ""

	// Base64EncodedParamName is the base64encoded variable name in command
	// return values when they are optionally base64 encoded.
	Base64EncodedParamName = "base64Encoded"

	// Base64EncodedDescriptionPrefix is the prefix for command return
	// description prefix when base64 encoded.
	Base64EncodedDescriptionPrefix = "Base64-encoded"

	// ChromeDevToolsDocBase is the base URL for the Chrome DevTools
	// documentation site.
	//
	// tot is "tip-of-tree"
	ChromeDevToolsDocBase = "https://chromedevtools.github.io/devtools-protocol/tot"
)

// ProtoName returns the protocol name of the type.
func ProtoName(t *pdl.Type, d *pdl.Domain) string {
	var prefix string
	if d != nil {
		prefix = d.Domain.String() + "."
	}
	return prefix + t.Name
}

// CamelName returns the CamelCase name of the type.
func CamelName(t *pdl.Type) string {
	return snaker.ForceCamelIdentifier(t.Name)
}

// EventMethodType returns the method type of the event.
func EventMethodType(t *pdl.Type, d *pdl.Domain) string {
	return EventMethodPrefix + snaker.ForceCamelIdentifier(ProtoName(t, d)) + EventMethodSuffix
}

// CommandMethodType returns the method type of the event.
func CommandMethodType(t *pdl.Type, d *pdl.Domain) string {
	return CommandMethodPrefix + snaker.ForceCamelIdentifier(ProtoName(t, d)) + CommandMethodSuffix
}

// TypeName returns the type name using the supplied prefix and suffix.
func TypeName(t *pdl.Type, prefix, suffix string) string {
	return prefix + CamelName(t) + suffix
}

// EventType returns the type of the event.
func EventType(t *pdl.Type) string {
	return TypeName(t, EventTypePrefix, EventTypeSuffix)
}

// CommandType returns the type of the command.
func CommandType(t *pdl.Type) string {
	return TypeName(t, CommandTypePrefix, CommandTypeSuffix)
}

// CommandReturnsType returns the type of the command return type.
func CommandReturnsType(t *pdl.Type) string {
	return TypeName(t, CommandReturnsPrefix, CommandReturnsSuffix)
}

// ParamDesc returns a parameter description.
func ParamDesc(t *pdl.Type) string {
	desc := t.Description
	if desc != "" {
		desc = " - " + genutil.CleanDesc(desc)
	}

	return snaker.ForceLowerCamelIdentifier(t.Name) + desc
}

// ParamList returns the list of parameters.
func ParamList(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool, all bool) string {
	var s string
	for _, p := range t.Parameters {
		if !all && p.Optional {
			continue
		}

		_, _, z := ResolveType(p, d, domains, sharedFunc)
		s += GoName(p, true) + " " + z + ","
	}

	return strings.TrimSuffix(s, ",")
}

// Resolve is a utility func to resolve the fully qualified name of a type's
// ref from the list of provided domains, relative to domain d when ref is not
// namespaced.
func Resolve(ref string, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool) (pdl.DomainType, *pdl.Type, string) {
	n := strings.SplitN(ref, ".", 2)

	// determine domain
	dtyp, typ := d.Domain, n[0]
	if len(n) == 2 {
		dtyp, typ = pdl.DomainType(n[0]), n[1]
	}

	// determine if ref points to an object
	var other *pdl.Type
	for _, z := range domains {
		if dtyp == z.Domain {
			for _, j := range z.Types {
				if j.Name == typ {
					other = j
					break
				}
			}
			break
		}
	}

	if other == nil {
		panic(fmt.Sprintf("could not resolve type %s in domain %s", ref, d.Domain))
	}

	var s string
	// add prefix if not an internal type and not defined in the domain
	if sharedFunc(dtyp.String(), typ) {
		if d.Domain != pdl.DomainType("cdp") {
			s += "cdp."
		}
	} else if dtyp != d.Domain {
		s += strings.ToLower(dtyp.String()) + "."
	}

	return dtyp, other, s + snaker.ForceCamelIdentifier(typ)
}

// ResolveType resolves the type relative to the Go domain.
//
// Returns the DomainType of the underlying type, the underlying type (or the
// original passed type if not a reference) and the fully qualified name type
// name.
func ResolveType(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool) (pdl.DomainType, *pdl.Type, string) {
	switch {
	case t.NoExpose || t.NoResolve || strings.HasPrefix(t.Ref, "*"):
		return d.Domain, t, t.Ref

	case t.Ref != "":
		dtyp, typ, z := Resolve(t.Ref, d, domains, sharedFunc)

		// add ptr if object
		var ptr string
		switch typ.Type {
		case pdl.TypeObject, pdl.TypeTimestamp:
			ptr = "*"
		}

		return dtyp, typ, ptr + z

	case t.Type == pdl.TypeArray:
		dtyp, typ, z := ResolveType(t.Items, d, domains, sharedFunc)
		return dtyp, typ, "[]" + z

	case t.Type == pdl.TypeObject && (t.Properties == nil || len(t.Properties) == 0):
		return d.Domain, t, GoEnumType(pdl.TypeAny)

	case t.Type == pdl.TypeObject:
		panic("should not encounter an object with defined properties that does not have Ref and Name")
	}

	return d.Domain, t, GoEnumType(t.Type)
}

// GoName returns the Go name.
func GoName(t *pdl.Type, noExposeOverride bool) string {
	if t.NoExpose || noExposeOverride {
		n := t.Name
		if n != "" && !unicode.IsUpper(rune(n[0])) {
			if goReservedNames[n] {
				n += "Val"
			}
			n = snaker.ForceLowerCamelIdentifier(n)
		}

		return n
	}

	return snaker.ForceCamelIdentifier(t.Name)
}

// GoTypeDef returns the Go type definition for the type.
func GoTypeDef(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool, extra []*pdl.Type, noExposeOverride, omitOnlyWhenOptional bool) string {
	switch {
	case t.Parameters != nil:
		return StructDef(append(extra, t.Parameters...), d, domains, sharedFunc, noExposeOverride, omitOnlyWhenOptional)

	case t.Type == pdl.TypeArray:
		_, o, _ := ResolveType(t.Items, d, domains, sharedFunc)
		return "[]" + GoTypeDef(o, d, domains, sharedFunc, nil, false, false)

	case t.Type == pdl.TypeObject:
		return StructDef(append(extra, t.Properties...), d, domains, sharedFunc, noExposeOverride, omitOnlyWhenOptional)

	case t.Type == pdl.TypeAny && t.Ref != "":
		return t.Ref
	}

	return GoEnumType(t.Type)
}

// GoType returns the Go type for the type.
func GoType(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool) string {
	_, _, z := ResolveType(t, d, domains, sharedFunc)
	return z
}

// EnumValueName returns the name for a enum value.
func EnumValueName(t *pdl.Type, v string) string {
	if t.EnumValueNameMap != nil {
		if e, ok := t.EnumValueNameMap[v]; ok {
			return e
		}
	}

	// special case for "negative" value
	var neg string
	if strings.HasPrefix(v, "-") {
		neg = "Negative"
	}

	return snaker.ForceCamelIdentifier(t.Name) + neg + snaker.ForceCamelIdentifier(v)
}

// GoEmptyValue returns the empty Go value for the type.
func GoEmptyValue(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool) string {
	typ := GoType(t, d, domains, sharedFunc)

	switch {
	case strings.HasPrefix(typ, "[]") || strings.HasPrefix(typ, "*"):
		return "nil"
	}

	return GoEnumEmptyValue(t.Type)
}

// RetTypeList returns a list of the return types.
func RetTypeList(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool) string {
	var s string

	b64ret := Base64EncodedRetParam(t)
	for _, p := range t.Returns {
		if p.Name == Base64EncodedParamName {
			continue
		}

		n := p.Name
		_, _, z := ResolveType(p, d, domains, sharedFunc)

		// if this is a base64 encoded item
		if b64ret != nil && b64ret.Name == p.Name {
			z = "[]byte"
		}

		s += snaker.ForceLowerCamelIdentifier(n) + " " + z + ","
	}

	return strings.TrimSuffix(s, ",")
}

// EmptyRetList returns a list of the empty return values.
func EmptyRetList(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool) string {
	var s string

	b64ret := Base64EncodedRetParam(t)
	for _, p := range t.Returns {
		if p.Name == Base64EncodedParamName {
			continue
		}

		_, o, z := ResolveType(p, d, domains, sharedFunc)
		v := GoEnumEmptyValue(o.Type)
		if strings.HasPrefix(z, "*") || strings.HasPrefix(z, "[]") || (b64ret != nil && b64ret.Name == p.Name) {
			v = "nil"
		}

		s += v + ", "
	}

	return strings.TrimSuffix(s, ", ")
}

// RetNameList returns a <valname>.<name> list for a command's return list.
func RetNameList(t *pdl.Type, valname string, d *pdl.Domain, domains []*pdl.Domain) string {
	var s string
	b64ret := Base64EncodedRetParam(t)
	for _, p := range t.Returns {
		if p.Name == Base64EncodedParamName {
			continue
		}

		n := valname + "." + GoName(p, false)
		if b64ret != nil && b64ret.Name == p.Name {
			n = "dec"
		}

		s += n + ", "
	}

	return strings.TrimSuffix(s, ", ")
}

// Base64EncodedRetParam returns the base64 encoded return parameter, or nil if
// no parameters are base64 encoded.
func Base64EncodedRetParam(t *pdl.Type) *pdl.Type {
	var last *pdl.Type
	for _, p := range t.Returns {
		if p.Name == Base64EncodedParamName {
			return last
		}
		if p.Type == pdl.TypeBinary || strings.HasPrefix(p.Description, Base64EncodedDescriptionPrefix) {
			return p
		}
		last = p
	}
	return nil
}

// StructDef returns a struct definition for a list of types.
func StructDef(types []*pdl.Type, d *pdl.Domain, domains []*pdl.Domain, sharedFunc func(string, string) bool, noExposeOverride, omitOnlyWhenOptional bool) string {
	s := "struct"
	if len(types) > 0 {
		s += " "
	}
	s += "{"
	for _, v := range types {
		s += "\n\t" + GoName(v, noExposeOverride) + " " + GoType(v, d, domains, sharedFunc)

		omit := ",omitempty"
		if (omitOnlyWhenOptional && !v.Optional) || v.AlwaysEmit {
			omit = ""
		}

		// add json tag
		if v.NoExpose {
			s += " `json:\"-\"`"
		} else {
			s += " `json:\"" + v.Name + omit + "\"`"
		}

		// add comment
		if v.Type != pdl.TypeObject && v.Description != "" {
			s += " // " + genutil.CleanDesc(v.Description)
		}
	}
	if len(types) > 0 {
		s += "\n"
	}
	s += "}"

	return s
}

// goReservedNames is the list of reserved names in Go.
var goReservedNames = map[string]bool{
	// language words
	"break":       true,
	"case":        true,
	"chan":        true,
	"const":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"func":        true,
	"go":          true,
	"goto":        true,
	"if":          true,
	"import":      true,
	"interface":   true,
	"map":         true,
	"package":     true,
	"range":       true,
	"return":      true,
	"select":      true,
	"struct":      true,
	"switch":      true,
	"type":        true,
	"var":         true,

	// go types
	"error":      true,
	"bool":       true,
	"string":     true,
	"byte":       true,
	"rune":       true,
	"uintptr":    true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"float32":    true,
	"float64":    true,
	"complex64":  true,
	"complex128": true,
}

// GoEnumType returns the Go type for the TypeEnum.
func GoEnumType(te pdl.TypeEnum) string {
	switch te {
	case pdl.TypeAny:
		return "easyjson.RawMessage"

	case pdl.TypeBoolean:
		return "bool"

	case pdl.TypeInteger:
		return "int64"

	case pdl.TypeNumber:
		return "float64"

	case pdl.TypeString, pdl.TypeBinary:
		return "string"

	case pdl.TypeTimestamp:
		return "time.Time"

	default:
		panic(fmt.Sprintf("called GoEnumType on non primitive type %s", te.String()))
	}
}

// GoEnumEmptyValue returns the Go empty value for the TypeEnum.
func GoEnumEmptyValue(te pdl.TypeEnum) string {
	switch te {
	case pdl.TypeBoolean:
		return `false`

	case pdl.TypeInteger:
		return `0`

	case pdl.TypeNumber:
		return `0`

	case pdl.TypeString, pdl.TypeBinary:
		return `""`

	case pdl.TypeTimestamp:
		return `time.Time{}`
	}

	return `nil`
}

// DocRefLink returns the reference documentation link for the type.
func DocRefLink(t *pdl.Type) string {
	if t.RawSee != "" {
		return t.RawSee
	}

	typ := "type"
	switch t.RawType {
	case "command":
		typ = "method"
	case "event":
		typ = "event"
	}

	i := strings.Index(t.RawName, ".")
	if i == -1 {
		return ""
	}

	domain := t.RawName[:i]
	name := t.RawName[i+1:]
	return ChromeDevToolsDocBase + "/" + domain + "#" + typ + "-" + name
}
