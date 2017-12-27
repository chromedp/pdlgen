# chromedp-gen

`chromedp-gen` generates Go code for the commands, events, and types for the
[Chrome Debugging Protocol][1] and is a core component of the [`chromedp`][2]
project. While `chromedp-gen`'s development is primarily driven by the needs of
the `chromedp` project, the aim of this project is to generate [type-safe,
fast, efficient, idiomatic Go code][3] usable by any Go application wishing to
drive Chrome through the CDP.

### Code Generation Overview

`chromedp-gen` works by applying [code templates][4] to the CDP domains,
commands, events, and types defined in the [`browser_protocol.json`][5] and
[`js_protocol.json`][6] files taken from the [Chromium source tree][7]. A
[number of "fixups" (such as correcting spelling mistakes)][8] are applied to
the CDP domain definitions so that generated code can be [Go-idiomatic][9].

`chromedp-gen` generates the [`github.com/chromedp/cdproto`][10] package and
constituent domain subpackages. As such, any Issue or Pull Request for the
`cdproto` project should be created here, and **NOT** on the `cdproto` project.

## Installing

`chromedp-gen` uses the [`qtc`][11], [`easyjson`][12], and [`goimports`][13]
tools to generate the templated CDP-domain code, fast JSON marshaler/unmarshalers,
and to fix missing imports in the generated code, respectively.

`chromedp-gen` expects these tools to be somewhere on your `$PATH`. Please
ensure that `$GOPATH/bin` is on your `$PATH`, and then install these tools and
their associated dependencies in the usual Go way:

```sh
$ go get -u \
    github.com/valyala/quicktemplate/qtc \
    github.com/mailru/easyjson/easyjson \
    golang.org/x/tools/cmd/goimports
```

`chromedp-gen` can then be installed in the usual Go fashion:

```sh
$ go get -u github.com/chromedp/chromedp-gen
```

## Using

By default, `chromdep-gen` generates the [`github.com/chromedp/cdproto`][6]
package and a `github.com/chromedp/cdproto/<domain>` package for each CDP
domain. The tool has sensible default options, and should be usable
out-of-the-box:

```sh
$ chromedp-gen
2017/12/25 12:14:01 BROWSER: master
2017/12/25 12:14:01 JS:      master
2017/12/25 12:14:01 RETRIEVING: https://chromium.googlesource.com/chromium/src/+/master/third_party/WebKit/Source/core/inspector/browser_protocol.json?format=TEXT
2017/12/25 12:14:01 WROTE: /home/ken/src/go/pkg/chromedp-gen/browser/master/browser_protocol.json
2017/12/25 12:14:01 RETRIEVING: https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.json?format=TEXT
2017/12/25 12:14:01 WROTE: /home/ken/src/go/pkg/chromedp-gen/js/master/js_protocol.json
2017/12/25 12:14:01 SKIPPING(domain ): Console [deprecated]
...
2017/12/25 12:14:01 CLEANING: /home/ken/src/go/src/github.com/chromedp/cdproto
2017/12/25 12:14:01 WRITING: 101 files
2017/12/25 12:14:01 WRITING: protocol.json
2017/12/25 12:14:01 RUNNING: goimports
2017/12/25 12:14:03 RUNNING: easyjson (stubs)
2017/12/25 12:14:03 RUNNING: easyjson
2017/12/25 12:14:09 RUNNING: gofmt
2017/12/25 12:14:09 done.
```

### Protocol Definition Retrieval and Caching

`chromedp-gen` downloads the `browser_protocol.json` and `js_protocol.json`
files directly from the [Chromium source tree][7], and generates a `har.json`
protocol defintion [from the HAR spec][14]. By default, these files are cached
in the `$GOPATH/pkg/chromedp-gen` directory and periodically updated.

### Command-line Options

`chromedp-gen` can be passed a single, combined protocol file via the `-proto`
command-line option for generating the commands, events, and types for the
Chromium Debugging Protocol domains. If the `-proto` option is not specified
(the default behavior), then the `browser_protocol.json` and `js_protocol.json`
protocol definition files will be retrieved from the [Chromium source tree][7]
and cached locally.

The revisions of `browser_protocol.json` and `js_protocol.json` that are
retrieved/cached can be controlled using the `-browser` and `-js` command-line
options, respectively, and can be any Git ref, branch, or tag in the [Chromium
source tree][7]. Both default to `master`.

Both `browser_protocol.json` and `js_protocol.json` will be updated
periodically after the cached files have "expired", based on the `-ttl` option.
Specifying `-ttl=0` forces retrieving and caching the files immediately. By
default, the `-ttl` option has a value of 24 hours.

The meta-protocol definition file containing the virtual `HAR` domain is
generated from [the HAR spec][14] and cached (similarly to the above) as
`har.json`. Since the HAR definition is frozen, the retrieval and caching is
controlled separately with the command-line option `-ttlHar`. A `-ttlHar=0`
indicates never to regenerate the `har.json` and is the default value.

The `browser_protocol.json`, `js_protocol.json`, and `har.json` files are
cached in the `$GOPATH/pkg/chromedp-gen` directory by default, and can be
changed by specifying the `-cache` option.

The following command-line options are available:

```sh
$ chromedp-gen --help
Usage of ./chromedp-gen:
  -browser string
    	browser protocol version to use (default "master")
  -cache string
    	protocol cache directory (default "/home/ken/src/go/pkg/chromedp-gen")
  -dep
    	toggle generation for deprecated APIs
  -exp
    	toggle generation for experimental APIs (default true)
  -js string
    	js protocol version to use (default "master")
  -noclean
    	toggle not cleaning (removing) existing directories
  -nocopy
    	toggle not copying combined protocol.json to out directory
  -nohar
    	toggle not generating HAR protocol and domain
  -out string
    	out directory
  -pkg string
    	out base package (default "github.com/chromedp/cdproto")
  -proto string
    	protocol.json path
  -redirect
    	toggle generation for redirect APIs
  -ttl duration
    	browser and js protocol cache ttl (default 24h0m0s)
  -ttlHar duration
    	har cache ttl
  -v	toggle verbose (default true)
  -wl string
    	comma-separated list of files to whitelist/ignore during clean (default "LICENSE,README.md,protocol.json,easyjson.go")
  -workers int
    	number of workers (default 9)
```
[1]: https://chromedevtools.github.io/devtools-protocol/
[2]: https://github.com/chromedp
[3]: https://godoc.org/github.com/chromedp/cdproto
[4]: /templates
[5]: https://chromium.googlesource.com/chromium/src/+/master/third_party/WebKit/Source/core/inspector/browser_protocol.json
[6]: https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.json
[7]: https://chromium.googlesource.com/chromium/src.git
[8]: /fixup
[9]: https://golang.org/doc/effective_go.html
[10]: https://github.com/chromedp/cdproto
[11]: https://github.com/valyala/quicktemplate
[12]: https://github.com/mailru/easyjson
[13]: https://golang.org/x/tools/cmd/goimports
[14]: http://www.softwareishard.com/blog/har-12-spec/
