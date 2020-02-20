// Package tplgen generates code from templates for Chrome DevTools Protocol
// definitions.
package tplgen

import (
	"context"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/chromedp/pdlgen/tplgen/cdprototpl"
	"github.com/chromedp/pdlgen/util"
	"github.com/chromedp/pdlgen/util/pdl"
	"github.com/chromedp/pdlgen/util/pdlcache"
	"github.com/spf13/afero"
)

// Flags are the exec flags.
type Flags struct {
	Debug       bool          `help:"toggle debug (writes generated files to disk without post-processing)"`
	Template    string        `help:"template" default:"cdproto"`
	CacheDir    string        `help:"protocol cache directory" default:"${cacheDir}"`
	TTL         time.Duration `help:"file retrieval caching ttl" default:"24h"`
	PDL         string        `help:"path to pdl file to use"`
	OS          string        `help:"os" default:"win64"`
	Channel     string        `help:"channel" default:"canary"`
	Browser     string        `help:"browser protocol version"`
	V8          string        `help:"v8 protocol version" name:"v8"`
	Har         bool          `help:"include har domain" default:"true"`
	DiffPDL     bool          `help:"display pdl diff" default:"true"`
	DiffSource  bool          `help:"display source diff" default:"true"`
	CleanSource bool          `help:"clean output source" default:"true"`

	Cdproto cdprototpl.Generator `embed prefix:"cdproto-"`
	// NoClean  bool          `help:"toggle not cleaning (removing) existing directories"`
	// NoDump   bool          `help:"toggle not dumping generated protocol file to out directory"`
	// GoPkg    string        `help:"go base package name"` // flag.String("go-pkg", "github.com/chromedp/cdproto", "")
	// GoWl     string        `help:""` // flag.String("go-wl", "LICENSE,README.md,*.pdl,go.mod,go.sum,"+easyjsonGo, "comma-separated list of files to whitelist (ignore)")
}

// NewFlags creates a new flag set.
func NewFlags() (*kong.Context, *Flags, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, nil, err
	}
	flags := new(Flags)
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	ctx := kong.Parse(flags, kong.Vars{
		"cacheDir": filepath.Join(cacheDir, "pdlgen"),
		"GOPATH":   gopath,
	})
	return ctx, flags, nil
}

// NewCache creates a new pdl cache.
func NewCache(flags *Flags, stdout, stderr io.Writer) (*pdlcache.Cache, error) {
	// build cache
	opts := []pdlcache.Option{
		pdlcache.WithCacheDir(flags.CacheDir),
		pdlcache.WithTTL(flags.TTL),
		pdlcache.WithLogf(func(s string, v ...interface{}) {
			fmt.Fprintf(stdout, s+"\n", v...)
		}),
	}
	if flags.Har {
		opts = append(opts, pdlcache.WithHar())
	}
	return pdlcache.New(opts...)
}

// Load loads the pdl definitions.
func Load(ctx context.Context, flags *Flags, cache *pdlcache.Cache) (*pdl.PDL, string, string, error) {
	if flags.PDL != "" {
		def, err := pdl.LoadFile(flags.PDL)
		if err != nil {
			return nil, "", "", err
		}
		return def, "", "", nil
	}
	// load pdl
	v, err := cache.LatestVersion(ctx, flags.OS, flags.Channel)
	if err != nil {
		return nil, "", "", err
	}
	ver, v8ver := v.Version, v.V8Version
	if flags.Browser != "" {
		ver = flags.Browser
	}
	if flags.V8 != "" {
		v8ver = flags.V8
	}
	def, err := cache.CombinedDef(ctx, ver, v8ver)
	if err != nil {
		return nil, "", "", err
	}
	return def, ver, v8ver, nil
}

// Exec loads pdl definitions and runs the generator with the specified command
// flags.
func Exec(ctx context.Context, stdout, stderr io.Writer) error {
	_, flags, err := NewFlags()
	if err != nil {
		return err
	}
	var g interface{}
	switch flags.Template {
	case "cdproto":
		g = flags.Cdproto
	default:
		return fmt.Errorf("unknown template type %q", flags.Template)
	}
	// read previous version
	var prevVer, prevV8Ver string
	if x, ok := g.(interface {
		ReadVersion() (string, string, error)
	}); ok {
		if prevVer, prevV8Ver, err = x.ReadVersion(); err != nil {
			return err
		}
	}
	// load pdl
	cache, err := NewCache(flags, stdout, stderr)
	if err != nil {
		return err
	}
	def, ver, v8ver, err := Load(ctx, flags, cache)
	if err != nil {
		return err
	}
	sortDomains(def)
	// display diff
	if flags.DiffPDL && prevVer != "" && prevV8Ver != "" {
		prev, err := cache.CombinedDef(ctx, prevVer, prevV8Ver)
		if err != nil {
			return err
		}
		sortDomains(prev)
		if err = Diff(ctx, stdout, def, ver, v8ver, prev, prevVer, prevV8Ver); err != nil {
			return err
		}
	}
	// retrieve whitelist
	var whitelist []string
	if x, ok := g.(interface {
		Whitelist() ([]string, error)
	}); ok {
		whitelist, err = x.Whitelist()
		if err != nil {
			return err
		}
		whitelist = whitelist
	}
	// generate
	if x, ok := g.(interface {
		Generate(context.Context, afero.Fs) error
	}); ok {
		var old afero.Fs
		x.Generate(ctx, old)
	}
	return nil
}

// Diff writes the diff between the pdl versions.
func Diff(ctx context.Context, w io.Writer, def *pdl.PDL, ver, v8ver string, prev *pdl.PDL, prevVer, prevV8Ver string) error {
	d, err := ioutil.TempDir("", "pdlgen-diff")
	if err != nil {
		return err
	}
	defer os.RemoveAll(d)
	if err := os.MkdirAll(filepath.Join(d, "current"), 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(d, "previous"), 0777); err != nil {
		return err
	}
	name := filepath.Join(d, "current", fmt.Sprintf("%s_%s.pdl", ver, v8ver))
	if err := ioutil.WriteFile(name, def.Bytes(), 0644); err != nil {
		return err
	}
	prevName := filepath.Join(d, "previous", fmt.Sprintf("%s_%s.pdl", prevVer, prevV8Ver))
	if err := ioutil.WriteFile(prevName, prev.Bytes(), 0644); err != nil {
		return err
	}
	buf, err := util.CompareFiles(prevName, name)
	if err != nil {
		return err
	}
	if buf != nil {
		_, err := w.Write(buf)
		return err
	}
	return nil
}

// sortDomains sorts the domains in the pdl alphabetically.
func sortDomains(def *pdl.PDL) {
	// sort domains
	sort.Slice(def.Domains, func(i, j int) bool {
		return strings.Compare(def.Domains[i].Domain.String(), def.Domains[j].Domain.String()) <= 0
	})
}

/*

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
				}
			}
		}
		// will process
		pkgs = append(pkgs, "") // genutil.PackageName(d))
		processed = append(processed, d)
		// cleanup types, events, commands
		d.Types = cleanupTypes("type", d.Domain.String(), d.Types)
		d.Events = cleanupTypes("event", d.Domain.String(), d.Events)
		d.Commands = cleanupTypes("command", d.Domain.String(), d.Commands)
	}
	// fixup
	//	fixup.FixDomains(processed)
	// get generator
	generator := Generators()["go"]
	if generator == nil {
		return errors.New("no generator")
	}
	// emit
	emitter, err := generator(processed, args.GoPkg)
	logf := func(string, ...interface{}) {}
	if args.Debug {
		logf = log.New(os.Stdout, "", log.LstdFlags).Printf
	}
	// create cache
	cache, err := createCache(logf, args.CacheDir, args.TTL, args.Type == "cdproto")
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
	if args.Debug {
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
*/

/*
// loadProtoDefs loads the protocol definitions either from the path specified
// in -proto or by retrieving the versions specified in the -browser and -js
// files.
func loadProtoDefs() (*pdl.PDL, error) {
	// load definition
	def, err := loadCombinedDef(ctx, cache, args.File, args.OS, args.Channel, args.Version)
	if err != nil {
		return err
	}
	g := gen.NewGenerator(logf)
	whitelist, err := emitter.Emit(g, def)
	if err != nil {
		return err
	}
	// not an overlay directory
	if whilelist == nil {
	}
	return nil
}

// createCache creates the PDL cache.
func createCache(logf func(string, ...interface{}), cacheDir string, ttl time.Duration, har bool) (*pdlcache.Cache, error) {
	var err error
	if args.Pdl != "" {
		util.Logf("PROTOCOL: %s", args.Pdl)
		buf, err := ioutil.ReadFile(args.Pdl)
		if err != nil {
			return nil, err
		}
		return pdl.Parse(buf)
	}
	var protoDefs []*pdl.PDL
	load := func(urlstr, typ, ver string) error {
		buf, err := util.Get(util.Cache{
			URL:    fmt.Sprintf(urlstr+"?format=TEXT", ver),
			Path:   filepath.Join(args.Cache, "pdl", typ, ver+".pdl"),
			TTL:    args.TTL,
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
		n := filepath.Join(args.Out, k)
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
				fn := filepath.Join(args.Out, n)
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
	util.Logf("RUNNING: easyjson")
	eg, _ := errgroup.WithContext(context.Background())
	for _, k := range pkgs {
		eg.Go(func(n string) func() error {
			return func() error {
				n = filepath.Join(args.Out, n)
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
	util.Logf("RUNNING: gofmt")
	eg, _ := errgroup.WithContext(context.Background())
	for _, k := range files {
		eg.Go(func(n string) func() error {
			return func() error {
				n = filepath.Join(args.Out, n)
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
	for _, z := range strings.Split(args.GoWl, ",") {
		if z == n || glob.Glob(z, n) {
			return true
		}
	}
	return false
}

*/
