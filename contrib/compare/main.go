// Command compare compares the published Chrome Debugging Protocol JSON
// definitions on GitHub against the PDL definitions generated from the live
// Chromium source tree.
package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/chromedp/cdproto-gen/diff"
)

const (
	browserProto = "https://github.com/ChromeDevTools/devtools-protocol/raw/master/json/browser_protocol.json"
	jsProto      = "https://github.com/ChromeDevTools/devtools-protocol/raw/master/json/js_protocol.json"
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// grab definitions and merge
	res := make(map[string]interface{}, 0)
	for _, urlstr := range []string{browserProto, jsProto} {
		m, err := grab(urlstr)
		if err != nil {
			return err
		}
		merge(res, m)
	}

	// load har and merge
	m, err := loadJSON(filepath.Join(os.Getenv("GOPATH"), "pkg/cdproto-gen/har/har.json"))
	if err != nil {
		return err
	}
	merge(res, m)

	// find latest protocol definition on disk
	files, err := diff.FindFilesWithMask(
		filepath.Join(os.Getenv("GOPATH"), "src/github.com/chromedp/cdproto"),
		`^protocol-([^-]+)-(2[0-9]{7})\.json$`, 2,
	)
	if err != nil {
		return err
	}
	sort.Slice(files, func(a, b int) bool {
		return files[a].Date.After(files[b].Date)
	})

	// load latest protocol definition
	proto, err := loadJSON(files[0].Name)
	if err != nil {
		return err
	}

	log.Printf("DEEPEQUAL: %t", reflect.DeepEqual(res, proto))

	// write to disk
	paths, err := writeJSON(res, proto)
	if err != nil {
		return err
	}

	log.Printf("DEEPEQUAL: %t", reflect.DeepEqual(res, proto))

	buf, err := diff.CompareFiles(paths[0], paths[1])
	if err != nil {
		return err
	}
	if buf != nil {
		_, err = os.Stdout.Write(buf)
		return err
	}

	return nil
}

// grab retrieves a remote file.
func grab(urlstr string) (map[string]interface{}, error) {
	log.Printf("RETRIEVING: %s", urlstr)
	req, err := http.NewRequest("GET", urlstr, nil)
	if err != nil {
		return nil, err
	}

	cl := &http.Client{}
	res, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err = json.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// loadJSON reads a JSON file from disk.
func loadJSON(filename string) (map[string]interface{}, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err = json.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// merge merges src to dst.
func merge(dst, src map[string]interface{}) {
	for k, v := range src {
		if x, ok := dst[k]; ok {
			dst[k] = mergeVal(x, v)
		} else {
			dst[k] = v
		}
	}
}

// mergeVal merges two values.
func mergeVal(a, b interface{}) interface{} {
	am, mapa := a.(map[string]interface{})
	bm, mapb := b.(map[string]interface{})
	if mapa && mapb {
		merge(am, bm)
		return am
	}

	as, slicea := a.([]interface{})
	bs, sliceb := b.([]interface{})
	if slicea && sliceb {
		res := make([]interface{}, len(as)+len(bs))
		copy(res, as)
		copy(res[len(as):], bs)
		return res
	}

	return b
}

// writeJSON writes data
func writeJSON(protos ...map[string]interface{}) ([]string, error) {
	var files []string
	for _, m := range protos {
		sortMap(m)

		buf, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return nil, err
		}

		f, err := ioutil.TempFile(os.TempDir(), "proto-")
		if err != nil {
			return nil, err
		}

		filename := f.Name()
		log.Printf("WRITING: %s", filename)
		_, err = f.Write(buf)
		if err != nil {
			return nil, err
		}

		files = append(files, filename)
	}
	return files, nil
}

// sortMap recurses through a map and ensures all elements are ordered.
func sortMap(m map[string]interface{}) {
	for _, v := range m {
		switch z := v.(type) {
		case map[string]interface{}:
			sortMap(z)
		case []interface{}:
			sort.Slice(z, func(a, b int) bool {
				am, aok := z[a].(map[string]interface{})
				if aok {
					sortMap(am)
				}
				bm, bok := z[b].(map[string]interface{})
				if bok {
					sortMap(bm)
				}

				if aok && bok {
					return compareMap(am, bm, "id", "name", "domain")
				}
				return false
			})
		}
	}
}

// compareMap compares a, b by comparing the first field from fields found in a
// to be lexicographically less than the field in b.
func compareMap(a, b map[string]interface{}, fields ...string) bool {
	for _, field := range fields {
		ai, aok := a[field]
		bi, bok := b[field]
		if aok && !bok {
			panic("aok and not bok!")
		}
		if !aok && bok {
			panic("not aok and bok!")
		}
		if aok && bok {
			return strings.Compare(ai.(string), bi.(string)) < 0
		}
	}
	return false
}
