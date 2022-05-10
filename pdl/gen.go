//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gedex/inflector"
	"github.com/kenshaw/snaker"

	"github.com/chromedp/cdproto-gen/pdl"
)

const (
	specURL = "http://www.softwareishard.com/blog/har-12-spec/"
)

var flagOut = flag.String("o", "har.go", "out file")

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run downloads and generates a HAR definition from the remote website,
// writing the generated definition to flagOut.
func run() error {
	// retrieve
	buf, err := grab(specURL)
	if err != nil {
		return err
	}

	// generate
	pdl, err := generate(buf)
	if err != nil {
		return err
	}

	// escape
	pdlBuf := bytes.Replace(pdl.Bytes(), []byte("`"), []byte("\\`"), -1)
	b := new(bytes.Buffer)
	fmt.Fprintf(b, harTpl, string(pdlBuf))
	return ioutil.WriteFile(*flagOut, b.Bytes(), 0o644)
}

// grab retrieves a url.
func grab(urlstr string) ([]byte, error) {
	req, err := http.NewRequest("GET", specURL, nil)
	if err != nil {
		return nil, err
	}
	cl := &http.Client{}
	res, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return ioutil.ReadAll(res.Body)
}

const (
	cacheDataID = "CacheData"
)

// generate generates a PDL from the supplied HTML page containing a single
// 'HAR' domain.
func generate(buf []byte) (*pdl.PDL, error) {
	// initial type map
	typeMap := map[string]pdl.Type{
		"HAR": {
			Type:        pdl.TypeObject,
			Name:        "HAR",
			Description: "Parent container for HAR log.",
			Properties: []*pdl.Type{{
				Name: "log",
				Ref:  "Log",
			}},
		},
		"NameValuePair": {
			Name:        "NameValuePair",
			Type:        pdl.TypeObject,
			Description: "Describes a name/value pair.",
			Properties: []*pdl.Type{{
				Name:        "name",
				Type:        pdl.TypeString,
				Description: "Name of the pair.",
			}, {
				Name:        "value",
				Type:        pdl.TypeString,
				Description: "Value of the pair.",
			}, {
				Name:        "comment",
				Type:        pdl.TypeString,
				Description: "A comment provided by the user or the application.",
				Optional:    true,
			}},
		},
	}

	// parse file
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

		// convert <type> -> [Type] in description
		desc = typeDescRE.ReplaceAllStringFunc(desc, func(s string) string {
			s = strings.ToUpper(string(rune(s[1]))) + s[2:len(s)-1]
			return "[" + s + "]"
		})

		// clean description
		desc = descCleanRE.ReplaceAllString(desc, "")
		desc = strings.ToUpper(desc[0:1]) + desc[1:]

		// grab properties and scan
		props, err := scanProps(id, readPropText(sel, doc))
		if err != nil {
			panic(fmt.Sprintf("could not scan properties for %s (%s): %v", n, id, err))
		}

		// add to type map
		typeMap[id] = pdl.Type{
			Type:        pdl.TypeObject,
			Name:        id,
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
	typeMap[cacheDataID] = pdl.Type{
		Name:        cacheDataID,
		Type:        pdl.TypeObject,
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
	var typs []*pdl.Type
	for _, n := range typeNames {
		typ := typeMap[n]
		typs = append(typs, &typ)
	}

	// create the protocol info
	return &pdl.PDL{
		Version: &pdl.Version{Major: 1, Minor: 3},
		Domains: []*pdl.Domain{{
			Domain:      pdl.DomainType("HAR"),
			Description: "HTTP Archive Format",
			Types:       typs,
		}},
	}, nil
}

// scanProps scans the supplied properties, converting into the appropriate
// type.
func scanProps(id string, propText string) ([]*pdl.Type, error) {
	var i int
	var props []*pdl.Type

	// scan properties
	scanner := bufio.NewScanner(strings.NewReader(propText))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// grab prop information
		propName := strings.TrimSpace(line[:strings.IndexAny(line, "[")])
		propDesc := strings.TrimSpace(line[strings.Index(line, "-")+1:])
		if propName == "" || propDesc == "" {
			return nil, fmt.Errorf("line %d missing either name or description", i)
		}
		opts := strings.TrimSpace(line[strings.Index(line, "[")+1 : strings.Index(line, "]")])

		// convert <type> -> [Type] in prop description
		propDesc = typeDescRE.ReplaceAllStringFunc(propDesc, func(s string) string {
			s = strings.ToUpper(string(rune(s[1]))) + s[2:len(s)-1]
			return "[" + s + "]"
		})

		// determine type
		typ := pdl.TypeEnum(opts)
		if z := strings.Index(opts, ","); z != -1 {
			typ = pdl.TypeEnum(strings.TrimSpace(opts[:z]))
		}

		// convert some fields to integers
		if strings.Contains(strings.ToLower(propName), "size") ||
			propName == "compression" || propName == "status" ||
			propName == "hitCount" {
			typ = pdl.TypeInteger
		}

		// fix object/array refs
		var ref string
		var items *pdl.Type
		fqPropName := fmt.Sprintf("%s.%s", id, propName)
		switch typ {
		case pdl.TypeObject:
			typ = pdl.TypeEnum("")
			ref = propRefMap[fqPropName]

		case pdl.TypeArray:
			items = &pdl.Type{
				Ref: propRefMap[fqPropName],
			}
		}

		// add property
		props = append(props, &pdl.Type{
			Name:        propName,
			Type:        typ,
			Description: propDesc,
			Ref:         ref,
			Items:       items,
			Optional:    strings.Contains(opts, "optional"),
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

var descCleanRE = regexp.MustCompile(`(?i)^this\s*objects?\s+`)

var typeDescRE = regexp.MustCompile(`(?i)<([a-z]+)>`)

const (
	harTpl = `package pdl

//go:generate go run gen.go -o har.go

// Generated by gen.go. DO NOT EDIT.

// HAR is the PDL formatted definition of HTTP Archive (HAR) types.
const HAR = ` + "`%s`\n"
)
