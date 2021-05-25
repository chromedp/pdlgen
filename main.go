// cdproto-gen is a tool to generate the low-level Chrome DevTools Protocol
// implementation types used by chromedp from the CDP protocol definitions
// (PDLs) in the Chromium source tree.
//
// Please see README.md for more information on using this tool.
package main

//go:generate qtc -dir gen/gotpl -ext qtpl
//go:generate gofmt -w -s gen/gotpl/

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/mailru/easyjson/bootstrap"
	"github.com/mailru/easyjson/parser"
	glob "github.com/ryanuber/go-glob"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/imports"

	"github.com/chromedp/cdproto-gen/diff"
	"github.com/chromedp/cdproto-gen/fixup"
	"github.com/chromedp/cdproto-gen/gen"
	"github.com/chromedp/cdproto-gen/gen/genutil"
	"github.com/chromedp/cdproto-gen/pdl"
	"github.com/chromedp/cdproto-gen/util"
)

const (
	easyjsonGo = "easyjson.go"
)

var (
	flagDebug = flag.Bool("debug", false, "toggle debug (writes generated files to disk without post-processing)")

	flagTTL = flag.Duration("ttl", 24*time.Hour, "file retrieval caching ttl")

	flagChromium = flag.String("chromium", "", "chromium protocol version")
	flagV8       = flag.String("v8", "", "v8 protocol version")
	flagLatest   = flag.Bool("latest", false, "use latest protocol")

	flagPdl = flag.String("pdl", "", "path to pdl file to use")

	flagCache = flag.String("cache", "", "protocol cache directory")
	flagOut   = flag.String("out", "", "package out directory")

	flagNoClean = flag.Bool("no-clean", false, "toggle not cleaning (removing) existing directories")
	flagNoDump  = flag.Bool("no-dump", false, "toggle not dumping generated protocol file to out directory")

	flagGoPkg = flag.String("go-pkg", "github.com/chromedp/cdproto", "go base package name")
	flagGoWl  = flag.String("go-wl", "LICENSE,README.md,*.pdl,go.mod,go.sum", "comma-separated list of files to whitelist (ignore)")

	// flagWorkers = flag.Int("workers", runtime.NumCPU(), "number of workers")
)

func main() {
	// add generator parameters
	var genTypes []string
	generators := gen.Generators()
	for n, g := range generators {
		genTypes = append(genTypes, n)
		g = g
	}

	flag.Parse()

	// run
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run runs the generator.
func run() error {
	var err error

	// set cache path
	if *flagCache == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		*flagCache = filepath.Join(cacheDir, "cdproto-gen")
	}

	// get latest versions
	if *flagChromium == "" {
		if *flagChromium, err = util.GetLatestVersion(util.Cache{
			URL:  util.ChromiumBase,
			Path: filepath.Join(*flagCache, "html", "chromium.html"),
			TTL:  *flagTTL,
		}); err != nil {
			return err
		}
	}
	if *flagV8 == "" {
		if *flagLatest {
			if *flagV8, err = util.GetLatestVersion(util.Cache{
				URL:  util.V8Base,
				Path: filepath.Join(*flagCache, "html", "v8.html"),
				TTL:  *flagTTL,
			}); err != nil {
				return err
			}
		} else {
			if *flagV8, err = util.GetDepVersion("v8", *flagChromium, util.Cache{
				URL:    fmt.Sprintf(util.ChromiumDeps+"?format=TEXT", *flagChromium),
				Path:   filepath.Join(*flagCache, "deps", "chromium", *flagChromium),
				TTL:    *flagTTL,
				Decode: true,
			}, util.Cache{
				URL:  util.V8Base + "/+refs?format=JSON",
				Path: filepath.Join(*flagCache, "refs", "v8.json"),
				TTL:  *flagTTL,
			}); err != nil {
				return err
			}
		}
	}

	// load protocol definitions
	protoDefs, err := loadProtoDefs()
	if err != nil {
		return err
	}
	sort.Slice(protoDefs.Domains, func(i, j int) bool {
		return strings.Compare(protoDefs.Domains[i].Domain.String(), protoDefs.Domains[j].Domain.String()) <= 0
	})

	if *flagOut == "" {
		*flagOut = filepath.Join(os.Getenv("GOPATH"), "src", *flagGoPkg)
	} else {
		*flagOut, err = filepath.Abs(*flagOut)
		if err != nil {
			return err
		}
	}

	// create out directory
	if err = os.MkdirAll(*flagOut, 0755); err != nil {
		return err
	}

	combinedDir := filepath.Join(*flagCache, "pdl", "combined")
	if err = os.MkdirAll(combinedDir, 0755); err != nil {
		return err
	}
	protoFile := filepath.Join(combinedDir, fmt.Sprintf("%s_%s.pdl", *flagChromium, *flagV8))

	// write protocol definitions
	if *flagPdl == "" {
		util.Logf("WRITING: %s", protoFile)
		if err = ioutil.WriteFile(protoFile, protoDefs.Bytes(), 0644); err != nil {
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
				os.Stdout.Write(diffBuf)
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

		// TODO: remove this pre-cleanup fixup at some point; right now,
		// it's necessary as the current Chrome stable release doesn't
		// yet support the new Browser.setDownloadBehavior.
		switch d.Domain {
		case "Page":
			for _, c := range d.Commands {
				switch c.Name {
				case "setDownloadBehavior":
					c.AlwaysEmit = true
				case "getLayoutMetrics":
					for _, t := range c.Returns {
						t.AlwaysEmit = true
					}
				}
			}
		}

		// will process
		pkgs = append(pkgs, genutil.PackageName(d))
		processed = append(processed, d)

		// cleanup types, events, commands
		d.Types = cleanupTypes("type", d.Domain.String(), d.Types)
		d.Events = cleanupTypes("event", d.Domain.String(), d.Events)
		d.Commands = cleanupTypes("command", d.Domain.String(), d.Commands)
	}

	// fixup
	fixup.FixDomains(processed)

	// get generator
	generator := gen.Generators()["go"]
	if generator == nil {
		return errors.New("no generator")
	}

	// emit
	emitter, err := generator(processed, *flagGoPkg)
	if err != nil {
		return err
	}
	files := emitter.Emit()

	// clean up files
	if !*flagNoClean {
		util.Logf("CLEANING: %s", *flagOut)
		outpath := *flagOut + string(filepath.Separator)
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
			if pn == "" || strings.HasPrefix(pn, ".") || strings.HasPrefix(fn, ".") || whitelisted(fn) || contains(files, pn) {
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
	if *flagDebug {
		return write(files)
	}

	// goimports (also writes to disk)
	if err = goimports(files); err != nil {
		return err
	}

	// easyjson
	if err = easyjson(pkgs); err != nil {
		return err
	}

	// gofmt
	if err = gofmt(fmtFiles(files, pkgs)); err != nil {
		return err
	}

	util.Logf("done.")
	return nil
}

// loadProtoDefs loads the protocol definitions either from the path specified
// in -proto or by retrieving the versions specified in the -browser and -js
// files.
func loadProtoDefs() (*pdl.PDL, error) {
	var err error

	if *flagPdl != "" {
		util.Logf("PROTOCOL: %s", *flagPdl)
		buf, err := ioutil.ReadFile(*flagPdl)
		if err != nil {
			return nil, err
		}
		return pdl.Parse(buf)
	}

	var protoDefs []*pdl.PDL
	load := func(urlstr, typ, ver string) error {
		buf, err := util.Get(util.Cache{
			URL:    fmt.Sprintf(urlstr+"?format=TEXT", ver),
			Path:   filepath.Join(*flagCache, "pdl", typ, ver+".pdl"),
			TTL:    *flagTTL,
			Decode: true,
		})
		if err != nil {
			return err
		}

		// parse
		protoDef, err := pdl.Parse(buf)
		if err != nil {
			return err
		}
		protoDefs = append(protoDefs, protoDef)
		return nil
	}

	// grab browser + js definition
	if err = load(util.ChromiumURL, "chromium", *flagChromium); err != nil {
		return nil, err
	}
	if err = load(util.V8URL, "v8", *flagV8); err != nil {
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
		if t.Deprecated && !t.AlwaysEmit {
			util.Logf("SKIPPING(%s): %s [deprecated]", pad(n, 7), typ)
			continue
		}

		if t.Redirect != nil && !t.AlwaysEmit {
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
func write(fileBuffers map[string]*bytes.Buffer) error {
	var keys []string
	for k := range fileBuffers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// add out path
		n := filepath.Join(*flagOut, k)

		// create directory
		if err := os.MkdirAll(filepath.Dir(n), 0755); err != nil {
			return err
		}

		// write file
		if err := ioutil.WriteFile(n, fileBuffers[k].Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

// goimports formats all the output file buffers on disk using goimports.
func goimports(fileBuffers map[string]*bytes.Buffer) error {
	util.Logf("RUNNING: goimports")

	var keys []string
	for k := range fileBuffers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	eg, _ := errgroup.WithContext(context.Background())
	for _, k := range keys {
		eg.Go(func(n string) func() error {
			return func() error {
				fn := filepath.Join(*flagOut, n)
				buf, err := imports.Process(fn, fileBuffers[n].Bytes(), nil)
				if err != nil {
					return err
				}
				if err = os.MkdirAll(filepath.Dir(fn), 0755); err != nil {
					return err
				}
				return ioutil.WriteFile(fn, buf, 0644)
			}
		}(k))
	}
	return eg.Wait()
}

// easyjson runs easy json on the list of packages.
func easyjson(pkgs []string) error {
	util.Logf("WRITING: easyjson stubs")
	// All the easyjson.go files are removed in the CLEANING step,
	// so that deprecated files (if any) won't stay in the repository.
	// Now generate the stubs first so that the source codes are valid
	// from the perspective of syntax.
	if err := easyjsonStubs(pkgs); err != nil {
		return err
	}

	util.Logf("RUNNING: easyjson")
	// Got error messages like this when running g.Run() concurrently:
	//   # github.com/chromedp/cdproto/cachestorage
	//   cachestorage/easyjson.go:8:3: can't find import: "encoding/json"
	//   # github.com/chromedp/cdproto/cast
	//   cast/easyjson.go:6:3: can't find import: "github.com/mailru/easyjson"
	// It seems that it fails more often on slow machines. The root cause is not clear yet,
	// maybe it's relevant to the issue https://github.com/golang/go/issues/26794.
	// The workaround for now is to run g.Run() one by one (take longer to finish).
	for _, n := range pkgs {
		n = filepath.Join(*flagOut, n)
		p := parser.Parser{AllStructs: true}
		if err := p.Parse(n, true); err != nil {
			return err
		}
		g := bootstrap.Generator{
			OutName:  filepath.Join(n, easyjsonGo),
			PkgPath:  p.PkgPath,
			PkgName:  p.PkgName,
			Types:    p.StructNames,
			NoFormat: true,
		}
		if err := g.Run(); err != nil {
			return err
		}
	}
	return nil
}

// easyjsonStubs runs easy json to generate stubs for the list of packages.
func easyjsonStubs(pkgs []string) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, k := range pkgs {
		eg.Go(func(n string) func() error {
			return func() error {
				n = filepath.Join(*flagOut, n)
				p := parser.Parser{AllStructs: true}
				if err := p.Parse(n, true); err != nil {
					return err
				}
				g := bootstrap.Generator{
					OutName:   filepath.Join(n, easyjsonGo),
					PkgPath:   p.PkgPath,
					PkgName:   p.PkgName,
					Types:     p.StructNames,
					NoFormat:  true,
					StubsOnly: true,
				}
				return g.Run()
			}
		}(k))
	}
	return eg.Wait()
}

// gofmt go formats all files on disk.
func gofmt(files []string) error {
	util.Logf("RUNNING: gofmt")
	eg, _ := errgroup.WithContext(context.Background())
	for _, k := range files {
		eg.Go(func(n string) func() error {
			return func() error {
				n = filepath.Join(*flagOut, n)
				in, err := ioutil.ReadFile(n)
				if err != nil {
					return err
				}
				out, err := format.Source(in)
				if err != nil {
					return err
				}
				return ioutil.WriteFile(n, out, 0644)
			}
		}(k))
	}
	return eg.Wait()
}

// fmtFiles returns the list of all files to format from the specified file
// buffers and packages.
func fmtFiles(files map[string]*bytes.Buffer, pkgs []string) []string {
	filelen := len(files)
	f := make([]string, filelen+len(pkgs))

	var i int
	for n := range files {
		f[i] = n
		i++
	}

	for i, pkg := range pkgs {
		f[i+filelen] = filepath.Join(pkg, easyjsonGo)
	}

	sort.Strings(f)
	return f
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
func whitelisted(n string) bool {
	for _, z := range strings.Split(*flagGoWl, ",") {
		if z == n || glob.Glob(z, n) {
			return true
		}
	}
	return false
}
