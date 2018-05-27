package pdl

import (
	"fmt"
	"strings"

	"github.com/knq/snaker"
)

// Version contains PDL file version information.
type Version struct {
	// Major is the major version.
	Major int

	// Minor is the minor version.
	Minor int
}

// Domain represents a Chrome Debugging Protocol domain.
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

// Type represents a type in the Chrome Debugging Protocol.
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

	// TimestampType is the timestamp subtype.
	TimestampType TimestampType `json:"-"`

	// NoExpose toggles whether or not to expose the type.
	NoExpose bool `json:"-"`

	// NoResolve toggles not resolving the type to a domain (ie, for special internal types).
	NoResolve bool `json:"-"`

	// EnumValueNameMap is a map to override the generated enum value name.
	EnumValueNameMap map[string]string `json:"-"`

	// EnumBitMask toggles it as a bit mask enum for TypeInteger enums.
	EnumBitMask bool `json:"-"`

	// Extra will be added as output after the the type is emitted.
	Extra string `json:"-"`
	// ---------------------------------
}

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

// Resolve finds the ref in the provided domains, relative to domain d when ref
// is not namespaced.
func Resolve(ref string, d *Domain, domains []*Domain, sharedFunc func(string, string) bool) (DomainType, *Type, string) {
	n := strings.SplitN(ref, ".", 2)

	// determine domain
	dtyp, typ := d.Domain, n[0]
	if len(n) == 2 {
		dtyp, typ = DomainType(n[0]), n[1]
	}

	// determine if ref points to an object
	var other *Type
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
		if d.Domain != DomainType("cdp") {
			s += "cdp."
		}
	} else if dtyp != d.Domain {
		s += strings.ToLower(dtyp.String()) + "."
	}

	return dtyp, other, s + snaker.ForceCamelIdentifier(typ)
}
