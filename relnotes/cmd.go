package relnotes

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// Args are the relnotes command arguments.
type Args struct {
	Mode            string `ox:"what to write - commit/tag/changelog,default:commit"`
	Diff            string `ox:"file with the output of apidiff -m - empty for the first release"`
	Version         string `ox:"version of the release"`
	Previous        string `ox:"version of the previous release - empty for the first"`
	Chromium        string `ox:"chromium version of the release"`
	V8              string `ox:"v8 version of the release,name:v8"`
	PrevChromium    string `ox:"chromium version of the previous release"`
	PrevV8          string `ox:"v8 version of the previous release,name:prev-v8"`
	Date            string `ox:"date of the release - YYYY-MM-DD"`
	AddedPackages   string `ox:"space separated packages that the release adds"`
	RemovedPackages string `ox:"space separated packages that the release removes"`
}

// New creates the relnotes command arguments.
func New() *Args {
	return new(Args)
}

// Run returns the func that runs the relnotes command.
func (args *Args) Run(stdout io.Writer) func(context.Context, []string) error {
	return func(_ context.Context, cliargs []string) error {
		if len(cliargs) != 0 {
			return fmt.Errorf("unexpected arguments: %s", strings.Join(cliargs, " "))
		}
		return args.exec(stdout)
	}
}

// exec writes the notes.
func (args *Args) exec(stdout io.Writer) error {
	rep := new(Report)
	if args.Diff != "" {
		f, err := os.Open(args.Diff)
		if err != nil {
			return fmt.Errorf("opening %s: %w", args.Diff, err)
		}
		defer f.Close()
		if rep, err = Parse(f); err != nil {
			return fmt.Errorf("parsing %s: %w", args.Diff, err)
		}
	}
	m := Meta{
		Version:         args.Version,
		Previous:        args.Previous,
		Chromium:        args.Chromium,
		V8:              args.V8,
		PrevChromium:    args.PrevChromium,
		PrevV8:          args.PrevV8,
		Date:            args.Date,
		AddedPackages:   strings.Fields(args.AddedPackages),
		RemovedPackages: strings.Fields(args.RemovedPackages),
	}
	var s string
	switch args.Mode {
	case "commit":
		s = rep.Commit(m)
	case "tag":
		s = rep.Tag(m)
	case "changelog":
		s = rep.Changelog(m)
	default:
		return fmt.Errorf("unknown mode %q", args.Mode)
	}
	_, err := io.WriteString(stdout, s)
	return err
}
