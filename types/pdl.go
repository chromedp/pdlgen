package types

import (
	"fmt"
	"regexp"
	"strings"
)

// Parse parses a PDL file into a ProtocolInfo.
//
// Rewrite of the Python script from the Chromium source tree.
//
// See: $CHROMIUM_SOURCE/third_party/inspector_protocol/pdl.py
func Parse(buf []byte) (*ProtocolInfo, error) {
	protoInfo := new(ProtocolInfo)

	var domain *Domain
	var item *Type
	var subitems *[]*Type
	var enumliterals *[]string
	var desc string
	var clearDesc bool

	for i, line := range strings.Split(string(buf), "\n") {
		// clear the description if toggled
		if clearDesc {
			desc = ""
			clearDesc = false
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
			protoInfo.Domains = append(protoInfo.Domains, domain)
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
				ID:           matches[0][3],
				Experimental: matches[0][1] != "",
				Deprecated:   matches[0][2] != "",
				Description:  strings.TrimSpace(desc),
			}
			assignType(item, matches[0][5], matches[0][4] != "")
			domain.Types = append(domain.Types, item)
			continue
		}

		// command or event
		if matches := commandEventRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			item = &Type{
				Name:         matches[0][4],
				Experimental: matches[0][1] != "",
				Deprecated:   matches[0][2] != "",
				Description:  strings.TrimSpace(desc),
			}
			if matches[0][3] == "command" {
				domain.Commands = append(domain.Commands, item)
			} else {
				domain.Events = append(domain.Events, item)
			}
			continue
		}

		// member to params / returns / properties
		if matches := memberRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			param := &Type{
				Name:         matches[0][6],
				Experimental: matches[0][1] != "",
				Deprecated:   matches[0][2] != "",
				Description:  strings.TrimSpace(desc),
				Optional:     matches[0][3] != "",
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
			protoInfo.Version = new(Version)
			continue
		}

		// version major
		if matches := majorRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			protoInfo.Version.Major = matches[0][1]
			continue
		}

		// version minor
		if matches := minorRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			protoInfo.Version.Minor = matches[0][1]
			continue
		}

		// redirect
		if matches := redirectRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			item.Redirect = DomainType(matches[0][1])
			continue
		}

		// enum literal
		if matches := enumLiteralRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			*enumliterals = append(*enumliterals, trimmed)
			continue
		}

		return nil, fmt.Errorf("line %d unknown token %q", i, line)
	}

	return protoInfo, nil
}

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
	enumLiteralRE     = regexp.MustCompile(`^      (  )?[^\s]+$`)
)

// assignType assigns item to be of type typ.
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

// primitiveTypes is a map of primitive type names to their enum value.
var primitiveTypes = map[string]TypeEnum{
	"integer": TypeInteger,
	"number":  TypeNumber,
	"boolean": TypeBoolean,
	"string":  TypeString,
	"object":  TypeObject,
	"any":     TypeAny,
	"array":   TypeArray,
}
