// Command relnotes writes the commit message, the tag annotation and the
// changelog entry of a release of cdproto from the output of apidiff.
package main

import (
	"context"
	"os"

	"github.com/xo/ox"

	"github.com/chromedp/pdlgen/relnotes"
)

func main() {
	args := relnotes.New()
	ox.RunContext(
		context.Background(),
		ox.Usage("relnotes", "writes the notes of a release of cdproto from the output of apidiff"),
		ox.Defaults(),
		ox.Exec(args.Run(os.Stdout)),
		ox.From(args),
	)
}
