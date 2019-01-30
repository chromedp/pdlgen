package util

import (
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Cache holds information about a cached file.
type Cache struct {
	URL    string
	Path   string
	TTL    time.Duration
	Decode bool
}

// Get retrieves a file from disk or from the remote URL, optionally base64
// decoding it and writing it to disk.
func Get(c Cache) ([]byte, error) {
	var err error

	if err = os.MkdirAll(filepath.Dir(c.Path), 0755); err != nil {
		return nil, err
	}

	// check if exists on disk
	fi, err := os.Stat(c.Path)
	if err == nil && c.TTL != 0 && !time.Now().After(fi.ModTime().Add(c.TTL)) {
		return ioutil.ReadFile(c.Path)
	}

	Logf("RETRIEVING: %s", c.URL)

	// retrieve
	cl := &http.Client{}
	req, err := http.NewRequest("GET", c.URL, nil)
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
	if c.Decode {
		buf, err = base64.StdEncoding.DecodeString(string(buf))
		if err != nil {
			return nil, err
		}
	}

	Logf("WRITING: %s", c.Path)
	if err = ioutil.WriteFile(c.Path, buf, 0644); err != nil {
		return nil, err
	}

	return buf, nil
}
