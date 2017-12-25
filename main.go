// chromedp-gen is a tool to generate the low-level Chrome Debugging Protocol
// implementation types used by chromedp, based off Chrome's protocol.json.
//
// Please see README.md for more information on using this tool.
package main

//go:generate qtc -dir templates -ext qtpl
//go:generate gofmt -w -s templates/

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/chromedp/chromedp-gen/fixup"
	"github.com/chromedp/chromedp-gen/gen"
	"github.com/chromedp/chromedp-gen/har"
	"github.com/chromedp/chromedp-gen/types"
)

const (
	chromiumSrc = "https://chromium.googlesource.com/"
	browserURL  = chromiumSrc + "chromium/src/+/%s/third_party/WebKit/Source/core/inspector/browser_protocol.json?format=TEXT"
	jsURL       = chromiumSrc + "v8/v8/+/%s/src/inspector/js_protocol.json?format=TEXT"
)

var (
	flagVerbose = flag.Bool("v", true, "toggle verbose")
	flagWorkers = flag.Int("workers", runtime.NumCPU()+1, "number of workers")

	flagProto   = flag.String("proto", "", "protocol.json path")
	flagBrowser = flag.String("browser", "master", "browser protocol version to use")
	flagJS      = flag.String("js", "master", "js protocol version to use")
	flagTTL     = flag.Duration("ttl", 24*time.Hour, "browser and js protocol cache ttl")
	flagTTLHar  = flag.Duration("ttlHar", 0, "har cache ttl")

	flagCache = flag.String("cache", filepath.Join(os.Getenv("GOPATH"), "pkg", "chromedp-gen"), "protocol cache directory")
	flagPkg   = flag.String("pkg", "github.com/chromedp/cdproto", "out base package")
	flagOut   = flag.String("out", "", "out directory")

	flagNoClean = flag.Bool("noclean", false, "toggle not cleaning (removing) existing directories")
	flagNoCopy  = flag.Bool("nocopy", false, "toggle not copying combined protocol.json to out directory")
	flagNoHar   = flag.Bool("nohar", false, "toggle not generating HAR protocol and domain")
	flagCleanWl = flag.String("wl", "LICENSE,README.md,protocol.json,easyjson.go", "comma-separated list of files to whitelist/ignore during clean")

	flagDep      = flag.Bool("dep", false, "toggle generation for deprecated APIs")
	flagExp      = flag.Bool("exp", true, "toggle generation for experimental APIs")
	flagRedirect = flag.Bool("redirect", false, "toggle generation for redirect APIs")
)

func main() {
	var err error

	flag.Parse()

	// fix out directory
	if *flagOut == "" {
		*flagOut = filepath.Join(os.Getenv("GOPATH"), "src", *flagPkg)
	}

	err = run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run runs the generator.
func run() error {
	// load protocol definitions
	protoInfo, err := loadProtocolInfo()
	if err != nil {
		return err
	}
	sort.Slice(protoInfo.Domains, func(i, j int) bool {
		return strings.Compare(protoInfo.Domains[i].String(), protoInfo.Domains[j].String()) <= 0
	})

	// create out directory
	err = os.MkdirAll(*flagOut, 0755)
	if err != nil {
		return err
	}

	// determine what to process
	pkgs := []string{"", "cdp"}
	var processed []*types.Domain
	for _, d := range protoInfo.Domains {
		// skip if not processing
		if (!*flagDep && d.Deprecated.Bool()) || (!*flagExp && d.Experimental.Bool()) {
			// extra info
			var extra []string
			if d.Deprecated.Bool() {
				extra = append(extra, "deprecated")
			}
			if d.Experimental.Bool() {
				extra = append(extra, "experimental")
			}

			logf("SKIPPING(%s): %s %v", pad("domain", 7), d, extra)
			continue
		}

		// will process
		pkgs = append(pkgs, d.PackageName())
		processed = append(processed, d)

		// cleanup types, events, commands
		cleanup(d)
	}

	// fixup
	fixup.FixDomains(processed)

	// generate
	files := gen.GenerateDomains(processed, *flagPkg, *flagRedirect)

	// clean up files
	if !*flagNoClean {
		logf("CLEANING: %s", *flagOut)
		wl := splitToMap(*flagCleanWl, ",")
		outpath := *flagOut + string(filepath.Separator)
		err = filepath.Walk(outpath, func(n string, fi os.FileInfo, err error) error {
			switch {
			case os.IsNotExist(err):
				return nil
			case err != nil:
				return err
			}

			fn, sn := n[len(outpath):], fi.Name()
			if n == outpath || fn == "" || strings.HasPrefix(fn, ".") || strings.HasPrefix(sn, ".") || wl[fn] || wl[sn] || contains(files, fn) {
				return nil
			}
			logf("REMOVING: %s", n)
			return os.RemoveAll(n)
		})
		if err != nil {
			return err
		}
	}

	// write
	err = write(files)
	if err != nil {
		return err
	}

	// write generate protocol info
	if !*flagNoCopy {
		logf("WRITING: protocol.json")
		buf, err := json.MarshalIndent(protoInfo, "", "  ")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(*flagOut, "protocol.json"), buf, 0644)
		if err != nil {
			return err
		}
	}

	// goimports
	err = goimports(files)
	if err != nil {
		return err
	}

	// easyjson
	err = easyjson(pkgs)
	if err != nil {
		return err
	}

	// gofmt
	err = gofmt(files)
	if err != nil {
		return err
	}

	logf("done.")
	return nil
}

// loadProtocolInfo loads the protocol.json either from the path specified in
// -proto or by retrieving the versions specified in the -browser and -js
// files. Unless -nohar is specified, the virtual "HAR" domain will be
// generated as well and added to the specification.
func loadProtocolInfo() (*types.ProtocolInfo, error) {
	var err error

	if *flagProto != "" {
		logf("PROTOCOL: %s", *flagProto)
		buf, err := ioutil.ReadFile(*flagProto)
		if err != nil {
			return nil, err
		}

		// unmarshal
		protoInfo := new(types.ProtocolInfo)
		err = json.Unmarshal(buf, protoInfo)
		if err != nil {
			return nil, err
		}
		return protoInfo, nil
	}

	var protos [][]byte
	load := func(typ, ver, urlstr string) error {
		urlstr = fmt.Sprintf(urlstr, ver)
		logf("%s: %s", pad(strings.ToUpper(typ), 7), urlstr)
		buf, err := fileCacher{
			path: filepath.Join(*flagCache, typ, ver),
			ttl:  *flagTTL,
		}.Get(urlstr, true, typ+"_protocol.json")
		if err != nil {
			return err
		}
		protos = append(protos, buf)
		return nil
	}

	// grab browser + js definitions
	err = load("browser", *flagBrowser, browserURL)
	if err != nil {
		return nil, err
	}
	err = load("js", *flagJS, jsURL)
	if err != nil {
		return nil, err
	}

	// grab and add har definition
	if !*flagNoHar {
		harBuf, err := har.LoadProto(&fileCacher{
			path: filepath.Join(*flagCache, "har"),
			ttl:  *flagTTLHar,
		})
		if err != nil {
			return nil, err
		}
		protos = append(protos, harBuf)
	}

	return combineProtoInfos(protos...)
}

// cleanupTypes removes deprecated types.
func cleanupTypes(n string, dtyp string, typs []*types.Type) []*types.Type {
	var ret []*types.Type

	for _, t := range typs {
		typ := dtyp + "." + t.IDorName()
		if !*flagDep && t.Deprecated.Bool() {
			logf("SKIPPING(%s): %s [deprecated]", pad(n, 7), typ)
			continue
		}

		if !*flagRedirect && string(t.Redirect) != "" {
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

// cleanup removes deprecated types, events, and commands from the domain.
func cleanup(d *types.Domain) {
	d.Types = cleanupTypes("type", d.String(), d.Types)
	d.Events = cleanupTypes("event", d.String(), d.Events)
	d.Commands = cleanupTypes("command", d.String(), d.Commands)
}

// write writes all file buffer to disk.
func write(fileBuffers map[string]*bytes.Buffer) error {
	logf("WRITING: %d files", len(fileBuffers))

	var keys []string
	for k := range fileBuffers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// add out path
		n := filepath.Join(*flagOut, k)

		// create directory
		dir := filepath.Dir(n)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}

		// write file
		err = ioutil.WriteFile(n, fileBuffers[k].Bytes(), 0644)
		if err != nil {
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

	eg, ctxt := errgroup.WithContext(context.Background())
	for _, k := range keys {
		eg.Go(func(n string) func() error {
			n = filepath.Join(*flagOut, n)
			return func() error {
				buf, err := exec.CommandContext(ctxt, "goimports", "-w", n).CombinedOutput()
				if err != nil {
					return fmt.Errorf("could not format %s, got:\n%s", n, string(buf))
				}
				return nil
			}
		}(k))
	}
	return eg.Wait()
}

// easyjson runs easy json on the list of packages.
func easyjson(pkgs []string) error {
	params := []string{"-pkg", "-all", "-output_filename", "easyjson.go"}

	// generate easyjson stubs
	logf("RUNNING: easyjson (stubs)")
	eg, ctxt := errgroup.WithContext(context.Background())
	for _, k := range pkgs {
		eg.Go(func(n string) func() error {
			return func() error {
				cmd := exec.CommandContext(ctxt, "easyjson", append(params, "-stubs")...)
				cmd.Dir = filepath.Join(*flagOut, n)
				buf, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("could not generate easyjson stubs for %s, got:\n%s", cmd.Dir, string(buf))
				}
				return nil
			}
		}(k))
	}
	err := eg.Wait()
	if err != nil {
		return err
	}

	// generate actual easyjson types
	logf("RUNNING: easyjson")
	eg, ctxt = errgroup.WithContext(context.Background())
	for _, k := range pkgs {
		eg.Go(func(n string) func() error {
			return func() error {
				cmd := exec.CommandContext(ctxt, "easyjson", params...)
				cmd.Dir = filepath.Join(*flagOut, n)
				buf, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("could not easyjson %s, got:\n%s", cmd.Dir, string(buf))
				}
				return nil
			}
		}(k))
	}
	return eg.Wait()
}

// gofmt formats all the output file buffers on disk using gofmt.
func gofmt(fileBuffers map[string]*bytes.Buffer) error {
	logf("RUNNING: gofmt")

	var keys []string
	for k := range fileBuffers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	eg, ctxt := errgroup.WithContext(context.Background())
	for _, k := range keys {
		eg.Go(func(n string) func() error {
			n = filepath.Join(*flagOut, n)
			return func() error {
				buf, err := exec.CommandContext(ctxt, "gofmt", "-w", "-s", n).CombinedOutput()
				if err != nil {
					return fmt.Errorf("could not format %s, got:\n%s", n, string(buf))
				}
				return nil
			}
		}(k))
	}
	return eg.Wait()
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

// Retrieve retrieves a file from disk or from the remote URL, optionally
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

// combineProtoInfos combines the types and commands from multiple JSON-encoded
// protocol definitions.
func combineProtoInfos(buffers ...[]byte) (*types.ProtocolInfo, error) {
	protoInfo := new(types.ProtocolInfo)
	for _, buf := range buffers {
		var pi types.ProtocolInfo
		err := json.Unmarshal(buf, &pi)
		if err != nil {
			return nil, err
		}
		if protoInfo.Version == nil {
			protoInfo.Version = pi.Version
		}
		protoInfo.Domains = append(protoInfo.Domains, pi.Domains...)
	}
	return protoInfo, nil
}

// pathJoin is a simple wrapper around filepath.Join to simplify inline syntax.
func pathJoin(n string, m ...string) string {
	return filepath.Join(append([]string{n}, m...)...)
}

// logf is a wrapper around log.Printf.
func logf(s string, v ...interface{}) {
	if *flagVerbose {
		log.Printf(s, v...)
	}
}

// splitToMap splits a string to a map.
func splitToMap(s string, sep string) map[string]bool {
	z := strings.Split(s, sep)
	m := make(map[string]bool, len(z))
	for _, v := range z {
		m[v] = true
	}
	return m
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
