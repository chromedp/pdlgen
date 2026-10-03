package gotpl

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/chromedp/pdlgen/gen/genutil"
	"github.com/chromedp/pdlgen/pdl"
	"github.com/xo/ox/strcase"
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
	CommandReturnsSuffix = "Result"
	// Base64Prefix is the start of the description of a text value that is
	// base64 encoded.
	Base64Prefix = "Base64-encoded"
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
	return strcase.ForceCamelIdentifier(t.Name)
}

// EventMethodType returns the method type of the event.
func EventMethodType(t *pdl.Type, d *pdl.Domain) string {
	return EventMethodPrefix + strcase.ForceCamelIdentifier(ProtoName(t, d)) + EventMethodSuffix
}

// CommandMethodType returns the method type of the command.
func CommandMethodType(t *pdl.Type, d *pdl.Domain) string {
	return CommandMethodPrefix + strcase.ForceCamelIdentifier(ProtoName(t, d)) + CommandMethodSuffix
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

// ResolveRef resolves the fully qualified name of the ref of a type from the
// list of domains. When the ref has no domain, ResolveRef resolves it relative
// to domain d.
func ResolveRef(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain) (pdl.DomainType, *pdl.Type) {
	n := strings.SplitN(t.Ref, ".", 2)
	// determine domain
	dtyp, typ := d.Domain, n[0]
	if len(n) == 2 {
		dtyp, typ = pdl.DomainType(n[0]), n[1]
	}
	ref := strings.ToLower(dtyp.String() + "." + typ)
	shortRef := strings.ToLower(typ)
	// determine if ref points to an object
	var resolved *pdl.Type
	for _, z := range domains {
		if dtyp == z.Domain {
			for _, j := range z.Types {
				if z.Domain == "cdp" && shortRef == strings.ToLower(strings.SplitN(j.RawName, ".", 2)[1]) {
					resolved = j
					break
				} else if ref == strings.ToLower(j.RawName) {
					resolved = j
					break
				}
			}
			break
		}
	}
	if resolved == nil {
		panic(fmt.Sprintf("could not resolve type %s in domain %s", ref, d.Domain))
	}
	return dtyp, resolved
}

// ResolveType resolves the type relative to the Go domain.
//
// It returns the DomainType of the underlying type and the underlying type,
// which is the original type when it is not a reference. It also returns the
// fully qualified type name.
func ResolveType(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain) (pdl.DomainType, *pdl.Type, string) {
	switch {
	case t.NoResolve || strings.HasPrefix(t.Ref, "*"):
		return d.Domain, t, t.Ref
	case t.Ref != "":
		dtyp, typ := ResolveRef(t, d, domains)
		// add prefix if is a type defined as having circular dependency issues
		var s string
		switch {
		case typ.IsCircularDep && d.Domain == "cdp":
		case typ.IsCircularDep && d.Domain != "cdp":
			s = "cdp."
		case dtyp != d.Domain:
			s = strings.ToLower(dtyp.String()) + "."
		}
		// Add a pointer for an object that is a struct. An object without
		// properties is a map, which needs none.
		var ptr string
		if typ.Type == pdl.TypeObject && typ.Properties != nil {
			ptr = "*"
		}
		return dtyp, typ, ptr + s + strcase.ForceCamelIdentifier(typ.Name)
	case t.Type == pdl.TypeArray:
		dtyp, typ, z := ResolveType(t.Items, d, domains)
		return dtyp, typ, "[]" + z
	case t.Type == pdl.TypeObject && (t.Properties == nil || len(t.Properties) == 0):
		return d.Domain, t, GoEnumType(pdl.TypeAny)
	case t.Type == pdl.TypeObject:
		panic("should not encounter an object with defined properties that does not have Ref and Name")
	case t.Type == pdl.TypeString && t.Enum == nil && strings.HasPrefix(t.Description, Base64Prefix):
		// The description says that the text is base64, so the value is a
		// []byte, which the JSON package decodes from base64.
		return d.Domain, t, "[]byte"
	}
	return d.Domain, t, GoEnumType(t.Type)
}

// GoName returns the Go name.
func GoName(t *pdl.Type, noExposeOverride bool) string {
	if noExposeOverride {
		n := t.Name
		if n != "" && !unicode.IsUpper(rune(n[0])) {
			if goReservedNames[n] {
				n += "Val"
			}
			n = strcase.ForceLowerCamelIdentifier(n)
		}
		return n
	}
	return strcase.ForceCamelIdentifier(t.Name)
}

// GoTypeDef returns the Go type definition for the type.
func GoTypeDef(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain, noExposeOverride, omitOnlyWhenOptional bool) string {
	switch {
	case t.Parameters != nil:
		return StructDef(t.Parameters, d, domains, noExposeOverride, omitOnlyWhenOptional, t.RawType == "command", structKey(t))
	case t.Type == pdl.TypeArray:
		_, o, _ := ResolveType(t.Items, d, domains)
		return "[]" + GoTypeDef(o, d, domains, false, false)
	case t.Type == pdl.TypeObject && t.Properties == nil:
		// The protocol declares no properties, so the object holds any keys,
		// as Network.Headers does. An empty struct cannot hold them.
		return "map[string]any"
	case t.Type == pdl.TypeObject:
		return StructDef(t.Properties, d, domains, noExposeOverride, omitOnlyWhenOptional, false, structKey(t))
	case t.Type == pdl.TypeAny && t.Ref != "":
		return t.Ref
	}
	return GoEnumType(t.Type)
}

// GoType returns the Go type for the type.
func GoType(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain) string {
	_, _, z := ResolveType(t, d, domains)
	return z
}

// EnumValueName returns the name for an enum value.
func EnumValueName(t *pdl.Type, v string) string {
	// special case for "negative" value
	var neg string
	if strings.HasPrefix(v, "-") {
		neg = "Negative"
	}
	return strcase.ForceCamelIdentifier(t.Name) + neg + strcase.ForceCamelIdentifier(v)
}

// GoEmptyValue returns the empty Go value for the type.
func GoEmptyValue(t *pdl.Type, d *pdl.Domain, domains []*pdl.Domain) string {
	typ := GoType(t, d, domains)
	switch {
	case strings.HasPrefix(typ, "[]") || strings.HasPrefix(typ, "*"):
		return "nil"
	}
	return GoEnumEmptyValue(t.Type)
}

// StructDef returns a struct definition for a list of types.
//
// When input is true, the struct holds the parameters of a command. An optional
// boolean is then a pointer. A nil pointer leaves the parameter out of the
// message, so the browser uses the default of the protocol. A pointer to false
// sends false. A boolean in any other struct is a plain bool, and the message
// always holds it. A binary value is a byte slice, which the JSON package
// encodes as base64 without a tag.
//
// An optional number or integer that is listed in pointerNumbers is a pointer
// in the same way. The key names the struct, as structKey returns it, and is
// empty for a struct that has no such fields.
func StructDef(types []*pdl.Type, d *pdl.Domain, domains []*pdl.Domain, noExposeOverride, omitOnlyWhenOptional, input bool, key string) string {
	var s strings.Builder
	s.WriteString("struct")
	if len(types) > 0 {
		s.WriteString(" ")
	}
	s.WriteString("{")
	for _, typ := range types {
		goType := GoType(typ, d, domains)
		omit := ",omitempty,omitzero"
		switch {
		case input && typ.Optional && typ.Type == pdl.TypeBoolean:
			goType = "*" + goType
		case isPointerNumber(key, typ):
			goType = "*" + goType
		case (omitOnlyWhenOptional && !typ.Optional) || (typ.Type == pdl.TypeBoolean):
			omit = ""
		}
		s.WriteString("\n\t" + GoName(typ, noExposeOverride) + " " + goType)
		// add json tag
		s.WriteString(" `json:\"" + typ.Name + omit + "\"`")
		// add comment
		if typ.Type != pdl.TypeObject && typ.Description != "" {
			s.WriteString(" // " + genutil.CleanDesc(typ.Description))
		}
	}
	if len(types) > 0 {
		s.WriteString("\n")
	}
	s.WriteString("}")
	return s.String()
}

// pointerNumbers lists the optional numbers and integers that are pointers. The
// key is the protocol name of a command, for its parameters, or of a type. The
// value is the protocol names of the fields.
//
// A plain number is left out of the message when it is zero. That is wrong for
// a field where zero is a value of its own and an absent field means something
// else: a default that is not zero, no change, no limit or a cleared state. A
// nil pointer leaves the field out, and a pointer to zero sends zero. The
// decision 2026-10-04-an-optional-number-can-be-a-pointer.md says how the list
// was made. Add a field here only when the protocol description of the field
// shows that zero and absent differ. A name that the protocol no longer has is
// ignored.
var pointerNumbers = map[string][]string{
	"Accessibility.getFullAXTree":           {"depth"},
	"Animation.seekAnimations":              {"currentTime"},
	"Audits.getEncodedResponse":             {"quality"},
	"Browser.Bounds":                        {"left", "top"},
	"CSS.forcePositionTryOption":            {"index"},
	"DOM.describeNode":                      {"depth"},
	"DOM.getDocument":                       {"depth"},
	"DOM.requestChildNodes":                 {"depth"},
	"DOMDebugger.getEventListeners":         {"depth"},
	"Debugger.Location":                     {"columnNumber"},
	"Debugger.setBreakpointByUrl":           {"columnNumber"},
	"Emulation.SafeAreaInsets":              {"top", "topMax", "left", "leftMax", "bottom", "bottomMax", "right", "rightMax"},
	"Emulation.setGeolocationOverride":      {"latitude", "longitude", "accuracy", "altitude", "altitudeAccuracy", "heading", "speed"},
	"Emulation.setVirtualTimePolicy":        {"budget"},
	"Emulation.updateScreen":                {"left", "top", "rotation"},
	"HAR.Content":                           {"compression"},
	"HAR.PageTimings":                       {"onContentLoad", "onLoad"},
	"HAR.Timings":                           {"blocked", "dns", "connect", "ssl"},
	"HeadlessExperimental.ScreenshotParams": {"quality"},
	"IO.read":                               {"offset"},
	"IndexedDB.Key":                         {"number", "date"},
	"Input.TouchPoint":                      {"id", "radiusX", "radiusY", "force"},
	"Input.imeSetComposition":               {"replacementStart", "replacementEnd"},
	"Input.synthesizeScrollGesture":         {"repeatDelayMs"},
	"Input.synthesizeTapGesture":            {"duration"},
	"LayerTree.replaySnapshot":              {"toStep"},
	"Network.configureDurableMessages":      {"maxTotalBufferSize", "maxResourceBufferSize"},
	"Network.enable":                        {"maxTotalBufferSize", "maxResourceBufferSize", "maxPostDataSize"},
	"Overlay.DisplayCutoutConfig":           {"cx", "cy"},
	"Page.captureScreenshot":                {"quality"},
	"Page.printToPDF":                       {"marginTop", "marginBottom", "marginLeft", "marginRight"},
	"Page.startScreencast":                  {"quality"},
	"Runtime.SerializationOptions":          {"maxDepth"},
	"Storage.overrideQuotaForOrigin":        {"quotaSize"},
	"Target.createTarget":                   {"left", "top"},
	"WebAuthn.Credential":                   {"activeCmtgKeyIndex"},
	"WebAuthn.setCredentialProperties":      {"activeCmtgKeyIndex", "signCount"},
}

// structKey returns the key of the struct in pointerNumbers. It is the
// protocol name of t when t is a command, for its parameters, or a type. It is
// empty for an event and for the result of a command, because a caller reads
// those and does not write them.
func structKey(t *pdl.Type) string {
	if t.RawType == "command" || t.RawType == "type" {
		return t.RawName
	}
	return ""
}

// isPointerNumber returns true when the field is an optional plain number or
// integer that pointerNumbers lists for the struct key.
func isPointerNumber(key string, field *pdl.Type) bool {
	if key == "" || !field.Optional || field.Ref != "" || field.Enum != nil {
		return false
	}
	if field.Type != pdl.TypeNumber && field.Type != pdl.TypeInteger {
		return false
	}
	return slices.Contains(pointerNumbers[key], field.Name)
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
		return "jsontext.Value"
	case pdl.TypeBoolean:
		return "bool"
	case pdl.TypeInteger:
		return "int64"
	case pdl.TypeNumber:
		return "float64"
	case pdl.TypeString:
		return "string"
	case pdl.TypeBinary:
		return "[]byte"
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
	case pdl.TypeString:
		return `""`
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
	domain, name := t.RawName[:i], t.RawName[i+1:]
	return ChromeDevToolsDocBase + "/" + domain + "#" + typ + "-" + name
}
