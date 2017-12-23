package types

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
)
