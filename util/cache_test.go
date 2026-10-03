package util

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	fetchDelay = time.Millisecond
	t.Cleanup(func() { fetchDelay = 2 * time.Second })
	encoded := base64.StdEncoding.EncodeToString([]byte("domain Test\n"))
	tests := []struct {
		name     string
		statuses []int
		body     string
		decode   bool
		want     string
		status   int
		calls    int32
	}{
		{"ok", []int{200}, encoded, true, "domain Test\n", 0, 1},
		{"not found is not retried", []int{404}, "<html>", true, "", 404, 1},
		{"server error is retried", []int{503, 502, 200}, encoded, true, "domain Test\n", 0, 3},
		{"too many requests is retried", []int{429, 200}, encoded, true, "domain Test\n", 0, 2},
		{"gives up", []int{500, 500, 500, 500, 500}, "", true, "", 500, fetchAttempts},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				i := int(calls.Add(1)) - 1
				status := test.statuses[min(i, len(test.statuses)-1)]
				w.WriteHeader(status)
				if status == http.StatusOK {
					_, _ = w.Write([]byte(test.body))
				} else {
					_, _ = w.Write([]byte("<html>error page</html>"))
				}
			}))
			defer srv.Close()
			path := filepath.Join(t.TempDir(), "a", "file.pdl")
			buf, err := Get(Cache{URL: srv.URL, Path: path, TTL: time.Hour, Decode: test.decode})
			if got := calls.Load(); got != test.calls {
				t.Errorf("expected %d requests, got: %d", test.calls, got)
			}
			if test.status != 0 {
				var serr *StatusError
				if !errors.As(err, &serr) || serr.Status != test.status {
					t.Fatalf("expected status error %d, got: %v", test.status, err)
				}
				if _, err := os.Stat(path); err == nil {
					t.Errorf("expected nothing to be cached after a failure")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if string(buf) != test.want {
				t.Errorf("expected %q, got: %q", test.want, buf)
			}
			if cached, err := os.ReadFile(path); err != nil || string(cached) != test.want {
				t.Errorf("expected %q in the cache, got: %q, %v", test.want, cached, err)
			}
		})
	}
}
