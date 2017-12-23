package har

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gedex/inflector"
	"github.com/knq/snaker"

	"github.com/chromedp/chromedp-gen/types"
)

const (
	specURL = "http://www.softwareishard.com/blog/har-12-spec/"

	cacheDataID = "CacheData"
)

// propRefMap is the map of property names to their respective type.
var propRefMap = map[string]string{
	"Log.creator":         "Creator",
	"Log.browser":         "Creator",
	"Log.pages":           "Page",
	"Log.entries":         "Entry",
	"Page.pageTimings":    "PageTimings",
	"Entry.request":       "Request",
	"Entry.response":      "Response",
	"Entry.cache":         "Cache",
	"Entry.timings":       "Timings",
	"Request.cookies":     "Cookie",
	"Request.headers":     "NameValuePair",
	"Request.queryString": "NameValuePair",
	"Request.postData":    "PostData",
	"Response.cookies":    "Cookie",
	"Response.headers":    "NameValuePair",
	"Response.content":    "Content",
	"PostData.params":     "Param",
	"Cache.beforeRequest": cacheDataID,
	"Cache.afterRequest":  cacheDataID,
}

// Cacher is an interface to retrieve and cache remote files to disk.
type Cacher interface {
	Load(...string) ([]byte, error)
	Cache([]byte, ...string) error
	Get(string, bool, ...string) ([]byte, error)
}

// LoadProto loads the HAR protocol definition using the cacher. If the
// har.json file is not cached, then it's generated from the remote spec and
// written to the cache.
func LoadProto(cacher Cacher) ([]byte, error) {
	// load file on disk
	harBuf, err := cacher.Load("har.json")
	if err == nil {
		return harBuf, nil
	}

	// grab spec file
	specBuf, err := cacher.Get(specURL, false, "spec.html")
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	harProto, err := generateDomain(specBuf)
	if err != nil {
		return nil, err
	}

	// marshal to json
	harBuf, err = json.MarshalIndent(harProto, "", "  ")
	if err != nil {
		return nil, err
	}

	// write
	err = cacher.Cache(harBuf, "har.json")
	if err != nil {
		return nil, err
	}

	return harBuf, nil
}

// generateDomain generates a HAR domain definition using the supplied cacher
// mechanism.
func generateDomain(buf []byte) (*types.ProtocolInfo, error) {
	// initial type map
	typeMap := map[string]types.Type{
		"HAR": {
			ID:          "HAR",
			Type:        types.TypeObject,
			Description: "Parent container for HAR log.",
			Properties: []*types.Type{{
				Name: "log",
				Ref:  "Log",
			}},
		},
		"NameValuePair": {
			ID:          "NameValuePair",
			Type:        types.TypeObject,
			Description: "Describes a name/value pair.",
			Properties: []*types.Type{{
				Name:        "name",
				Type:        types.TypeString,
				Description: "Name of the pair.",
			}, {
				Name:        "value",
				Type:        types.TypeString,
				Description: "Value of the pair.",
			}, {
				Name:        "comment",
				Type:        types.TypeString,
				Description: "A comment provided by the user or the application.",
				Optional:    types.Bool(true),
			}},
		},
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	// loop over type definitions
	doc.Find(`h3:contains("HAR Data Structure") + p + p + ul a`).Each(func(i int, s *goquery.Selection) {
		n := s.Text()

		// skip browser (same as creator)
		switch n {
		case "browser", "queryString", "headers":
			return
		}

		// generate the object ID
		id := inflector.Singularize(snaker.ForceCamelIdentifier(n))
		if strings.HasSuffix(id, "um") {
			id = strings.TrimSuffix(id, "um") + "a"
		}
		if strings.HasSuffix(id, "Timing") {
			id += "s"
		}

		// base selector
		sel := fmt.Sprintf(".harType#%s", n)

		// grab description
		desc := strings.TrimSpace(doc.Find(sel + " + p").Text())
		if desc == "" {
			panic(fmt.Sprintf("%s (%s) has no description", n, id))
		}

		// grab properties and scan
		props, err := scanProps(id, readPropText(sel, doc))
		if err != nil {
			panic(fmt.Sprintf("could not scan properties for %s (%s): %v", n, id, err))
		}

		// add to type map
		typeMap[id] = types.Type{
			ID:          id,
			Type:        types.TypeObject,
			Description: desc,
			Properties:  props,
		}
	})

	// grab and scan cachedata properties
	cacheDataPropText := readPropText(`p:contains("Both beforeRequest and afterRequest object share the following structure.")`, doc)
	cacheDataProps, err := scanProps(cacheDataID, cacheDataPropText)
	if err != nil {
		return nil, err
	}
	typeMap[cacheDataID] = types.Type{
		ID:          cacheDataID,
		Type:        types.TypeObject,
		Description: "Describes the cache data for beforeRequest and afterRequest.",
		Properties:  cacheDataProps,
	}

	// sort by type names
	var typeNames []string
	for n := range typeMap {
		typeNames = append(typeNames, n)
	}
	sort.Strings(typeNames)

	// add to type list
	var typs []*types.Type
	for _, n := range typeNames {
		typ := typeMap[n]
		typs = append(typs, &typ)
	}

	// create the protocol info
	return &types.ProtocolInfo{
		Version: &types.Version{Major: "1", Minor: "2"},
		Domains: []*types.Domain{{
			Domain:      types.DomainType("HAR"),
			Description: "HTTP Archive Format",
			Types:       typs,
		}},
	}, nil
}

func scanProps(id string, propText string) ([]*types.Type, error) {
	// scan properties
	var props []*types.Type
	scanner := bufio.NewScanner(strings.NewReader(propText))
	i := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// grab prop stuff
		propName := strings.TrimSpace(line[:strings.IndexAny(line, "[")])
		propDesc := strings.TrimSpace(line[strings.Index(line, "-")+1:])
		if propName == "" || propDesc == "" {
			return nil, fmt.Errorf("line %d missing either name or description", i)
		}
		opts := strings.TrimSpace(line[strings.Index(line, "[")+1 : strings.Index(line, "]")])

		// determine type
		typ := types.TypeEnum(opts)
		if z := strings.Index(opts, ","); z != -1 {
			typ = types.TypeEnum(strings.TrimSpace(opts[:z]))
		}

		// convert some fields to integers
		if strings.Contains(strings.ToLower(propName), "size") ||
			propName == "compression" || propName == "status" ||
			propName == "hitCount" {
			typ = types.TypeInteger
		}

		// fix object/array refs
		var ref string
		var items *types.Type
		fqPropName := fmt.Sprintf("%s.%s", id, propName)
		switch typ {
		case types.TypeObject:
			typ = types.TypeEnum("")
			ref = propRefMap[fqPropName]

		case types.TypeArray:
			items = &types.Type{
				Ref: propRefMap[fqPropName],
			}
		}

		// add property
		props = append(props, &types.Type{
			Name:        propName,
			Type:        typ,
			Description: propDesc,
			Ref:         ref,
			Items:       items,
			Optional:    types.Bool(strings.Contains(opts, "optional")),
		})

		i++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return props, nil
}

func readPropText(sel string, doc *goquery.Document) string {
	text := strings.TrimSpace(doc.Find(sel).NextAllFiltered("ul").Text())
	j := strings.Index(text, "\n\n")
	if j == -1 {
		panic(fmt.Sprintf("could not find property description for `%s`", sel))
	}
	return text[:j]
}
