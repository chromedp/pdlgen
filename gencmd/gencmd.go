// Package gencmd is the pdlgen command: it generates the Go package
// cdproto from the Chrome DevTools Protocol definitions (PDLs) in the Chromium
// and V8 source trees.
//
// Please see README.md for more information on using the command.
package gencmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	glob "github.com/ryanuber/go-glob"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/imports"

	"github.com/chromedp/pdlgen/diff"
	"github.com/chromedp/pdlgen/fixup"
	"github.com/chromedp/pdlgen/gen"
	"github.com/chromedp/pdlgen/gen/genutil"
	"github.com/chromedp/pdlgen/pdl"
	"github.com/chromedp/pdlgen/util"
)

// DefaultWhitelist is the default list of files that the clean step keeps.
const DefaultWhitelist = "LICENSE,README.md,CHANGELOG.md,*.pdl,go.mod,go.sum"

// Args are the pdlgen command arguments.
type Args struct {
	Chromium string `ox:"chromium protocol version - default is the latest"`
	V8       string `ox:"v8 protocol version - default is the version in the chromium DEPS,name:v8"`
	Latest   bool   `ox:"use the latest v8 protocol"`
	PDL      string `ox:"path to a combined pdl file to use,name:pdl"`
	Cache    string `ox:"protocol cache directory"`
	TTL      string `ox:"file retrieval caching ttl - a duration such as 24h or 0,name:ttl"`
	Out      string `ox:"package out directory,short:o"`
	NoClean  bool   `ox:"do not remove files from the out directory that were not generated"`
	GoPkg    string `ox:"go base package name,name:go-pkg"`
	GoWl     string `ox:"comma-separated list of files to keep in the out directory,name:go-wl"`
	Debug    bool   `ox:"write the generated files without goimports and gofmt"`

	// ttl is TTL as a duration.
	ttl time.Duration
}

// New creates the pdlgen command arguments.
func New() *Args {
	return &Args{
		TTL:   "24h",
		GoPkg: "github.com/chromedp/cdproto",
		GoWl:  DefaultWhitelist,
	}
}

// Run returns the func that runs the pdlgen command.
func (args *Args) Run(stdout io.Writer) func(context.Context, []string) error {
	return func(ctx context.Context, cliargs []string) error {
		return args.Exec(ctx, stdout, cliargs)
	}
}

// Exec runs the pdlgen command.
func (args *Args) Exec(ctx context.Context, stdout io.Writer, cliargs []string) error {
	if len(cliargs) != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(cliargs, " "))
	}
	ttl, err := time.ParseDuration(args.TTL)
	if err != nil {
		return fmt.Errorf("invalid ttl %q: %w", args.TTL, err)
	}
	args.ttl = ttl
	return args.run(ctx, stdout)
}

// run runs the generator.
func (args *Args) run(ctx context.Context, stdout io.Writer) error {
	var err error

	// set cache path
	if args.Cache == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		args.Cache = filepath.Join(cacheDir, "pdlgen")
	}

	// get latest versions
	if args.Chromium == "" {
		if args.Chromium, err = util.GetLatestVersion(util.Cache{
			URL:  util.ChromiumBase,
			Path: filepath.Join(args.Cache, "html", "chromium.html"),
			TTL:  args.ttl,
		}); err != nil {
			return err
		}
	}
	if args.V8 == "" {
		if args.Latest {
			if args.V8, err = util.GetLatestVersion(util.Cache{
				URL:  util.V8Base,
				Path: filepath.Join(args.Cache, "html", "v8.html"),
				TTL:  args.ttl,
			}); err != nil {
				return err
			}
		} else {
			if args.V8, err = util.GetDepVersion("v8", args.Chromium, util.Cache{
				URL:    fmt.Sprintf(util.ChromiumDeps+"?format=TEXT", args.Chromium),
				Path:   filepath.Join(args.Cache, "deps", "chromium", args.Chromium),
				TTL:    args.ttl,
				Decode: true,
			}, util.Cache{
				URL:  util.V8Base + "/+refs?format=JSON",
				Path: filepath.Join(args.Cache, "refs", "v8.json"),
				TTL:  args.ttl,
			}); err != nil {
				return err
			}
		}
	}

	// load protocol definitions
	protoDefs, err := args.loadProtoDefs()
	if err != nil {
		return err
	}
	sort.Slice(protoDefs.Domains, func(i, j int) bool {
		return strings.Compare(protoDefs.Domains[i].Domain.String(), protoDefs.Domains[j].Domain.String()) <= 0
	})

	if args.Out == "" {
		args.Out = filepath.Join(os.Getenv("GOPATH"), "src", args.GoPkg)
	} else {
		args.Out, err = filepath.Abs(args.Out)
		if err != nil {
			return err
		}
	}

	// create out directory
	if err = os.MkdirAll(args.Out, 0o755); err != nil {
		return err
	}

	combinedDir := filepath.Join(args.Cache, "pdl", "combined")
	if err = os.MkdirAll(combinedDir, 0o755); err != nil {
		return err
	}
	protoFile := filepath.Join(combinedDir, fmt.Sprintf("%s_%s.pdl", args.Chromium, args.V8))

	// write protocol definitions
	if args.PDL == "" {
		util.Logf("WRITING: %s", protoFile)
		if err = os.WriteFile(protoFile, protoDefs.Bytes(), 0o644); err != nil {
			return err
		}

		// display differences between generated definitions and previous version on disk
		if runtime.GOOS != "windows" {
			diffBuf, err := diff.WalkAndCompare(combinedDir, `^([0-9_.]+)\.pdl$`, protoFile, func(a, b *diff.FileInfo) bool {
				n := strings.Split(strings.TrimSuffix(filepath.Base(a.Name), ".pdl"), "_")
				m := strings.Split(strings.TrimSuffix(filepath.Base(b.Name), ".pdl"), "_")
				if n[0] == m[0] {
					return util.CompareSemver(n[1], m[1])
				}
				return util.CompareSemver(n[0], m[0])
			})
			if err != nil {
				return err
			}
			if diffBuf != nil {
				if _, err := stdout.Write(diffBuf); err != nil {
					return err
				}
			}
		}
	}

	// determine what to process
	pkgs := []string{"", "cdp"}
	var processed []*pdl.Domain
	for _, d := range protoDefs.Domains {
		// skip if not processing
		if d.Deprecated {
			var extra []string
			extra = append(extra, "deprecated")
			util.Logf("SKIPPING(%s): %s %v", pad("domain", 7), d.Domain.String(), extra)
			continue
		}

		// will process
		pkgs = append(pkgs, genutil.PackageName(d))
		processed = append(processed, d)

		// cleanup types, events, commands
		d.Types = cleanupTypes("type", d.Domain.String(), d.Types)
		d.Events = cleanupTypes("event", d.Domain.String(), d.Events)
		d.Commands = cleanupTypes("command", d.Domain.String(), d.Commands)
	}

	// commands and events without parameters are empty structs
	for _, d := range processed {
		for _, typs := range [][]*pdl.Type{d.Events, d.Commands} {
			for _, t := range typs {
				if t.Parameters == nil {
					t.Parameters = []*pdl.Type{}
				}
			}
		}
	}

	// remove name stuttering
	fixup.FixDomains(processed)

	// get generator
	generator := gen.Generators()["go"]
	if generator == nil {
		return errors.New("no generator")
	}

	// emit
	emitter, err := generator(processed, args.GoPkg, gen.Versions{Chromium: args.Chromium, V8: args.V8})
	if err != nil {
		return err
	}
	files := emitter.Emit()

	// clean up files
	if !args.NoClean {
		util.Logf("CLEANING: %s", args.Out)
		outpath := args.Out + string(filepath.Separator)
		err = filepath.Walk(outpath, func(n string, fi os.FileInfo, err error) error {
			switch {
			case os.IsNotExist(err) || n == outpath:
				return nil
			case err != nil:
				return err
			}

			// skip if file or path starts with ., is whitelisted, or is one of
			// the files whose output will be overwritten
			pn, fn := n[len(outpath):], fi.Name()
			if pn == "" || strings.HasPrefix(pn, ".") || strings.HasPrefix(fn, ".") || args.whitelisted(fn) || contains(files, pn) {
				return nil
			}

			util.Logf("REMOVING: %s", n)
			return os.RemoveAll(n)
		})
		if err != nil {
			return err
		}
	}

	util.Logf("WRITING: %d files", len(files))

	// dump files and exit
	if args.Debug {
		return args.write(files)
	}

	// goimports (also writes to disk)
	if err = args.goimports(ctx, files); err != nil {
		return err
	}

	filenames := slices.Sorted(maps.Keys(files))

	// gofmt
	if err = args.gofmt(ctx, filenames); err != nil {
		return err
	}

	util.Logf("done.")
	return nil
}

// loadProtoDefs loads the protocol definitions either from the path specified
// in -proto or by retrieving the versions specified in the -browser and -js
// files.
func (args *Args) loadProtoDefs() (*pdl.PDL, error) {
	var err error

	if args.PDL != "" {
		util.Logf("PROTOCOL: %s", args.PDL)
		buf, err := os.ReadFile(args.PDL)
		if err != nil {
			return nil, err
		}
		return pdl.Parse(buf)
	}

	var protoDefs []*pdl.PDL
	load := func(urlstr, typ, ver string) error {
		base := path.Base(urlstr)
		if !strings.HasSuffix(base, ".pdl") {
			return fmt.Errorf("invalid url %q", urlstr)
		}
		name := strings.TrimSuffix(base, ".pdl")
		buf, err := util.Get(util.Cache{
			URL:    fmt.Sprintf(urlstr+"?format=TEXT", ver),
			Path:   filepath.Join(args.Cache, "pdl", typ, name+"_"+ver+".pdl"),
			TTL:    args.ttl,
			Decode: true,
		})
		if err != nil {
			return err
		}

		// add grab
		p := pdl.NewParser(pdl.Grab(func(s string) ([]byte, error) {
			u := strings.TrimSuffix(urlstr, base) + s
			n := strings.TrimSuffix(path.Base(u), ".pdl")
			return util.Get(util.Cache{
				URL:    fmt.Sprintf(u+"?format=TEXT", ver),
				Path:   filepath.Join(args.Cache, "pdl", typ, name+"_"+n+"_"+ver+".pdl"),
				TTL:    args.ttl,
				Decode: true,
			})
		}))

		// parse
		protoDef, err := p.Parse(buf)
		if err != nil {
			return err
		}
		protoDefs = append(protoDefs, protoDef)
		return nil
	}

	// grab browser + js definition
	if err = load(util.ChromiumURL, "chromium", args.Chromium); err != nil {
		return nil, err
	}
	if err = load(util.V8URL, "v8", args.V8); err != nil {
		return nil, err
	}

	// grab har definition
	har, err := pdl.Parse([]byte(pdl.HAR))
	if err != nil {
		return nil, err
	}

	return pdl.Combine(append(protoDefs, har)...), nil
}

// cleanupTypes removes deprecated and redirected types.
func cleanupTypes(n string, dtyp string, typs []*pdl.Type) []*pdl.Type {
	var ret []*pdl.Type

	for _, t := range typs {
		typ := dtyp + "." + t.Name
		if t.Deprecated {
			util.Logf("SKIPPING(%s): %s [deprecated]", pad(n, 7), typ)
			continue
		}

		if t.Redirect != nil {
			util.Logf("SKIPPING(%s): %s [redirect:%s]", pad(n, 7), typ, t.Redirect)
			continue
		}

		if t.Properties != nil {
			t.Properties = cleanupTypes(n[0:1]+" property", typ, t.Properties)
		}

		if t.Parameters != nil {
			t.Parameters = cleanupTypes(n[0:1]+" param", typ, t.Parameters)
		}

		if t.Returns != nil {
			t.Returns = cleanupTypes(n[0:1]+" return param", typ, t.Returns)
		}

		ret = append(ret, t)
	}

	return ret
}

// write writes all file buffer to disk.
func (args *Args) write(fileBuffers map[string]*bytes.Buffer) error {
	var keys []string
	for k := range fileBuffers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// add out path
		n := filepath.Join(args.Out, k)

		// create directory
		if err := os.MkdirAll(filepath.Dir(n), 0o755); err != nil {
			return err
		}

		// write file
		if err := os.WriteFile(n, fileBuffers[k].Bytes(), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// goimports formats all the output file buffers on disk using goimports.
func (args *Args) goimports(ctx context.Context, fileBuffers map[string]*bytes.Buffer) error {
	util.Logf("RUNNING: goimports")

	var keys []string
	for k := range fileBuffers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	eg, _ := errgroup.WithContext(ctx)
	for _, k := range keys {
		eg.Go(func(n string) func() error {
			return func() error {
				fn := filepath.Join(args.Out, n)
				buf, err := imports.Process(fn, fileBuffers[n].Bytes(), nil)
				if err != nil {
					return err
				}
				if err = os.MkdirAll(filepath.Dir(fn), 0o755); err != nil {
					return err
				}
				return os.WriteFile(fn, buf, 0o644)
			}
		}(k))
	}
	return eg.Wait()
}

// gofmt go formats all files on disk.
func (args *Args) gofmt(ctx context.Context, files []string) error {
	util.Logf("RUNNING: gofmt")
	eg, _ := errgroup.WithContext(ctx)
	for _, k := range files {
		eg.Go(func(n string) func() error {
			return func() error {
				n = filepath.Join(args.Out, n)
				in, err := os.ReadFile(n)
				if err != nil {
					return err
				}
				out, err := format.Source(in)
				if err != nil {
					return err
				}
				return os.WriteFile(n, out, 0o644)
			}
		}(k))
	}
	return eg.Wait()
}

// contains determines if any key in m is equal to n or starts with the path
// prefix equal to n.
func contains(m map[string]*bytes.Buffer, n string) bool {
	d := n + string(filepath.Separator)
	for k := range m {
		if n == k || strings.HasPrefix(k, d) {
			return true
		}
	}
	return false
}

// pad pads a string.
func pad(s string, n int) string {
	n = n - len(s)
	if n < 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}

// whitelisted checks if n is a whitelisted file.
func (args *Args) whitelisted(n string) bool {
	for z := range strings.SplitSeq(args.GoWl, ",") {
		if z == n || glob.Glob(z, n) {
			return true
		}
	}
	return false
}
