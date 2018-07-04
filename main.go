// cdproto-gen is a tool to generate the low-level Chrome Debugging Protocol
// implementation types used by chromedp, based off Chrome's protocol
// definitions.
//
// Please see README.md for more information on using this tool.
package main

//go:generate qtc -dir gen/gotpl -ext qtpl
//go:generate gofmt -w -s gen/gotpl/

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
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
)

const (
	chromiumSrc = "https://chromium.googlesource.com"
	browserURL  = "chromium/src/+/%s/third_party/blink/renderer/core/inspector/browser_protocol.pdl"
	jsURL       = "v8/v8/+/%s/src/inspector/js_protocol.pdl"
	easyjsonGo  = "easyjson.go"
)

var (
	flagDebug = flag.Bool("debug", false, "toggle debug (writes generated files to disk without post-processing)")

	flagPdl     = flag.String("pdl", "", "path to pdl file to use")
	flagBrowser = flag.String("browser", "master", "browser version to retrieve/use")
	flagJS      = flag.String("js", "master", "js version to retrieve/use")
	flagTTL     = flag.Duration("ttl", 24*time.Hour, "browser and js cache ttl")

	flagCache = flag.String("cache", filepath.Join(os.Getenv("GOPATH"), "pkg", "cdproto-gen"), "protocol cache directory")
	flagOut   = flag.String("out", "", "out directory")

	flagNoClean = flag.Bool("no-clean", false, "toggle not cleaning (removing) existing directories")
	flagNoDump  = flag.Bool("no-dump", false, "toggle not dumping generated protocol file to out directory")

	flagGoPkg = flag.String("go-pkg", "github.com/chromedp/cdproto", "go base package name")
	flagGoWl  = flag.String("go-wl", "LICENSE,README.md,protocol*.pdl,"+easyjsonGo, "comma-separated list of files to whitelist (ignore)")

	flagWorkers = flag.Int("workers", runtime.NumCPU(), "number of workers")
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
	}

	// create out directory
	err = os.MkdirAll(*flagOut, 0755)
	if err != nil {
		return err
	}

	protoFile := filepath.Join(*flagOut, fmt.Sprintf("protocol-%s_%s-%s.pdl", *flagBrowser, *flagJS, time.Now().Format("20060102")))

	// write protocol definitions
	if *flagPdl == "" {
		logf("WRITING: %s", protoFile)
		if err = ioutil.WriteFile(protoFile, protoDefs.Bytes(), 0644); err != nil {
			return err
		}

		// display differences between generated definitions and previous version on disk
		if runtime.GOOS != "windows" {
			diffBuf, err := diff.WalkAndCompare(*flagOut, `^protocol-([^-]+)-(2[0-9]{7})\.pdl$`, 2, protoFile)
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
			if d.Deprecated {
				extra = append(extra, "deprecated")
			}
			logf("SKIPPING(%s): %s %v", pad("domain", 7), d.Domain.String(), extra)
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
		logf("CLEANING: %s", *flagOut)
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

			logf("REMOVING: %s", n)
			return os.RemoveAll(n)
		})
		if err != nil {
			return err
		}
	}

	logf("WRITING: %d files", len(files))

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

	logf("done.")
	return nil
}

// loadProtoDefs loads the protocol definitions either from the path specified
// in -proto or by retrieving the versions specified in the -browser and -js
// files.
func loadProtoDefs() (*pdl.PDL, error) {
	var err error

	if *flagPdl != "" {
		logf("PROTOCOL: %s", *flagPdl)
		buf, err := ioutil.ReadFile(*flagPdl)
		if err != nil {
			return nil, err
		}

		return pdl.Parse(buf)
	}

	var protoDefs []*pdl.PDL
	load := func(typ, file, ver string) error {
		urlstr := fmt.Sprintf("%s/%s?format=TEXT", chromiumSrc, fmt.Sprintf(file, ver))
		logf("%s: %s", pad(strings.ToUpper(typ), 7), urlstr)
		buf, err := fileCacher{
			path: filepath.Join(*flagCache, typ, ver),
			ttl:  *flagTTL,
		}.Get(urlstr, true, path.Base(file))
		if err != nil {
			return err
		}

		// convert PDL to JSON definition
		protoDef, err := pdl.Parse(buf)
		if err != nil {
			return err
		}

		protoDefs = append(protoDefs, protoDef)
		return nil
	}

	// grab browser definition
	err = load("browser", browserURL, *flagBrowser)
	if err != nil {
		return nil, err
	}

	// grab js definition
	err = load("js", jsURL, *flagJS)
	if err != nil {
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
			logf("SKIPPING(%s): %s [deprecated]", pad(n, 7), typ)
			continue
		}

		if t.Redirect != nil {
			logf("SKIPPING(%s): %s [redirect:%s]", pad(n, 7), typ, t.Redirect)
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
	logf("RUNNING: goimports")

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
	logf("RUNNING: easyjson")
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
					OutName:  filepath.Join(n, easyjsonGo),
					PkgPath:  p.PkgPath,
					PkgName:  p.PkgName,
					Types:    p.StructNames,
					NoFormat: true,
				}
				return g.Run()
			}
		}(k))
	}
	return eg.Wait()
}

// gofmt go formats all files on disk.
func gofmt(files []string) error {
	logf("RUNNING: gofmt")
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

// fileCacher handles caching files to a path with a ttl.
type fileCacher struct {
	path string
	ttl  time.Duration
}

// Load attempts to load the file from disk, disregarding ttl.
func (fc fileCacher) Load(names ...string) ([]byte, error) {
	return ioutil.ReadFile(pathJoin(fc.path, names...))
}

// Cache writes buf to the fileCacher path joined with names.
func (fc fileCacher) Cache(buf []byte, names ...string) error {
	logf("WRITING: %s", pathJoin(fc.path, names...))
	return ioutil.WriteFile(pathJoin(fc.path, names...), buf, 0644)
}

// Get retrieves a file from disk or from the remote URL, optionally
// base64 decoding it and writing it to disk.
func (fc fileCacher) Get(urlstr string, b64Decode bool, names ...string) ([]byte, error) {
	n := pathJoin(fc.path, names...)
	cd := filepath.Dir(n)
	err := os.MkdirAll(cd, 0755)
	if err != nil {
		return nil, err
	}

	// check if exists on disk
	fi, err := os.Stat(n)
	if err == nil && fc.ttl != 0 && !time.Now().After(fi.ModTime().Add(fc.ttl)) {
		// logf("LOADING: %s", n)
		return ioutil.ReadFile(n)
	}

	logf("RETRIEVING: %s", urlstr)

	// retrieve
	cl := &http.Client{}
	req, err := http.NewRequest("GET", urlstr, nil)
	if err != nil {
		return nil, err
	}
	res, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// decode
	if b64Decode {
		buf, err = base64.StdEncoding.DecodeString(string(buf))
		if err != nil {
			return nil, err
		}
	}

	// write
	err = fc.Cache(buf, names...)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// pathJoin is a simple wrapper around filepath.Join to simplify inline syntax.
func pathJoin(n string, m ...string) string {
	return filepath.Join(append([]string{n}, m...)...)
}

// logf is a wrapper around log.Printf.
func logf(s string, v ...interface{}) {
	log.Printf(s, v...)
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
	return s + strings.Repeat(" ", n-len(s))
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
