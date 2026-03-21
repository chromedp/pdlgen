package pdl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DefaultParser is the default parser.
var DefaultParser = NewParser()

// Parser wraps a parser.
type Parser struct {
	grab func(string) ([]byte, error)
}

// NewParser creates a new parser.
func NewParser(opts ...ParserOption) *Parser {
	p := new(Parser)
	for _, o := range opts {
		o(p)
	}
	if p.grab == nil {
		p.grab = func(s string) ([]byte, error) {
			return nil, fmt.Errorf("cannot retrieve: %q", s)
		}
	}
	return p
}

// Parse parses a PDL file contained in buf.
//
// Rewrite of the Python script from the Chromium source tree.
//
// See: $CHROMIUM_SOURCE/third_party/inspector_protocol/pdl.py
// Rev: a42a629f67ac9aae0aaa8fbd912c654559c5d880
func (p *Parser) Parse(buf []byte) (*PDL, error) {
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
		includeRE         = regexp.MustCompile(`^include (.*)`)
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

		// include
		if matches := includeRE.FindAllStringSubmatch(line, -1); len(matches) != 0 {
			buf, err := p.grab(matches[0][1])
			if err != nil {
				return nil, fmt.Errorf("include: %w", err)
			}
			inc, err := p.Parse(buf)
			if err != nil {
				return nil, fmt.Errorf("include: parse: %w", err)
			}
			pdl = Combine(pdl, inc)
			continue
		}

		return nil, fmt.Errorf("line %d unknown token %q", i, line)
	}

	return pdl, nil
}

// ParserOption is a parser option.
type ParserOption func(*Parser)

// Grab is a [ParserOption] to set the grab func.
func Grab(grab func(string) ([]byte, error)) ParserOption {
	return func(p *Parser) {
		p.grab = grab
	}
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
