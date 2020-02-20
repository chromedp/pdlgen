// pdlgen is a tool to generate the low-level Chrome DevTools Protocol
// implementation types used by chromedp from the CDP protocol definitions
// (PDLs) in the Chromium source tree.
//
// Please see README.md for more information on using this tool.
package main

//go:generate go run ./tools/qtplgen
//go:generate go run ./tools/precache

import (
	"context"
	"fmt"
	"os"

	"github.com/chromedp/pdlgen/tplgen"
)

func main() {
	if err := tplgen.Exec(context.Background(), os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
