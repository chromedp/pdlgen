package pdlcache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/pdlgen/util/pdl"
	"github.com/kenshaw/diskcache"
)

const (
	BrowserBase = "https://chromium.googlesource.com/chromium/src"
	BrowserDeps = BrowserBase + "/+/%s/DEPS"
	BrowserDef  = BrowserBase + "/+/%s/third_party/blink/public/devtools_protocol/browser_protocol.pdl"
	V8Base      = "https://chromium.googlesource.com/v8/v8"
	V8Def       = V8Base + "/+/%s/include/js_protocol.pdl"
	// browser < 80.0.3978.0 uses this path
	PrevBrowserVer = "80.0.3978.0"
	PrevBrowserDef = BrowserBase + "/+/%s/third_party/blink/renderer/core/inspector/browser_protocol.pdl"
	// v8 < 7.6.303.13 uses this path
	// PrevV8Ver = "7.6.303.13"
	// PrevV8Def = V8Base + "/+/%s/src/inspector/js_protocol.pdl"
)

// Cache manages looking up and retrieving browser and v8 protocol definitions
// (PDLs) from the Chromium source tree.
type Cache struct {
	name       string
	cacheDir   string
	cache      *diskcache.Cache
	ttl        time.Duration
	refs       map[string]map[string]Ref
	defs       map[string]*pdl.PDL
	additional []*pdl.PDL
	logf       func(string, ...interface{})
	sync.RWMutex
}

// New creates a new pdl definition cache.
func New(opts ...Option) (*Cache, error) {
	c := &Cache{
		name: "pdlgen",
		refs: make(map[string]map[string]Ref),
		defs: make(map[string]*pdl.PDL),
		logf: func(string, ...interface{}) {},
	}
	for _, o := range opts {
		o(c)
	}
	// determine cache dir
	if c.cacheDir == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		c.cacheDir = filepath.Join(dir, "pdlgen")
	}
	var err error
	c.cache, err = diskcache.New(
		diskcache.WithBasePathFs(c.cacheDir),
		diskcache.WithTTL(c.ttl),
		diskcache.WithHeaderWhitelist("Date", "Content-Type"),
		diskcache.WithErrorTruncator(),
		diskcache.WithMinifier(),
		diskcache.WithGzipCompression(),
		diskcache.WithMatchers(
			diskcache.Match(
				`GET`,
				`^https://chromium\.googlesource\.com$`,
				`^/(?P<project>v8|chromium)/(v8|src)/\+refs$`,
				`refs/{{project}}`,
				diskcache.WithPrefixStripper([]byte(")]}'\n"), "application/json"),
				diskcache.WithFlatGzipCompression(),
			),
			diskcache.Match(
				`GET`,
				`^https://chromium\.googlesource\.com$`,
				`^/chromium/src/\+/(?P<ver>[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?)/DEPS$`,
				`deps/browser/{{ver}}`,
				diskcache.WithBase64Decoder("text/plain"),
				diskcache.WithFlatGzipCompression(),
			),
			diskcache.Match(
				`GET`,
				`^https://chromium\.googlesource\.com$`,
				`^/(?P<project>v8|chromium)/(v8|src)/\+/(?P<ver>[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?).+?/(js|browser)_protocol\.pdl$`,
				`pdl/{{project}}/{{ver}}.pdl`,
				diskcache.WithBase64Decoder("text/plain"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// get retrieves the passed url.
func (c *Cache) get(ctx context.Context, urlstr string) ([]byte, error) {
	req, err := http.NewRequest("GET", urlstr, nil)
	if err != nil {
		return nil, err
	}
	cached, err := c.cache.Cached(req)
	if err != nil {
		return nil, err
	}
	if !cached {
		c.logf("RETRIEVING: %s", urlstr)
	}
	// retrieve and decode
	cl := &http.Client{Transport: c.cache}
	res, err := cl.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not retrieve %s (status: %d)", urlstr, res.StatusCode)
	}
	return ioutil.ReadAll(res.Body)
}

// Refs returns the refs for the url.
func (c *Cache) Refs(ctx context.Context, urlstr string) (map[string]Ref, error) {
	c.RLock()
	refs := c.refs[urlstr]
	c.RUnlock()
	if refs != nil {
		return refs, nil
	}
	c.Lock()
	defer c.Unlock()
	// grab, unmarshal, store
	buf, err := c.get(ctx, urlstr+"/+refs?format=JSON")
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(buf, &refs); err != nil {
		return nil, err
	}
	c.refs[urlstr] = refs
	return refs, nil
}

// Def retrieves and decodes a pdl from the chromium source tree.
func (c *Cache) Def(ctx context.Context, urlstr string) (*pdl.PDL, error) {
	c.RLock()
	def := c.defs[urlstr]
	c.RUnlock()
	if def != nil {
		return def, nil
	}
	c.Lock()
	defer c.Unlock()
	buf, err := c.get(ctx, urlstr+"?format=TEXT")
	if err != nil {
		return nil, err
	}
	def, err = pdl.Parse(buf)
	if err != nil {
		return nil, err
	}
	c.defs[urlstr] = def
	return def, nil
}

// BrowserRefs retrieves the browser refs.
func (c *Cache) BrowserRefs(ctx context.Context) (map[string]Ref, error) {
	return c.Refs(ctx, BrowserBase)
}

// BrowserDef retrieves the pdl for the specified browser version.
func (c *Cache) BrowserDef(ctx context.Context, ver string) (*pdl.PDL, error) {
	urlstr := BrowserDef
	if CompareSemver(ver, PrevBrowserVer) {
		urlstr = PrevBrowserDef
	}
	return c.Def(ctx, fmt.Sprintf(urlstr, ver))
}

// V8Refs retrieves the v8 refs.
func (c *Cache) V8Refs(ctx context.Context) (map[string]Ref, error) {
	return c.Refs(ctx, V8Base)
}

// V8Def retrieves the pdl for the specified v8 version.
func (c *Cache) V8Def(ctx context.Context, ver string) (*pdl.PDL, error) {
	return c.Def(ctx, fmt.Sprintf(V8Def, ver))
}

// depsRevRE is a regexp to match a sha1 hash in a DEPS file.
var depsRevRE = regexp.MustCompile(`(?is)\s+'([0-9a-f]+)'`)

// BrowserV8Version determines the v8 version for a specific browser
// version. Specifically, retrieves the revision (sha1 hash) listed in the
// browser version's DEPS.
func (c *Cache) BrowserV8Version(ctx context.Context, ver string) (string, error) {
	buf, err := c.get(ctx, fmt.Sprintf(BrowserDeps, ver)+"?format=TEXT")
	if err != nil {
		return "", err
	}
	// determine revision
	mark := []byte("'v8_revision':")
	i := bytes.Index(buf, mark)
	if i == -1 {
		return "", fmt.Errorf("could not find v8 revision for browser %s", ver)
	}
	buf = buf[i+len(mark):]
	m := depsRevRE.FindSubmatch(buf)
	if m == nil {
		return "", fmt.Errorf("no matching v8 revision found for browser %s", ver)
	}
	rev := string(m[1])
	// grab refs
	r, err := c.Refs(ctx, V8Base)
	if err != nil {
		return "", err
	}
	// find tag
	for k, v := range r {
		if !strings.HasPrefix(k, "refs/tags/") {
			continue
		}
		if v.Value == rev {
			return strings.TrimPrefix(k, "refs/tags/"), nil
		}
	}
	return "", fmt.Errorf("could not find %s revision tag for rev %s", rev, m[1])
}

// CombinedDef retrieves the combined browser and v8 protocol definitions
// for a given chromium version.
func (c *Cache) CombinedDef(ctx context.Context, ver, v8ver string) (*pdl.PDL, error) {
	v8Def, err := c.V8Def(ctx, v8ver)
	if err != nil {
		return nil, err
	}
	browserDef, err := c.BrowserDef(ctx, ver)
	if err != nil {
		return nil, err
	}
	return pdl.Combine(append([]*pdl.PDL{v8Def, browserDef}, c.additional...)...), nil
}

// CombinedDefBytes retrieves the combined browser and v8 protocol definitions
// for a given chromium version.
func (c *Cache) CombinedDefBytes(ctx context.Context, ver, v8ver string) ([]byte, error) {
	combined, err := c.CombinedDef(ctx, ver, v8ver)
	if err != nil {
		return nil, err
	}
	return combined.Bytes(), nil
}

// LatestCombinedDef returns the latest combined def for the specified os and
// channel.
func (c *Cache) LatestCombinedDef(ctx context.Context, os, channel string) (*pdl.PDL, error) {
	ver, err := c.LatestVersion(ctx, os, channel)
	if err != nil {
		return nil, err
	}
	return c.CombinedDef(ctx, ver.Version, ver.V8Version)
}

// LatestCombinedBytes retrieves the latest combined def for the specified os
// and channel.
func (c *Cache) LatestCombinedBytes(ctx context.Context, os, channel string) ([]byte, error) {
	def, err := c.LatestCombinedDef(ctx, os, channel)
	if err != nil {
		return nil, err
	}
	return def.Bytes(), nil
}

// Ref wraps a ref.
type Ref struct {
	Value  string `json:"value"`
	Target string `json:"target"`
}

// Version wraps browser version information.
type Version struct {
	BranchCommit       string `json:"branch_commit"`
	BranchBasePosition string `json:"branch_base_position"`
	SkiaCommit         string `json:"skia_commit"`
	V8Version          string `json:"v8_version"`
	PreviousVersion    string `json:"previous_version"`
	V8Commit           string `json:"v8_commit"`
	TrueBranch         string `json:"true_branch"`
	PreviousReldate    string `json:"previous_reldate"`
	BranchBaseCommit   string `json:"branch_base_commit"`
	Version            string `json:"version"`
	CurrentReldate     string `json:"current_reldate"`
	CurrentVersion     string `json:"current_version"`
	OS                 string `json:"os"`
	Channel            string `json:"channel"`
	ChromiumCommit     string `json:"chromium_commit"`
}

// String satisfies the fmt.Stringer interface.
func (v Version) String() string {
	return fmt.Sprintf("Chromium %s (v8: %s, os: %s, channel: %s)", v.Version, v.V8Version, v.OS, v.Channel)
}

// VersionEntry is a OS version entry detailing the available to browser version entries.
type VersionEntry struct {
	OS       string    `json:"os"`
	Versions []Version `json:"versions"`
}

// Option is a cache option.
type Option func(*Cache)

// WithCacheDir is a pdl definition cache option to set the cache directory.
func WithCacheDir(cacheDir string) Option {
	return func(c *Cache) {
		c.cacheDir = cacheDir
	}
}

// WithLogf is a pdl definition cache option to set a log func.
func WithLogf(logf func(string, ...interface{})) Option {
	return func(c *Cache) {
		c.logf = logf
	}
}

// WithTTL is a pdl definition cache option to set the ttl for retrieved PDL
// files.
func WithTTL(ttl time.Duration) Option {
	return func(c *Cache) {
		c.ttl = ttl
	}
}

// WithHar is a pdl definition cache option to include the HAR domain
// definition in the combined PDL format.
func WithHar() Option {
	return func(c *Cache) {
		harDef, err := pdl.Parse([]byte(pdl.HAR))
		if err != nil {
			panic(err)
		}
		c.additional = append(c.additional, harDef)
	}
}

// old version using gittiles
// GetLatestVersion determines the latest tag version listed on the gitiles
// html page.
/*
func (c *Cache) GetLatestVersion() (string, error) {
	buf, err := c.Get(index)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	var vers []*semver.Version
	doc.Find(`h3:contains("Tags") + ul li`).Each(func(i int, s *goquery.Selection) {
		if t := s.Text(); VerRE.MatchString(t) {
			vers = append(vers, MakeSemver(t))
		}
	})
	if len(vers) < 1 {
		return "", fmt.Errorf("could not find a valid tag at %s", index.URL)
	}
	sort.Sort(semver.Collection(vers))
	return strings.Replace(vers[len(vers)-1].String(), "-", ".", -1), nil
}
*/
