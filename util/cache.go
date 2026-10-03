package util

import (
	"encoding/base64"
	"fmt"
	"io"
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

	if err = os.MkdirAll(filepath.Dir(c.Path), 0o755); err != nil {
		return nil, err
	}

	// check if exists on disk
	fi, err := os.Stat(c.Path)
	if err == nil && c.TTL != 0 && !time.Now().After(fi.ModTime().Add(c.TTL)) {
		return os.ReadFile(c.Path)
	}

	buf, err := fetch(c.URL)
	if err != nil {
		return nil, err
	}

	// decode
	if c.Decode {
		buf, err = base64.StdEncoding.DecodeString(string(buf))
		if err != nil {
			return nil, fmt.Errorf("decoding %s: %w", c.URL, err)
		}
	}

	Logf("WRITING: %s", c.Path)
	if err = os.WriteFile(c.Path, buf, 0o644); err != nil {
		return nil, err
	}

	return buf, nil
}

// Retrieval limits.
const (
	// fetchAttempts is the number of times fetch requests a URL.
	fetchAttempts = 4
	// fetchTimeout is the time limit for one request.
	fetchTimeout = time.Minute
)

// fetchDelay is the time to wait before the first retry. Each later retry
// waits twice as long as the one before. It is a variable so that tests can
// shorten it.
var fetchDelay = 2 * time.Second

// StatusError is the error for a response whose status is not 200.
type StatusError struct {
	URL    string
	Status int
}

// Error satisfies the error interface.
func (err *StatusError) Error() string {
	return fmt.Sprintf("retrieving %s: status %d", err.URL, err.Status)
}

// fetch retrieves the URL. It retries a network error, a 429 status and a 5xx
// status, and it returns a [StatusError] for any other status that is not 200.
func fetch(urlstr string) ([]byte, error) {
	cl := &http.Client{Timeout: fetchTimeout}
	delay := fetchDelay
	var err error
	for i := range fetchAttempts {
		if i != 0 {
			Logf("RETRYING: %s (%v)", urlstr, err)
			time.Sleep(delay)
			delay *= 2
		}
		Logf("RETRIEVING: %s", urlstr)
		var buf []byte
		var retry bool
		if buf, retry, err = fetchOnce(cl, urlstr); err == nil {
			return buf, nil
		}
		if !retry {
			break
		}
	}
	return nil, err
}

// fetchOnce makes one request. It reports whether a failure is worth a retry.
func fetchOnce(cl *http.Client, urlstr string) ([]byte, bool, error) {
	req, err := http.NewRequest(http.MethodGet, urlstr, nil)
	if err != nil {
		return nil, false, err
	}
	res, err := cl.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("retrieving %s: %w", urlstr, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		retry := res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500
		return nil, retry, &StatusError{URL: urlstr, Status: res.StatusCode}
	}
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, true, fmt.Errorf("reading %s: %w", urlstr, err)
	}
	return buf, false, nil
}
