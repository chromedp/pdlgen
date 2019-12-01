package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/PuerkitoBio/goquery"
)

const (
	ChromiumBase = "https://chromium.googlesource.com/chromium/src"
	ChromiumDeps = ChromiumBase + "/+/%s/DEPS"
	ChromiumURL  = ChromiumBase + "/+/%s/third_party/blink/public/devtools_protocol/browser_protocol.pdl"

	V8Base = "https://chromium.googlesource.com/v8/v8"
	V8URL  = V8Base + "/+/%s/include/js_protocol.pdl"

	// v8 <= 7.6.303.13 uses this path. left for posterity.
	V8URLOld = V8Base + "/+/%s/src/inspector/js_protocol.pdl"

	// chromium < 80.0.3978.0 uses this path. left for posterity.
	ChromiumURLOld = ChromiumBase + "/+/%s/third_party/blink/renderer/core/inspector/browser_protocol.pdl"
)

// Logf is a shared logging function.
var Logf = log.Printf

// GetLatestVersion determines the latest tag version listed on the gitiles
// html page.
func GetLatestVersion(index Cache) (string, error) {
	buf, err := Get(index)
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

// Ref wraps a ref.
type Ref struct {
	Value  string `json:"value"`
	Target string `json:"target"`
}

// GetRefs returns the refs for the url.
func GetRefs(c Cache) (map[string]Ref, error) {
	// grab refs
	buf, err := Get(c)
	if err != nil {
		return nil, err
	}

	// chop first line
	buf = buf[bytes.Index(buf, []byte("\n")):]

	// unmarshal
	var refs map[string]Ref
	if err = json.Unmarshal(buf, &refs); err != nil {
		return nil, err
	}
	return refs, nil
}

var revRE = regexp.MustCompile(`(?is)\s+'([0-9a-f]+)'`)

// GetDepVersion version retrieves the v8 version used for the browser version.
func GetDepVersion(typ, ver string, deps, refs Cache) (string, error) {
	buf, err := Get(deps)
	if err != nil {
		return "", err
	}

	// determine revision
	mark := []byte("'" + typ + "_revision':")
	i := bytes.Index(buf, mark)
	if i == -1 {
		return "", fmt.Errorf("could not find revision for %s version %s", typ, ver)
	}
	buf = buf[i+len(mark):]
	m := revRE.FindSubmatch(buf)
	if m == nil {
		return "", fmt.Errorf("no revision for %s version %s", typ, ver)
	}
	rev := string(m[1])

	// grab refs
	r, err := GetRefs(refs)
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
