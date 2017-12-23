// Package fixup modifies/alters/fixes and adds to the low level type
// definitions for the Chrome Debugging Protocol domains, as generated from
// protocol.json.
//
// The goal of package fixup is to fix the issues associated with generating Go
// code from the existing Chrome domain definitions, and is wrapped up in one
// high-level func, FixDomains.
//
// Currently, FixDomains will do the following:
//  - add 'Inspector.MethodType' type as a string enumeration of all the event/command names.
//  - add 'Inspector.MessageError' type as a object with code (integer), and message (string).
//  - add 'Inspector.Message' type as a object with id (integer), method (MethodType), params (interface{}), error (MessageError).
//  - add 'Inspector.DetachReason' type and change event 'Inspector.detached''s parameter reason's type.
//  - add 'Inspector.ErrorType' type.
//  - change type of Network.TimeSinceEpoch, Network.MonotonicTime, and
//    Runtime.Timestamp to internal Timestamp type.
//  - convert object properties and event/command parameters that are enums into independent types.
//  - change '*.modifiers' parameters to type Input.Modifier.
//  - add 'DOM.NodeType' type and convert "nodeType" parameters to it.
//  - change Page.Frame.id/parentID to FrameID type.
//  - add additional properties to 'Page.Frame' and 'DOM.Node' for use by higher level packages.
//  - add special unmarshaler to NodeId, BackendNodeId, FrameId to handle
//    values from older (v1.1) protocol versions. -- NOTE: this might need to be
//    applied to more types, such as network.LoaderId
//  - rename 'Input.GestureSourceType' -> 'Input.GestureType'.
//  - rename CSS.CSS* types.
//  - add Error() method to 'Runtime.ExceptionDetails' type so that it can be used as error.
//  - change 'Network.Headers' type to map[string]interface{}.
//
// Please note that the above is not an exhaustive list of all modifications
// applied to the domains, however it does attempt to give a comprehensive
// overview of the most important changes to the definition vs the vanilla
// specification.
package fixup

import (
	"fmt"
	"strings"

	"github.com/knq/snaker"

	"github.com/chromedp/chromedp-gen/templates"
	"github.com/chromedp/chromedp-gen/types"
)

// Specific type names to use for the applied fixes to the protocol domains.
//
// These need to be here in case the location of these types change (see above)
// relative to the generated 'cdp' package.
const (
	domNodeIDRef = "NodeID"
	domNodeRef   = "*Node"
)

// FixDomains modifies, updates, alters, fixes, and adds to the types defined
// in the domains, so that the generated Chrome Debugging Protocol domain code
// is more Go-like and easier to use.
//
// Please see package-level documentation for the list of changes made to the
// various debugging protocol domains.
func FixDomains(domains []*types.Domain) {
	// method type
	methodType := &types.Type{
		ID:               "MethodType",
		Type:             types.TypeString,
		Description:      "Chrome Debugging Protocol method type (ie, event and command names).",
		EnumValueNameMap: make(map[string]string),
		Extra:            templates.ExtraMethodTypeDomainDecoder(),
	}

	// message error type
	messageErrorType := &types.Type{
		ID:          "MessageError",
		Type:        types.TypeObject,
		Description: "Message error type.",
		Properties: []*types.Type{{
			Name:        "code",
			Type:        types.TypeInteger,
			Description: "Error code.",
		}, {
			Name:        "message",
			Type:        types.TypeString,
			Description: "Error message.",
		}},
		Extra: `// Error satisfies error interface.
func (e *MessageError) Error() string {
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}
`,
	}

	// message type
	messageType := &types.Type{
		ID:          "Message",
		Type:        types.TypeObject,
		Description: "Chrome Debugging Protocol message sent to/read over websocket connection.",
		Properties: []*types.Type{{
			Name:        "id",
			Type:        types.TypeInteger,
			Description: "Unique message identifier.",
			Optional:    true,
		}, {
			Name:        "method",
			Ref:         "Inspector.MethodType",
			Description: "Event or command type.",
			Optional:    true,
		}, {
			Name:        "params",
			Type:        types.TypeAny,
			Description: "Event or command parameters.",
			Optional:    true,
		}, {
			Name:        "result",
			Type:        types.TypeAny,
			Description: "Command return values.",
			Optional:    true,
		}, {
			Name:        "error",
			Ref:         "MessageError",
			Description: "Error message.",
			Optional:    true,
		}},
	}

	// detach reason type
	detachReasonType := &types.Type{
		ID:          "DetachReason",
		Type:        types.TypeString,
		Enum:        []string{"target_closed", "canceled_by_user", "replaced_with_devtools", "Render process gone."},
		Description: "Detach reason.",
	}

	// cdp error types
	errorValues := []string{"channel closed", "invalid result", "unknown result"}
	errorValueNameMap := make(map[string]string)
	for _, e := range errorValues {
		errorValueNameMap[e] = "Err" + snaker.ForceCamelIdentifier(e)
	}
	errorType := &types.Type{
		ID:               "ErrorType",
		Type:             types.TypeString,
		Enum:             errorValues,
		EnumValueNameMap: errorValueNameMap,
		Description:      "Error type.",
		Extra:            templates.ExtraCDPTypes(),
	}

	// modifier type
	modifierType := &types.Type{
		ID:          "Modifier",
		Type:        types.TypeInteger,
		EnumBitMask: true,
		Description: "Input key modifier type.",
		Enum:        []string{"None", "Alt", "Ctrl", "Meta", "Shift"},
		Extra: `// ModifierCommand is an alias for ModifierMeta.
const ModifierCommand Modifier = ModifierMeta`,
	}

	// node type type -- see: https://developer.mozilla.org/en/docs/Web/API/Node/nodeType
	nodeTypeType := &types.Type{
		ID:          "NodeType",
		Type:        types.TypeInteger,
		Description: "Node type.",
		Enum: []string{
			"Element", "Attribute", "Text", "CDATA", "EntityReference",
			"Entity", "ProcessingInstruction", "Comment", "Document",
			"DocumentType", "DocumentFragment", "Notation",
		},
	}

	// process domains
	for _, d := range domains {
		switch d.Domain {
		case "Inspector":
			// add Inspector types
			d.Types = append(d.Types, messageErrorType, messageType, methodType, detachReasonType, errorType)

			// find detached event's reason parameter and change type
			for _, e := range d.Events {
				if e.Name == "detached" {
					for _, t := range e.Parameters {
						if t.Name == "reason" {
							t.Ref = "DetachReason"
							t.Type = types.TypeEnum("")
							break
						}
					}
					break
				}
			}

		case "CSS":
			for _, t := range d.Types {
				if t.ID == "CSSComputedStyleProperty" {
					t.ID = "ComputedProperty"
				}
			}

		case "Input":
			// add Input types
			d.Types = append(d.Types, modifierType)
			for _, t := range d.Types {
				switch t.ID {
				case "GestureSourceType":
					t.ID = "GestureType"

				case "TimeSinceEpoch":
					t.Type = types.TypeTimestamp
					t.TimestampType = types.TimestampTypeSecond
					t.Extra = templates.ExtraTimestampTemplate(t, d)
				}
			}

		case "DOM":
			// add DOM types
			d.Types = append(d.Types, nodeTypeType)

			for _, t := range d.Types {
				switch t.ID {
				case "NodeId", "BackendNodeId":
					t.Extra += templates.ExtraFixStringUnmarshaler(snaker.ForceCamelIdentifier(t.ID), "ParseInt", ", 10, 64")

				case "Node":
					t.Properties = append(t.Properties,
						&types.Type{
							Name:        "Parent",
							Ref:         domNodeRef,
							Description: "Parent node.",
							NoResolve:   true,
							NoExpose:    true,
						},
						&types.Type{
							Name:        "Invalidated",
							Ref:         "chan struct{}",
							Description: "Invalidated channel.",
							NoResolve:   true,
							NoExpose:    true,
						},
						&types.Type{
							Name:        "State",
							Ref:         "NodeState",
							Description: "Node state.",
							NoResolve:   true,
							NoExpose:    true,
						},
						&types.Type{
							Name:        "",
							Ref:         "sync.RWMutex",
							Description: "Read write mutex.",
							NoResolve:   true,
							NoExpose:    true,
						},
					)
					t.Extra += templates.ExtraNodeTemplate()
				}
			}

		case "Page":
			for _, t := range d.Types {
				switch t.ID {
				case "FrameId":
					t.Extra += templates.ExtraFixStringUnmarshaler(snaker.ForceCamelIdentifier(t.ID), "", "")

				case "Frame":
					t.Properties = append(t.Properties,
						&types.Type{
							Name:        "State",
							Ref:         "FrameState",
							Description: "Frame state.",
							NoResolve:   true,
							NoExpose:    true,
						},
						&types.Type{
							Name:        "Root",
							Ref:         domNodeRef,
							Description: "Frame document root.",
							NoResolve:   true,
							NoExpose:    true,
						},
						&types.Type{
							Name:        "Nodes",
							Ref:         "map[" + domNodeIDRef + "]" + domNodeRef,
							Description: "Frame nodes.",
							NoResolve:   true,
							NoExpose:    true,
						},
						&types.Type{
							Name:        "",
							Ref:         "sync.RWMutex",
							Description: "Read write mutex.",
							NoResolve:   true,
							NoExpose:    true,
						},
					)
					t.Extra += templates.ExtraFrameTemplate()

					// convert Frame.id/parentId to $ref of FrameID
					for _, p := range t.Properties {
						if p.Name == "id" || p.Name == "parentId" {
							p.Ref = "FrameId"
							p.Type = types.TypeEnum("")
						}
					}
				}
			}

		case "Network":
			for _, t := range d.Types {
				// change Monotonic to TypeTimestamp and add extra unmarshaling template
				if t.ID == "TimeSinceEpoch" {
					t.Type = types.TypeTimestamp
					t.TimestampType = types.TimestampTypeSecond
					t.Extra = templates.ExtraTimestampTemplate(t, d)
				}

				// change Monotonic to TypeTimestamp and add extra unmarshaling template
				if t.ID == "MonotonicTime" {
					t.Type = types.TypeTimestamp
					t.TimestampType = types.TimestampTypeMonotonic
					t.Extra = templates.ExtraTimestampTemplate(t, d)
				}

				// change Headers to be a map[string]interface{}
				if t.ID == "Headers" {
					t.Type = types.TypeAny
					t.Ref = "map[string]interface{}"
				}
			}

		case "Runtime":
			var typs []*types.Type
			for _, t := range d.Types {
				switch t.ID {
				case "Timestamp":
					t.Type = types.TypeTimestamp
					t.TimestampType = types.TimestampTypeMillisecond
					t.Extra += templates.ExtraTimestampTemplate(t, d)

				case "ExceptionDetails":
					t.Extra += templates.ExtraExceptionDetailsTemplate()
				}

				typs = append(typs, t)
			}
			d.Types = typs
		}

		for _, t := range d.Types {
			// convert object properties
			if t.Properties != nil {
				t.Properties = convertObjectProperties(t.Properties, d, t.ID)
			}
		}

		// process events and commands
		processParameters(d, d.Events)
		processParameters(d, d.Commands)

		// fix input enums
		if d.Domain == "Input" {
			for _, t := range d.Types {
				if t.Enum != nil && t.ID != "Modifier" {
					t.EnumValueNameMap = make(map[string]string)
					for _, v := range t.Enum {
						prefix := ""
						switch t.ID {
						case "GestureType":
							prefix = "Gesture"
						case "ButtonType":
							prefix = "Button"
						}
						n := prefix + snaker.ForceCamelIdentifier(v)
						if t.ID == "KeyType" {
							n = "Key" + strings.Replace(n, "Key", "", -1)
						}
						t.EnumValueNameMap[v] = strings.Replace(n, "Cancell", "Cancel", -1)
					}
				}
			}
		}

		for _, t := range d.Types {
			// fix type stuttering
			if !t.NoExpose && !t.NoResolve {
				id := strings.TrimPrefix(t.ID, d.Domain.String())
				if id == "" {
					continue
				}

				t.ID = id
			}
		}
	}

}

// processParameters the Parameters and Returns properties.
func processParameters(d *types.Domain, typs []*types.Type) {
	for _, t := range typs {
		t.Parameters = convertObjectProperties(t.Parameters, d, t.Name)
		if t.Returns != nil {
			t.Returns = convertObjectProperties(t.Returns, d, t.Name)
		}
	}
}

// convertObjectProperties converts object properties.
func convertObjectProperties(params []*types.Type, d *types.Domain, name string) []*types.Type {
	r := make([]*types.Type, 0)
	for _, p := range params {
		switch {
		case p.Items != nil:
			r = append(r, &types.Type{
				Name:        p.Name,
				Type:        types.TypeArray,
				Description: p.Description,
				Optional:    p.Optional,
				Items:       convertObjectProperties([]*types.Type{p.Items}, d, name+"."+p.Name)[0],
			})

		case p.Enum != nil:
			r = append(r, fixupEnumParameter(name, p, d))

		case p.Name == "modifiers":
			r = append(r, &types.Type{
				Name:        p.Name,
				Ref:         "Modifier",
				Description: p.Description,
				Optional:    p.Optional,
			})

		case p.Name == "nodeType":
			r = append(r, &types.Type{
				Name:        p.Name,
				Ref:         "DOM.NodeType",
				Description: p.Description,
				Optional:    p.Optional,
			})

		case p.Ref == "GestureSourceType":
			r = append(r, &types.Type{
				Name:        p.Name,
				Ref:         "GestureType",
				Description: p.Description,
				Optional:    p.Optional,
			})

		case p.Ref == "CSSComputedStyleProperty":
			r = append(r, &types.Type{
				Name:        p.Name,
				Ref:         "ComputedProperty",
				Description: p.Description,
				Optional:    p.Optional,
			})

		case p.Ref != "" && !p.NoExpose && !p.NoResolve:
			ref := strings.SplitN(p.Ref, ".", 2)
			if len(ref) == 1 {
				ref[0] = strings.TrimPrefix(ref[0], d.Domain.String())
			} else {
				ref[1] = strings.TrimPrefix(ref[1], ref[0])
			}

			z := strings.Join(ref, ".")
			if z == "" {
				z = p.Ref
			}

			r = append(r, &types.Type{
				Name:        p.Name,
				Ref:         z,
				Description: p.Description,
				Optional:    p.Optional,
			})

		default:
			r = append(r, p)
		}
	}

	return r
}

// addEnumValues adds orig.Enum values to type named n's Enum values in domain.
func addEnumValues(d *types.Domain, n string, p *types.Type) {
	// find type
	var typ *types.Type
	for _, t := range d.Types {
		if t.ID == n {
			typ = t
			break
		}
	}
	if typ == nil {
		typ = &types.Type{
			ID:          n,
			Type:        types.TypeString,
			Description: p.Description,
			Optional:    p.Optional,
		}
		d.Types = append(d.Types, typ)
	}

	// combine typ.Enum and vals
	v := make(map[string]bool)
	all := append(typ.Enum, p.Enum...)
	for _, z := range all {
		v[z] = false
	}

	var i int
	typ.Enum = make([]string, len(v))
	for _, z := range all {
		if !v[z] {
			typ.Enum[i] = z
			i++
		}
		v[z] = true
	}
}

// enumRefMap is the fully qualified parameter name to ref.
var enumRefMap = map[string]string{
	"Animation.Animation.type":                         "Type",
	"Console.ConsoleMessage.level":                     "MessageLevel",
	"Console.ConsoleMessage.source":                    "MessageSource",
	"CSS.CSSMedia.source":                              "MediaSource",
	"CSS.forcePseudoState.forcedPseudoClasses":         "PseudoClass",
	"Debugger.setPauseOnExceptions.state":              "ExceptionsState",
	"Emulation.ScreenOrientation.type":                 "OrientationType",
	"Emulation.setTouchEmulationEnabled.configuration": "EnabledConfiguration",
	"Input.dispatchKeyEvent.type":                      "KeyType",
	"Input.dispatchMouseEvent.button":                  "ButtonType",
	"Input.dispatchMouseEvent.type":                    "MouseType",
	"Input.dispatchTouchEvent.type":                    "TouchType",
	"Input.emulateTouchFromMouseEvent.button":          "ButtonType",
	"Input.emulateTouchFromMouseEvent.type":            "MouseType",
	"Input.TouchPoint.state":                           "TouchState",
	"Log.LogEntry.level":                               "Level",
	"Log.LogEntry.source":                              "Source",
	"Log.ViolationSetting.name":                        "Violation",
	"Network.Request.mixedContentType":                 "MixedContentType",
	"Network.Request.referrerPolicy":                   "ReferrerPolicy",
	"Page.startScreencast.format":                      "ScreencastFormat",
	"Runtime.consoleAPICalled.type":                    "APIType",
	"Runtime.ObjectPreview.subtype":                    "Subtype",
	"Runtime.ObjectPreview.type":                       "Type",
	"Runtime.PropertyPreview.subtype":                  "Subtype",
	"Runtime.PropertyPreview.type":                     "Type",
	"Runtime.RemoteObject.subtype":                     "Subtype",
	"Runtime.RemoteObject.type":                        "Type",
	"Tracing.start.transferMode":                       "TransferMode",
	"Tracing.TraceConfig.recordMode":                   "RecordMode",
}

// fixupEnumParameter takes an enum parameter, adds it to the domain and
// returns a type suitable for use in place of the type.
func fixupEnumParameter(typ string, p *types.Type, d *types.Domain) *types.Type {
	fqname := strings.TrimSuffix(fmt.Sprintf("%s.%s.%s", d.Domain, typ, p.Name), ".")
	ref := snaker.ForceCamelIdentifier(typ + "." + p.Name)
	if n, ok := enumRefMap[fqname]; ok {
		ref = n
	}

	// add enum values to type name
	addEnumValues(d, ref, p)

	return &types.Type{
		Name:        p.Name,
		Ref:         ref,
		Description: p.Description,
		Optional:    p.Optional,
	}
}
