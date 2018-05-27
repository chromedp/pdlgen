package pdl

// DomainType is the Chrome domain type.
type DomainType string

// String satisfies Stringer.
func (dt DomainType) String() string {
	return string(dt)
}

// TypeEnum is the Chrome domain type enum.
type TypeEnum string

// TypeEnum values.
const (
	TypeAny       TypeEnum = "any"
	TypeArray     TypeEnum = "array"
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
