# cdproto-gen

`cdproto-gen` generates Go code for the commands, events, and types for the
[Chrome DevTools Protocol][devtools-protocol] and is a core component of the
[`chromedp`][chromedp] project. While `cdproto-gen`'s development is primarily
driven by the needs of the `chromedp` project, the aim of this project is to
generate [type-safe, fast, efficient, idiomatic Go code][cdproto] usable by any
Go application needing to drive Chrome through the CDP.

**NOTE:** Any Issue or Pull Request intended for the `cdproto` project should
be created here, and **NOT** on the `cdproto` project.

### Protocol Definition Retrieval and Caching

`cdproto-gen` retrieves the [`browser_protocol.pdl`][browser-protocol] and
[`js_protocol.pdl`][js-protocol] files from the [Chromium source tree][chromium-src]
By default, these files are cached in the `$GOPATH/pkg/cdproto-gen` directory
and periodically updated (see below).

Additionally, a [HAR definition][har-spec] will be used for generating a
special HAR domain.

### Code Generation

`cdproto-gen` works by applying [templates](/templates) and [fixups](/fixups)
(such as spelling corrections that assist with generating [idiomatic Go][effective-go])
to the CDP domains defined in `browser_protocol.pdl` and `js_protocol.pdl`.
From the protocol definitions, `cdproto-gen` generates the [`github.com/chromedp/cdproto`][cdproto]
package and a `github.com/chromedp/cdproto/<domain>` subpackage for each
domain. CDP types that have circular dependencies are placed in the
`github.com/chromedp/cdproto/cdp` package.

## Installing

`cdproto-gen` is installed in the usual Go way:

```sh
$ go get -u github.com/chromedp/cdproto-gen
```

## Using

By default, `chromdep-gen` generates the [`github.com/chromedp/cdproto`][cdproto-godoc]
package and a `github.com/chromedp/cdproto/<domain>` package for each CDP
domain. The tool has sensible default options, and should be usable
out-of-the-box:

```sh
$ cdproto-gen
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

`cdproto-gen` can be passed a single, combined protocol file via the `-proto`
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
`$GOPATH/pkg/cdproto-gen` directory by default, and can be changed by
specifying the `-cache` option.

Additional command-line options are also available:

```sh
$ cdproto-gen --help
Usage of ./cdproto-gen:
  -browser string
    	browser version to retrieve/use (default "master")
  -cache string
    	protocol cache directory (default "/home/ken/src/go/pkg/cdproto-gen")
  -debug
    	toggle debug (writes generated files to disk without post-processing)
  -go-pkg string
    	go base package name (default "github.com/chromedp/cdproto")
  -go-wl string
    	comma-separated list of files to whitelist (ignore) (default "LICENSE,README.md,protocol*.pdl,easyjson.go")
  -js string
    	js version to retrieve/use (default "master")
  -no-clean
    	toggle not cleaning (removing) existing directories
  -no-dump
    	toggle not dumping generated protocol file to out directory
  -out string
    	out directory
  -pdl string
    	path to pdl file to use
  -ttl duration
    	browser and js cache ttl (default 24h0m0s)
  -workers int
    	number of workers (default 8)
```

## Working with Templates

`cdproto-gen`'s code generation makes use of  [`quicktemplate`][quicktemplate]
templates. As such, in order to modify the templates, the `qtc` template
compiler needs to be available on `$PATH`.

`qtc` can be installed in the usual Go fashion:

```sh
$ go get -u github.com/valyala/quicktemplate/qtc
```

After modifying the `gen/gotpl/*.qtpl` files, `qtc` needs to be run. Simply run
`go generate` in the `$GOPATH/src/github.com/chromedp/cdproto-gen` directory,
and rebuild/run `cdproto-gen`:

```sh
$ cd $GOPATH/src/github.com/chromedp/cdproto-gen
$ go generate && go build && ./cdproto-gen
qtc: 2018/07/04 10:21:30 Compiling *.qtpl template files in directory "gen/gotpl"
qtc: 2018/07/04 10:21:30 Compiling "gen/gotpl/domain.qtpl" to "gen/gotpl/domain.qtpl.go"...
qtc: 2018/07/04 10:21:30 Compiling "gen/gotpl/extra.qtpl" to "gen/gotpl/extra.qtpl.go"...
qtc: 2018/07/04 10:21:30 Compiling "gen/gotpl/file.qtpl" to "gen/gotpl/file.qtpl.go"...
qtc: 2018/07/04 10:21:30 Compiling "gen/gotpl/type.qtpl" to "gen/gotpl/type.qtpl.go"...
qtc: 2018/07/04 10:21:30 Total files compiled: 4
2018/07/04 10:21:30 BROWSER: https://chromium.googlesource.com/chromium/src/+/master/third_party/blink/renderer/core/inspector/browser_protocol.pdl?format=TEXT
2018/07/04 10:21:30 JS     : https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.pdl?format=TEXT
2018/07/04 10:21:30 WRITING: /home/ken/src/go/src/github.com/chromedp/cdproto/protocol-master_master-20180704.pdl
2018/07/04 10:21:30 SKIPPING(domain ): Console [deprecated]
2018/07/04 10:21:30 SKIPPING(command): DOM.hideHighlight [redirect:Overlay.hideHighlight]
2018/07/04 10:21:30 SKIPPING(command): DOM.highlightNode [redirect:Overlay.highlightNode]
2018/07/04 10:21:30 SKIPPING(command): DOM.highlightRect [redirect:Overlay.highlightRect]
2018/07/04 10:21:30 SKIPPING(command): DOMSnapshot.getSnapshot [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Emulation.setNavigatorOverrides [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Emulation.setVisibleSize [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Network.canClearBrowserCache [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Network.canClearBrowserCookies [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Network.canEmulateNetworkConditions [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Network.setUserAgentOverride [redirect:Emulation]
2018/07/04 10:21:30 SKIPPING(command): Page.addScriptToEvaluateOnLoad [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.clearDeviceMetricsOverride [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.clearDeviceOrientationOverride [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.clearGeolocationOverride [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.deleteCookie [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.getCookies [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.removeScriptToEvaluateOnLoad [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.setDeviceMetricsOverride [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.setDeviceOrientationOverride [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.setGeolocationOverride [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Page.setTouchEmulationEnabled [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Runtime.setAsyncCallStackDepth [redirect:Debugger]
2018/07/04 10:21:30 SKIPPING(domain ): Schema [deprecated]
2018/07/04 10:21:30 SKIPPING(event  ): Security.certificateError [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Security.handleCertificateError [deprecated]
2018/07/04 10:21:30 SKIPPING(command): Security.setOverrideCertificateErrors [deprecated]
2018/07/04 10:21:30 SKIPPING(e param): Target.detachedFromTarget.targetId [deprecated]
2018/07/04 10:21:30 SKIPPING(e param): Target.receivedMessageFromTarget.targetId [deprecated]
2018/07/04 10:21:30 SKIPPING(c param): Target.detachFromTarget.targetId [deprecated]
2018/07/04 10:21:30 SKIPPING(c param): Target.sendMessageToTarget.targetId [deprecated]
2018/07/04 10:21:30 SKIPPING(c param): Tracing.start.categories [deprecated]
2018/07/04 10:21:30 SKIPPING(c param): Tracing.start.options [deprecated]
2018/07/04 10:21:30 CLEANING: /home/ken/src/go/src/github.com/chromedp/cdproto
2018/07/04 10:21:30 WRITING: 101 files
2018/07/04 10:21:30 RUNNING: goimports
2018/07/04 10:21:31 RUNNING: easyjson
2018/07/04 10:21:37 RUNNING: gofmt
2018/07/04 10:21:37 done.
```

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
