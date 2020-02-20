# pdlgen

`pdlgen` generates Go code for the commands, events, and types for the [Chrome
DevTools Protocol][devtools-protocol] and is a core component of the
[`chromedp`][chromedp] project. While `pdlgen`'s development is primarily
driven by the needs of the `chromedp` project, the aim of this project is to
provide a easy way to generate code for various Chrome DevTools projects based
on Chromium's PDL definitions.

At the moment, `pdlgen` currently builds [type-safe, fast, efficient, idiomatic
Go][cdproto] code for [`cdproto`][cdproto], with plans to provide a simple
harness/framework for other projects to include their templates with this tool.

**NOTE:** Any Issue or Pull Request intended for the `cdproto` project should
be created here, and **NOT** on the `cdproto` project.

## How it Works

`pdlgen` retrieves the [`browser_protocol.pdl`][browser-protocol] and
[`js_protocol.pdl`][js-protocol] files from the [Chromium source
tree][chromium-src] By default, these files are cached in the in `<user's cache directory>/pdlgen/`
(configurable through the `-cache-dir` flag)

## Installing

`pdlgen` is installed in the usual Go way:

```sh
$ go get -u github.com/chromedp/pdlgen
```

## Using

By default, `chromdep-gen` generates the [`github.com/chromedp/cdproto`][cdproto-godoc]
package and a `github.com/chromedp/cdproto/<domain>` package for each CDP
domain. The tool has sensible default options, and should be usable
out-of-the-box:

```sh
$ pdlgen
2018/07/04 10:21:30 BROWSER: https://chromium.googlesource.com/chromium/src/+/master/third_party/blink/renderer/core/inspector/browser_protocol.pdl?format=TEXT
2018/07/04 10:21:30 JS     : https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.pdl?format=TEXT
2018/07/04 10:21:30 WRITING: /home/ken/src/go/src/github.com/chromedp/cdproto/protocol-master_master-20180704.pdl
2018/07/04 10:21:30 SKIPPING(domain ): Console [deprecated]
...
2018/07/04 10:21:30 CLEANING: /home/ken/src/go/src/github.com/chromedp/cdproto
2018/07/04 10:21:30 WRITING: 101 files
2018/07/04 10:21:30 RUNNING: goimports
2018/07/04 10:21:31 RUNNING: easyjson
2018/07/04 10:21:37 RUNNING: gofmt
2018/07/04 10:21:37 done.
```

### Command-line options

`pdlgen` can be passed a single, combined protocol file via the `-proto`
command-line option for generating the commands, events, and types for the
Chromium DevTools Protocol domains. If the `-proto` option is not specified
(the default behavior), then the `browser_protocol.pdl` and `js_protocol.pdl`
protocol definition files will be retrieved from the [Chromium source
tree][chromium-src] and cached locally.

The revisions of `browser_protocol.pdl` and `js_protocol.pdl` that are
retrieved/cached can be controlled using the `-browser` and `-js` command-line
options, respectively, and can be any Git ref, branch, or tag in the [Chromium
source tree][chromium-src]. Both default to `master`.

Both `browser_protocol.pdl` and `js_protocol.pdl` will be updated
periodically after the cached files have "expired", based on the `-ttl` option.
Specifying `-ttl=0` forces retrieving and caching the files immediately. By
default, the `-ttl` option has a value of 24 hours.

The `browser_protocol.pdl` and `js_protocol.pdl` files are cached in the
`$GOPATH/pkg/pdlgen` directory by default, and can be changed by
specifying the `-cache-dir` option.

Additional command-line options are also available:

```sh
$ pdlgen --help
Usage of ./pdlgen:
```

## Code Generators

The following are notes about each of the `pdlgen` code generators.

### Generator `cdproto`

The [`cdproto` generator](gen/cdproto) uses [`quicktemplate`]

`pdlgen` works by applying [templates](/templates) and [fixups](/fixups)
(such as spelling corrections that assist with generating [idiomatic Go][effective-go])
to the CDP domains defined in `browser_protocol.pdl` and `js_protocol.pdl`.
From the protocol definitions, `pdlgen` generates the [`github.com/chromedp/cdproto`][cdproto]
package and a `github.com/chromedp/cdproto/<domain>` subpackage for each
domain. CDP types that have circular dependencies are placed in the
`github.com/chromedp/cdproto/cdp` package.


Additionally, a [HAR definition][har-spec] will be used for generating a
special HAR domain.

## Working with Templates

[devtools-protocol]: https://chromedevtools.github.io/devtools-protocol/
[chromedp]: https://github.com/chromedp
[cdproto]: https://github.com/chromedp/cdproto
[browser-protocol]: https://chromium.googlesource.com/chromium/src/+/master/third_party/blink/renderer/core/inspector/browser_protocol.pdl
[js-protocol]: https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.pdl
[chromium-src]: https://chromium.googlesource.com/chromium/src.git
[har-spec]: http://www.softwareishard.com/blog/har-12-spec/
[effective-go]: https://golang.org/doc/effective_go.html
[cdproto-godoc]: https://godoc.org/github.com/chromedp/cdproto
[quicktemplate]: https://github.com/valyala/quicktemplate
