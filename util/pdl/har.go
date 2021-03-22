package pdl

import (
	_ "embed"
)

//go:generate go run ../../tools/hargen

//go:embed har.pdl
var HAR []byte
