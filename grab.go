// +build ignore

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"

	"github.com/chromedp/cdproto-gen/pdl"
	"github.com/chromedp/cdproto-gen/util"
)

var (
	flagTTL        = flag.Duration("ttl", 24*time.Hour, "file retrieval caching ttl")
	flagCache      = flag.String("cache", "", "protocol cache directory")
	flagMajorCount = flag.Int("major-count", 10, "major count")
	flagMinorCount = flag.Int("minor-count", 10, "count")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

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

	// create combined dir
	combinedDir := filepath.Join(*flagCache, "pdl", "combined")
	if err = os.MkdirAll(combinedDir, 0755); err != nil {
		return err
	}

	// get refs
	refs, err := util.GetRefs(util.Cache{
		URL:  util.ChromiumBase + "/+refs?format=JSON",
		Path: filepath.Join(*flagCache, "refs", "chromium.json"),
		TTL:  *flagTTL,
	})
	if err != nil {
		return err
	}

	// find tags
	var vers []*semver.Version
	for k := range refs {
		if !strings.HasPrefix(k, "refs/tags/") {
			continue
		}
		k = strings.TrimPrefix(k, "refs/tags/")
		if util.VerRE.MatchString(k) {
			vers = append(vers, util.MakeSemver(k))
		}
	}
	sort.Sort(semver.Collection(vers))

	var last int
	var majorCount, minorCount int
	var buf, chromiumBuf, v8Buf []byte
	harBuf := []byte(pdl.HAR)
	for i := len(vers) - 1; i >= 0 && majorCount < *flagMajorCount; i-- {
		// grab major
		ver := strings.Replace(vers[i].String(), "-", ".", -1)
		if !util.VerRE.MatchString(ver) {
			continue
		}
		major, err := strconv.Atoi(ver[:strings.Index(ver, ".")])
		if err != nil {
			return err
		}

		// break if less than 66
		if major < 67 {
			break
		}

		if minorCount < *flagMinorCount {
			// grab chromium pdl
			if chromiumBuf, err = util.Get(util.Cache{
				URL:    fmt.Sprintf(util.ChromiumURL+"?format=TEXT", ver),
				Path:   filepath.Join(*flagCache, "pdl", "chromium", ver+".pdl"),
				TTL:    *flagTTL,
				Decode: true,
			}); err != nil {
				return err
			}

			// grab deps
			var v8ver string
			if v8ver, err = util.GetDepVersion("v8", ver, util.Cache{
				URL:    fmt.Sprintf(util.ChromiumDeps+"?format=TEXT", ver),
				Path:   filepath.Join(*flagCache, "deps", "chromium", ver),
				TTL:    *flagTTL,
				Decode: true,
			}, util.Cache{
				URL:  util.V8Base + "/+refs?format=JSON",
				Path: filepath.Join(*flagCache, "refs", "v8.json"),
				TTL:  *flagTTL,
			}); err != nil {
				return err
			}

			// skip if not a numbered version
			if !util.VerRE.MatchString(v8ver) {
				continue
			}

			// grab v8 pdl
			if v8Buf, err = util.Get(util.Cache{
				URL:    fmt.Sprintf(util.V8URL+"?format=TEXT", v8ver),
				Path:   filepath.Join(*flagCache, "pdl", "v8", v8ver+".pdl"),
				TTL:    *flagTTL,
				Decode: true,
			}); err != nil {
				return err
			}

			// combine
			if buf, err = pdl.CombineBytes(chromiumBuf, v8Buf, harBuf); err != nil {
				return err
			}

			out := filepath.Join(combinedDir, fmt.Sprintf("%s_%s.pdl", ver, v8ver))
			util.Logf("WRITING: %s", out)
			if err = ioutil.WriteFile(out, buf, 0644); err != nil {
				return err
			}

			minorCount++
		}

		if last != major {
			last = major
			if majorCount != 0 {
				minorCount = 0
			}
			majorCount++
		}
	}

	return nil
}
