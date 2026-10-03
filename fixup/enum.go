package fixup

import (
	"fmt"
	"strings"

	"github.com/xo/ox/strcase"

	"github.com/chromedp/pdlgen/pdl"
)

// docBase is the base address of the protocol documentation.
const docBase = "https://chromedevtools.github.io/devtools-protocol/tot/"

// extractEnums replaces each inline enum in the domain with a reference to a
// named type, which it adds to the domain.
//
// An inline enum is declared directly on a property of a type. It can also be
// declared on a parameter or a return value of a command or an event. The name
// of the new type is the name of the declaring type, command or event, followed
// by the name of the property. For example, the "type" parameter of
// "Input.dispatchKeyEvent" gets the type "DispatchKeyEventType".
//
// Each place gets a type of its own, even when two places have the same values.
// The protocol changes the values of each place on its own.
func extractEnums(d *pdl.Domain) error {
	// extract from the properties of the types first, as the new types are
	// added to the list
	for _, t := range append([]*pdl.Type(nil), d.Types...) {
		if t.Properties == nil {
			continue
		}
		var err error
		if t.Properties, err = extractProps(d, t, t.Properties, t.Name); err != nil {
			return err
		}
	}
	for _, typs := range [][]*pdl.Type{d.Events, d.Commands} {
		for _, t := range typs {
			var err error
			if t.Parameters, err = extractProps(d, t, t.Parameters, t.Name); err != nil {
				return err
			}
			if t.Returns != nil {
				if t.Returns, err = extractProps(d, t, t.Returns, t.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// extractProps returns props with each inline enum replaced. The name is the
// path to the props, from the type, event or command that declares them.
func extractProps(d *pdl.Domain, parent *pdl.Type, props []*pdl.Type, name string) ([]*pdl.Type, error) {
	r := make([]*pdl.Type, 0, len(props))
	for _, p := range props {
		switch {
		case p.Items != nil && p.Enum != nil:
			// an array of an inline enum
			ref, err := addEnumType(d, parent, p, name)
			if err != nil {
				return nil, err
			}
			c := *p
			c.Enum = nil
			c.Items = &pdl.Type{
				RawType:       p.RawType,
				RawName:       d.Domain.String() + "." + ref,
				IsCircularDep: p.IsCircularDep,
				Name:          p.Name,
				Ref:           ref,
			}
			r = append(r, &c)
		case p.Items != nil:
			// an array of a type that holds an inline enum
			items, err := extractProps(d, parent, []*pdl.Type{p.Items}, name+"."+p.Name)
			if err != nil {
				return nil, err
			}
			c := *p
			c.Items = items[0]
			r = append(r, &c)
		case p.Enum != nil:
			ref, err := addEnumType(d, parent, p, name)
			if err != nil {
				return nil, err
			}
			r = append(r, &pdl.Type{
				RawType:       p.RawType,
				RawName:       d.Domain.String() + "." + ref,
				IsCircularDep: p.IsCircularDep,
				Name:          p.Name,
				Ref:           ref,
				Description:   p.Description,
				Optional:      p.Optional,
			})
		default:
			r = append(r, p)
		}
	}
	return r, nil
}

// addEnumType adds a type for the enum property p to the domain, and returns
// its name.
func addEnumType(d *pdl.Domain, parent, p *pdl.Type, name string) (string, error) {
	ref := strcase.ForceCamelIdentifier(name + "." + p.Name)
	rawName := d.Domain.String() + "." + ref
	for _, t := range d.Types {
		if t.RawName == rawName {
			return "", fmt.Errorf("enum %s.%s needs the type name %s, which the domain %s already has", name, p.Name, ref, d.Domain)
		}
	}
	kind := "type"
	switch parent.RawType {
	case "command":
		kind = "method"
	case "event":
		kind = "event"
	}
	d.Types = append(d.Types, &pdl.Type{
		RawSee:        docBase + strings.Replace(parent.RawName, ".", "#"+kind+"-", 1),
		RawType:       p.RawType,
		RawName:       rawName,
		IsCircularDep: p.IsCircularDep,
		Name:          ref,
		Type:          pdl.TypeString,
		Description:   p.Description,
		Enum:          p.Enum,
	})
	return ref, nil
}
