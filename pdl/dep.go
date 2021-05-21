package pdl

import (
	"strings"
)

// circularDeps is the list of types that can cause circular dependency
// issues.
var circularDeps = map[string]bool{
	"browser.browsercontextid":            true,
	"dom.backendnodeid":                   true,
	"dom.backendnode":                     true,
	"dom.compatibilitymode":               true,
	"dom.nodeid":                          true,
	"dom.node":                            true,
	"dom.nodetype":                        true,
	"dom.pseudotype":                      true,
	"dom.rgba":                            true,
	"dom.shadowroottype":                  true,
	"network.loaderid":                    true,
	"network.monotonictime":               true,
	"network.timesinceepoch":              true,
	"page.adframetype":                    true,
	"page.crossoriginisolatedcontexttype": true,
	"page.frameid":                        true,
	"page.frame":                          true,
	"page.gatedapifeatures":               true,
	"page.origintrial":                    true,
	"page.origintrialstatus":              true,
	"page.origintrialtoken":               true,
	"page.origintrialtokenstatus":         true,
	"page.origintrialtokenwithstatus":     true,
	"page.origintrialusagerestriction":    true,
	"page.securecontexttype":              true,
}

// IsCircularDep returns whether or not a type will cause circular dependency
// issues. Useful for generating Go packages.
func IsCircularDep(dtyp, typ string) bool {
	return circularDeps[strings.ToLower(dtyp+"."+typ)]
}
