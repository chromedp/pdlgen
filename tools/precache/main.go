// Command precache downloads all available Chrome DevTools Protocol definition
// files (PDLs) using pdlcache.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/pdlgen/util/pdlcache"
)

func main() {
	cacheDir := flag.String("cache-dir", "", "protocol cache directory")
	ttl := flag.Duration("ttl", 24*time.Hour, "ttl")
	osstr := flag.String("os", "win64", "platform")
	channel := flag.String("channel", "canary", "channel")
	flag.Parse()
	if err := run(*cacheDir, *ttl, *osstr, *channel); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(cacheDir string, ttl time.Duration, osstr, channel string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := pdlcache.New(
		pdlcache.WithCacheDir(cacheDir),
		pdlcache.WithTTL(ttl),
		pdlcache.WithLogf(log.Printf),
	)
	if err != nil {
		return err
	}
	recent, err := c.OmahaProxyRecent(ctx)
	if err != nil {
		return err
	}
	for _, ver := range recent {
		if ver.OS != osstr || ver.Channel != channel {
			continue
		}
		v8ver, err := c.BrowserV8Version(ctx, ver.Version)
		if err != nil {
			continue
		}
		log.Printf("BROWSER: %s (%s, %s) => V8 %s", ver.Version, ver.OS, ver.Channel, v8ver)
		if _, err := c.CombinedDef(ctx, ver.Version, v8ver); err != nil {
			return err
		}
	}
	return nil
}
