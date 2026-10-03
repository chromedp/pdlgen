// Command pdlgen generates the Go package cdproto from the protocol
// definitions in the Chromium and V8 source trees. The package holds the
// commands, events and types of the Chrome DevTools Protocol.
//
// README.md describes how to use the command.
package main

import (
	"context"
	"os"

	"github.com/xo/ox"

	"github.com/chromedp/pdlgen/gencmd"
)

var (
	name    = "pdlgen"
	version = "0.0.0-dev"
)

func main() {
	args := gencmd.New()
	ox.RunContext(
		context.Background(),
		ox.Usage(name, "generates the Go package cdproto from the Chrome DevTools Protocol definitions"),
		ox.VersionString(version),
		ox.Defaults(),
		ox.Exec(args.Run(os.Stdout)),
		ox.From(args),
	)
}
