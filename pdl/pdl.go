// Package pdl contains types and funcs for working with Chrome DevTools
// Protocol definitions (ie, PDL files).
package pdl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// PDL contains information about the domains, types, commands, and events of
// the Chrome DevTools Protocol.
type PDL struct {
	// Copyright is the file copyright.
	Copyright string

	// Version is the file version information.
	Version *Version

	// Domains are the available domains.
	Domains []*Domain
}

// Parse parses a PDL file contained in buf.
//
// Rewrite of the Python script from the Chromium source tree.
//
// See: $CHROMIUM_SOURCE/third_party/inspector_protocol/pdl.py
// Rev: a42a629f67ac9aae0aaa8fbd912c654559c5d880
func Parse(buf []byte) (*PDL, error) {
	// regexp's copied from pdl.py in the chromium source tree.
	var (
		domainRE          = regexp.MustCompile(`^(experimental )?(deprecated )?domain (.*)`)
		dependsRE         = regexp.MustCompile(`^  depends on ([^\s]+)`)
		typeRE            = regexp.MustCompile(`^  (experimental )?(deprecated )?type (.*) extends (array of )?([^\s]+)`)
		commandEventRE    = regexp.MustCompile(`^  (experimental )?(deprecated )?(command|event) (.*)`)
		memberRE          = regexp.MustCompile(`^      (experimental )?(deprecated )?(optional )?(array of )?([^\s]+) ([^\s]+)`)
		paramsRetsPropsRE = regexp.MustCompile(`^    (parameters|returns|properties)`)
		enumRE            = regexp.MustCompile(`^    enum`)
		versionRE         = regexp.MustCompile(`^version`)
		majorRE           = regexp.MustCompile(`^  major (\d+)`)
		minorRE           = regexp.MustCompile(`^  minor (\d+)`)
		redirectRE        = regexp.MustCompile(`^    redirect ([^\s]+)`)
		redirectCommentRE = regexp.MustCompile(`^Use '([^']+)' instead$`)
		enumLiteralRE     = regexp.MustCompile(`^      (  )?[^\s]+$`)
	)

	pdl := new(PDL)

	// state objects
	var domain *Domain
	var item *Type
	var subitems *[]*Type
	var enumliterals *[]string
	var desc string
	var copyright, clearDesc bool

	for i, line := range strings.Split(string(buf), "\n") {
		// clear the description if toggled
		if clearDesc {
			desc, clearDesc = "", false
		}

		// trim the line
		trimmed := strings.TrimSpace(line)

		// add to desc
		if strings.HasPrefix(trimmed, "#") {
			if len(desc) != 0 {
				desc += "\n"
			}
			desc += strings.TrimSpace(trimmed[1:])
			continue
		} else {
			if !copyright {
				copyright, pdl.Copyright = true, desc
			}
			clearDesc = true
		}

		// skip empty line
		if len(trimmed) == 0 {
			continue
		}

		// domain
		if matches := domainRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			domain = &Domain{
				Domain:       DomainType(matches[0][3]),
				Experimental: matches[0][1] != "",
				Deprecated:   matches[0][2] != "",
				Description:  strings.TrimSpace(desc),
			}
			pdl.Domains = append(pdl.Domains, domain)
			continue
		}

		// dependencies
		if matches := dependsRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			domain.Dependencies = append(domain.Dependencies, matches[0][1])
			continue
		}

		// type
		if matches := typeRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			item = &Type{
				RawType:       "type",
				RawName:       domain.Domain.String() + "." + matches[0][3],
				IsCircularDep: IsCircularDep(domain.Domain.String(), matches[0][3]),
				Name:          matches[0][3],
				Experimental:  matches[0][1] != "",
				Deprecated:    matches[0][2] != "",
				Description:   strings.TrimSpace(desc),
			}
			assignType(item, matches[0][5], matches[0][4] != "")
			domain.Types = append(domain.Types, item)
			continue
		}

		// command or event
		if matches := commandEventRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			item = &Type{
				RawName:       domain.Domain.String() + "." + matches[0][4],
				IsCircularDep: IsCircularDep(domain.Domain.String(), matches[0][4]),
				Name:          matches[0][4],
				Experimental:  matches[0][1] != "",
				Deprecated:    matches[0][2] != "",
				Description:   strings.TrimSpace(desc),
			}
			if matches[0][3] == "command" {
				item.RawType = "command"
				domain.Commands = append(domain.Commands, item)
			} else {
				item.RawType = "event"
				domain.Events = append(domain.Events, item)
			}
			continue
		}

		// member to params / returns / properties
		if matches := memberRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			param := &Type{
				RawName:       domain.Domain.String() + "." + matches[0][6],
				IsCircularDep: IsCircularDep(domain.Domain.String(), matches[0][6]),
				Name:          matches[0][6],
				Experimental:  matches[0][1] != "",
				Deprecated:    matches[0][2] != "",
				Description:   strings.TrimSpace(desc),
				Optional:      matches[0][3] != "",
			}
			assignType(param, matches[0][5], matches[0][4] != "")
			if matches[0][5] == "enum" {
				param.Enum = make([]string, 0)
				enumliterals = &param.Enum
			}
			*subitems = append(*subitems, param)
			continue
		}

		// parameters, returns, properties definition
		if matches := paramsRetsPropsRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			switch matches[0][1] {
			case "parameters":
				item.Parameters = make([]*Type, 0)
				subitems = &item.Parameters
			case "returns":
				item.Returns = make([]*Type, 0)
				subitems = &item.Returns
			case "properties":
				item.Properties = make([]*Type, 0)
				subitems = &item.Properties
			}
			continue
		}

		// enum
		if matches := enumRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			item.Enum = make([]string, 0)
			enumliterals = &item.Enum
			continue
		}

		// version
		if matches := versionRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			pdl.Version = new(Version)
			continue
		}

		// version major
		if matches := majorRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			pdl.Version.Major, _ = strconv.Atoi(matches[0][1])
			continue
		}

		// version minor
		if matches := minorRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			pdl.Version.Minor, _ = strconv.Atoi(matches[0][1])
			continue
		}

		// redirect
		if matches := redirectRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			item.Redirect = &Redirect{
				Domain: DomainType(matches[0][1]),
			}
			if m := redirectCommentRE.FindAllStringSubmatch(desc, -1); len(m) != 0 {
				name := m[0][1]
				if n := strings.LastIndex(name, "."); n != -1 {
					name = name[n+1:]
				}
				item.Redirect.Name = name
			}
			continue
		}

		// enum literal
		if matches := enumLiteralRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			*enumliterals = append(*enumliterals, trimmed)
			continue
		}

		return nil, fmt.Errorf("line %d unknown token %q", i, line)
	}

	return pdl, nil
}

// primitiveTypes is a map of primitive type names to their enum value.
var primitiveTypes = map[string]TypeEnum{
	"any":     TypeAny,
	"array":   TypeArray,
	"binary":  TypeBinary,
	"boolean": TypeBoolean,
	"integer": TypeInteger,
	"number":  TypeNumber,
	"object":  TypeObject,
	"string":  TypeString,
}

// assignType assigns item as a type of typ.
func assignType(item *Type, typ string, isArray bool) {
	if isArray {
		item.Type = TypeArray
		item.Items = new(Type)
		assignType(item.Items, typ, false)
		return
	}

	if typ == "enum" {
		typ = "string"
	}

	if pt, ok := primitiveTypes[typ]; ok {
		item.Type = pt
	} else {
		item.Ref = typ
	}
}

// LoadFile loads a PDL file from the specified filename.
func LoadFile(filename string) (*PDL, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Parse(buf)
}

// Combine combines domains from multiple PDL definitions into a single PDL.
func Combine(pdls ...*PDL) *PDL {
	pdl := new(PDL)
	for _, p := range pdls {
		if pdl.Copyright == "" {
			pdl.Copyright = p.Copyright
		}
		if pdl.Version == nil {
			pdl.Version = new(Version)
		}
		if p.Version != nil {
			if pdl.Version.Major < p.Version.Major {
				pdl.Version.Major = p.Version.Major
				pdl.Version.Minor = p.Version.Minor
			} else if pdl.Version.Minor < p.Version.Minor {
				pdl.Version.Minor = p.Version.Minor
			}
		}
		pdl.Domains = append(pdl.Domains, p.Domains...)
	}
	return pdl
}

// CombineBytes combines domains from multiple PDL definitions into a single
// PDL.
func CombineBytes(buffers ...[]byte) ([]byte, error) {
	var pdls []*PDL
	for _, buf := range buffers {
		pdl, err := Parse(buf)
		if err != nil {
			return nil, err
		}
		pdls = append(pdls, pdl)
	}
	return Combine(pdls...).Bytes(), nil
}

// Bytes generates file contents for the PDL.
func (pdl *PDL) Bytes() []byte {
	buf := new(bytes.Buffer)

	// writeDesc conditionally writes a description.
	writeDesc := func(desc, indent string) {
		if desc == "" {
			return
		}
		for _, line := range strings.Split(desc, "\n") {
			if line != "" {
				line = " " + line
			}
			fmt.Fprintln(buf, indent+"#"+line)
		}
	}

	// writeDecl writes a declaration line.
	writeDecl := func(typ, name, desc, indent string, experimental, deprecated, optional bool, extra ...string) {
		writeDesc(desc, indent)
		var v []string
		if experimental {
			v = append(v, "experimental")
		}
		if deprecated {
			v = append(v, "deprecated")
		}
		if optional {
			v = append(v, "optional")
		}
		v = append(v, typ, name)
		fmt.Fprintln(buf, indent+strings.Join(append(v, extra...), " "))
	}

	writeRedirect := func(typ *Type, indent string) {
		if typ.Redirect == nil {
			return
		}
		if typ.Redirect.Name != "" {
			fmt.Fprintln(buf, indent+"# Use '"+typ.Redirect.Domain.String()+"."+typ.Redirect.Name+"' instead")
		}
		fmt.Fprintln(buf, indent+"redirect "+typ.Redirect.Domain.String())
	}

	// writeProps writes a list of types for object properties.
	writeProps := func(typ string, indent string, props []*Type) {
		if len(props) == 0 {
			return
		}
		if indent != "" {
			fmt.Fprint(buf, indent)
		}
		fmt.Fprintln(buf, typ)
		for _, p := range props {
			ref := p.Type.String()
			if p.Ref != "" {
				ref = p.Ref
			}
			if p.Type == TypeArray {
				ref = p.Items.Ref
				if ref == "" {
					ref = p.Items.Type.String()
				}
				ref = "array of " + ref
			}
			if len(p.Enum) != 0 {
				ref = "enum"
			}
			writeDecl(ref, p.Name, p.Description, indent+"  ", p.Experimental, p.Deprecated, p.Optional)
			if len(p.Enum) != 0 {
				for _, e := range p.Enum {
					fmt.Fprintln(buf, indent+"    "+e)
				}
			}
		}
	}

	// add copyright
	if pdl.Copyright != "" {
		writeDesc(pdl.Copyright, "")
		fmt.Fprintln(buf)
	}

	// add version
	if pdl.Version != nil {
		fmt.Fprintln(buf, "version")
		fmt.Fprintln(buf, "  major "+strconv.Itoa(pdl.Version.Major))
		fmt.Fprintln(buf, "  minor "+strconv.Itoa(pdl.Version.Minor))
		fmt.Fprintln(buf)
	}

	// copy and sort domains
	domains := make([]*Domain, len(pdl.Domains))
	copy(domains, pdl.Domains)
	sort.Slice(domains, func(i, j int) bool {
		return strings.Compare(domains[i].Domain.String(), domains[j].Domain.String()) < 0
	})

	// write each domain
	for _, d := range domains {
		// write domain stanza
		writeDecl("domain", d.Domain.String(), d.Description, "", d.Experimental, d.Deprecated, false)

		// write depends
		for _, dep := range d.Dependencies {
			fmt.Fprintln(buf, "  depends on "+dep)
		}
		fmt.Fprintln(buf)

		// sort types
		types := make([]*Type, len(d.Types))
		copy(types, d.Types)
		sort.Slice(types, func(i, j int) bool {
			return strings.Compare(types[i].Name, types[i].Name) < 0
		})

		// write types
		for _, typ := range types {
			extends := typ.Type.String()
			if typ.Type == TypeArray {
				if typ.Items != nil {
					extends = typ.Items.Type.String()
					if extends == "" {
						extends = typ.Items.Ref
					}
				}
				extends = "array of " + extends
			}
			writeDecl("type", typ.Name, typ.Description, "  ", typ.Experimental, typ.Deprecated, typ.Optional, "extends", extends)
			writeRedirect(typ, "  ")
			if len(typ.Enum) != 0 {
				fmt.Fprintln(buf, "    enum")
				for _, e := range typ.Enum {
					fmt.Fprintln(buf, "     ", e)
				}
			}
			writeProps("properties", "    ", typ.Properties)
			fmt.Fprintln(buf)
		}

		// sort commands
		commands := make([]*Type, len(d.Commands))
		copy(commands, d.Commands)
		sort.Slice(commands, func(i, j int) bool {
			return strings.Compare(commands[i].Name, commands[i].Name) < 0
		})

		// write commands
		for _, c := range commands {
			writeDecl("command", c.Name, c.Description, "  ", c.Experimental, c.Deprecated, c.Optional)
			writeRedirect(c, "    ")
			writeProps("parameters", "    ", c.Parameters)
			writeProps("returns", "    ", c.Returns)
			fmt.Fprintln(buf)
		}

		// sort events
		events := make([]*Type, len(d.Events))
		copy(events, d.Events)
		sort.Slice(events, func(i, j int) bool {
			return strings.Compare(events[i].Name, events[i].Name) < 0
		})

		// write events
		for _, e := range events {
			writeDecl("event", e.Name, e.Description, "  ", e.Experimental, e.Deprecated, e.Optional)
			writeRedirect(e, "    ")
			writeProps("parameters", "    ", e.Parameters)
			fmt.Fprintln(buf)
		}
	}

	return append(bytes.TrimRightFunc(buf.Bytes(), unicode.IsSpace), '\n')
}

// Version holds information for the the version Chrome DevTools Protocol
// definition.
type Version struct {
	// Major is the major version.
	Major int

	// Minor is the minor version.
	Minor int
}

// Domain represents a Chrome DevTools Protocol domain.
type Domain struct {
	// Domain is the name of the domain.
	Domain DomainType

	// Description is the domain description.
	Description string

	// Experimental indicates whether or not the domain is experimental.
	Experimental bool

	// Deprecated indicates whether or not the domain is deprecated.
	Deprecated bool

	// Dependencies are the domains' dependencies.
	Dependencies []string

	// Types are the list of types in the domain.
	Types []*Type

	// Commands are the list of commands in the domain.
	Commands []*Type

	// Events is the list of events types in the domain.
	Events []*Type
}

// DomainType is the Chrome domain type.
type DomainType string

// String satisfies Stringer.
func (dt DomainType) String() string {
	return string(dt)
}

// Type represents a Chrome DevTools Protocol type.
type Type struct {
	// Type is the base type of the type.
	Type TypeEnum

	// Name is the name of the type.
	Name string

	// Description is the type description.
	Description string

	// Experimental indicates whether or not the type is experimental.
	Experimental bool

	// Deprecated indicates if the type is deprecated. Used for commands and event parameters.
	Deprecated bool

	// Optional indicates whether or not the type is optional.
	Optional bool

	// Ref is the type the object refers to.
	Ref string

	// Items is the contained type for arrays.
	Items *Type

	// Parameters are object parameters for commands or events.
	Parameters []*Type

	// Returns are the return values for commands.
	Returns []*Type

	// Properties are object properties.
	Properties []*Type

	// Redirect is a type to redirect to, if any.
	Redirect *Redirect

	// Enum are string enum values.
	Enum []string

	// ---------------------------------
	// additional fields

	// RawType is the raw type.
	RawType string `json:"-"`

	// RawName is the raw type name.
	RawName string `json:"-"`

	// RawSee is a raw see url reference.
	RawSee string `json:"-"`

	// TimestampType is the timestamp subtype.
	TimestampType TimestampType `json:"-"`

	// IsCircularDep indicates a type that causes circular dependencies.
	IsCircularDep bool `json:"-"`

	// NoExpose toggles whether or not to expose the type.
	NoExpose bool `json:"-"`

	// NoResolve toggles not resolving the type to a domain (ie, for special
	// internal types).
	NoResolve bool `json:"-"`

	// AlwaysEmit forces the value to always be emitted when marshaled to JSON.
	AlwaysEmit bool `json:"-"`

	// EnumValueNameMap is a map to override the generated enum value name.
	EnumValueNameMap map[string]string `json:"-"`

	// EnumBitMask toggles it as a bit mask enum for TypeInteger enums.
	EnumBitMask bool `json:"-"`

	// Extra will be added as output after the the type is emitted.
	Extra string `json:"-"`
}

// TypeEnum is the Chrome domain type enum.
type TypeEnum string

// TypeEnum values.
const (
	TypeAny       TypeEnum = "any"
	TypeArray     TypeEnum = "array"
	TypeBinary    TypeEnum = "binary"
	TypeBoolean   TypeEnum = "boolean"
	TypeInteger   TypeEnum = "integer"
	TypeNumber    TypeEnum = "number"
	TypeObject    TypeEnum = "object"
	TypeString    TypeEnum = "string"
	TypeTimestamp TypeEnum = "timestamp"
)

// String satisfies stringer.
func (te TypeEnum) String() string {
	return string(te)
}

// TimestampType are the various timestamp subtypes.
type TimestampType int

// TimestampType values.
const (
	TimestampTypeMillisecond TimestampType = 1 + iota
	TimestampTypeSecond
	TimestampTypeMonotonic
)

// Redirect holds type redirect information.
type Redirect struct {
	// Domain is the domain to redirect to.
	Domain DomainType

	// Name is the name of the command, event, or type to redirect to.
	Name string
}

// String satisfies the fmt.Stringer interface.
func (r Redirect) String() string {
	if r.Name != "" {
		return r.Domain.String() + "." + r.Name
	}
	return r.Domain.String()
}
