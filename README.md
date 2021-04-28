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
[`js_protocol.pdl`][js-protocol] files from the [Chromium source tree][chromium-src].
By default, these files are cached in the `<default root cache directory>/cdproto-gen`
directory and periodically updated (see below). The `<default root cache directory>`
is returned by [os.UserCacheDir()](https://pkg.go.dev/os#UserCacheDir).

Additionally, a [HAR definition][har-spec] will be used for generating a
special HAR domain.

### Code Generation

`cdproto-gen` works by applying [templates](/gen/gotpl) and [fixup](/fixup)
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
domain. The tool has sensible default options, but the `-out` option should be
specified, and points to the directory into which the [`github.com/chromedp/cdproto`][cdproto]
repository is cloned:

```sh
$ ./cdproto-gen -chromium=92.0.4501.1 -v8=9.2.173 -out=../cdproto
2021/05/18 13:39:42 RETRIEVING: https://chromium.googlesource.com/chromium/src/+/92.0.4501.1/third_party/blink/public/devtools_protocol/browser_protocol.pdl?format=TEXT
2021/05/18 13:39:42 WRITING: /home/ken/.cache/cdproto-gen/pdl/chromium/92.0.4501.1.pdl
2021/05/18 13:39:42 RETRIEVING: https://chromium.googlesource.com/v8/v8/+/9.2.173/include/js_protocol.pdl?format=TEXT
2021/05/18 13:39:43 WRITING: /home/ken/.cache/cdproto-gen/pdl/v8/9.2.173.pdl
2021/05/18 13:39:43 WRITING: /home/ken/.cache/cdproto-gen/pdl/combined/92.0.4501.1_9.2.173.pdl
2021/05/18 13:39:43 SKIPPING(domain ): Console [deprecated]
...
2021/05/18 13:39:43 CLEANING: /home/ken/src/chromedp/cdproto
2021/05/18 13:39:43 WRITING: 123 files
2021/05/18 13:39:43 RUNNING: goimports
2021/05/18 13:39:43 RUNNING: easyjson
2021/05/18 13:39:51 RUNNING: gofmt
2021/05/18 13:39:51 done.
```

### Command-line options

`cdproto-gen` can be passed a single, combined protocol file via the `-pdl`
command-line option for generating the commands, events, and types for the
Chromium DevTools Protocol domains. If the `-pdl` option is not specified
(the default behavior), then the `browser_protocol.pdl` and `js_protocol.pdl`
protocol definition files will be retrieved from the [Chromium source
tree][chromium-src] and cached locally.

The revisions of `browser_protocol.pdl` and `js_protocol.pdl` that are
retrieved/cached can be controlled using the `-chromium` and `-v8` command-line
options, respectively, and can be any Git ref, branch, or tag in the [Chromium
source tree][chromium-src]. Both default to `master`.

Both `browser_protocol.pdl` and `js_protocol.pdl` will be updated
periodically after the cached files have "expired", based on the `-ttl` option.
Specifying `-ttl=0` forces retrieving and caching the files immediately. By
default, the `-ttl` option has a value of 24 hours.

The `browser_protocol.pdl` and `js_protocol.pdl` files are cached in the
`<default root cache directory>/pkg/cdproto-gen` directory by default, and can
be changed by specifying the `-cache` option.

The `-out` command-line option should point to the directory into which the
[`github.com/chromedp/cdproto`][cdproto] repository is cloned, so that easyjson
can find the `go.mod` file and use it.

Additional command-line options are also available:

```sh
$ cdproto-gen --help
Usage of ./cdproto-gen:
  -cache string
    	protocol cache directory
  -chromium string
    	chromium protocol version
  -debug
    	toggle debug (writes generated files to disk without post-processing)
  -go-pkg string
    	go base package name (default "github.com/chromedp/cdproto")
  -go-wl string
    	comma-separated list of files to whitelist (ignore) (default "LICENSE,README.md,*.pdl,go.mod,go.sum,easyjson.go")
  -latest
    	use latest protocol
  -no-clean
    	toggle not cleaning (removing) existing directories
  -no-dump
    	toggle not dumping generated protocol file to out directory
  -out string
    	package out directory
  -pdl string
    	path to pdl file to use
  -ttl duration
    	file retrieval caching ttl (default 24h0m0s)
  -v8 string
    	v8 protocol version
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
`go generate` in the root directory of this repository, and rebuild/run `./cdproto-gen`:

```sh
$ go generate && go build && ./cdproto-gen -chromium=92.0.4501.1 -v8=9.2.173 -out=../cdproto
qtc: 2021/05/18 14:11:20 Compiling *.qtpl template files in directory "gen/gotpl"
qtc: 2021/05/18 14:11:20 Compiling "gen/gotpl/domain.qtpl" to "gen/gotpl/domain.qtpl.go"...
qtc: 2021/05/18 14:11:20 Compiling "gen/gotpl/extra.qtpl" to "gen/gotpl/extra.qtpl.go"...
qtc: 2021/05/18 14:11:20 Compiling "gen/gotpl/file.qtpl" to "gen/gotpl/file.qtpl.go"...
qtc: 2021/05/18 14:11:20 Compiling "gen/gotpl/type.qtpl" to "gen/gotpl/type.qtpl.go"...
qtc: 2021/05/18 14:11:20 Total files compiled: 4
2021/05/18 14:11:20 RETRIEVING: https://chromium.googlesource.com/chromium/src/+/92.0.4501.1/third_party/blink/public/devtools_protocol/browser_protocol.pdl?format=TEXT
2021/05/18 14:11:21 WRITING: /home/ken/.cache/cdproto-gen/pdl/chromium/92.0.4501.1.pdl
2021/05/18 14:11:21 RETRIEVING: https://chromium.googlesource.com/v8/v8/+/9.2.173/include/js_protocol.pdl?format=TEXT
2021/05/18 14:11:21 WRITING: /home/ken/.cache/cdproto-gen/pdl/v8/9.2.173.pdl
2021/05/18 14:11:21 WRITING: /home/ken/.cache/cdproto-gen/pdl/combined/92.0.4501.1_9.2.173.pdl
2021/05/18 14:11:21 SKIPPING(domain ): Console [deprecated]
2021/05/18 14:11:21 SKIPPING(t property): DOM.Node.importedDocument [deprecated]
2021/05/18 14:11:21 SKIPPING(command): DOM.getFlattenedDocument [deprecated]
2021/05/18 14:11:21 SKIPPING(command): DOM.hideHighlight [redirect:Overlay.hideHighlight]
2021/05/18 14:11:21 SKIPPING(command): DOM.highlightNode [redirect:Overlay.highlightNode]
2021/05/18 14:11:21 SKIPPING(command): DOM.highlightRect [redirect:Overlay.highlightRect]
2021/05/18 14:11:21 SKIPPING(command): DOMSnapshot.getSnapshot [deprecated]
2021/05/18 14:11:21 SKIPPING(e param): Debugger.paused.asyncCallStackTraceId [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Debugger.getWasmBytecode [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Debugger.pauseOnAsyncCall [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Debugger.restartFrame [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Emulation.setNavigatorOverrides [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Emulation.setVisibleSize [deprecated]
2021/05/18 14:11:21 SKIPPING(event  ): HeadlessExperimental.needsBeginFramesChanged [deprecated]
2021/05/18 14:11:21 SKIPPING(c return param): LayerTree.compositingReasons.compositingReasons [deprecated]
2021/05/18 14:11:21 SKIPPING(event  ): Network.requestIntercepted [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Network.canClearBrowserCache [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Network.canClearBrowserCookies [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Network.canEmulateNetworkConditions [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Network.continueInterceptedRequest [deprecated]
2021/05/18 14:11:21 SKIPPING(c return param): Network.setCookie.success [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Network.setRequestInterception [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Network.setUserAgentOverride [redirect:Emulation]
2021/05/18 14:11:21 SKIPPING(t property): Overlay.GridHighlightConfig.cellBorderColor [deprecated]
2021/05/18 14:11:21 SKIPPING(t property): Overlay.GridHighlightConfig.cellBorderDash [deprecated]
2021/05/18 14:11:21 SKIPPING(event  ): Page.frameClearedScheduledNavigation [deprecated]
2021/05/18 14:11:21 SKIPPING(event  ): Page.frameScheduledNavigation [deprecated]
2021/05/18 14:11:21 SKIPPING(event  ): Page.downloadWillBegin [deprecated]
2021/05/18 14:11:21 SKIPPING(event  ): Page.downloadProgress [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.addScriptToEvaluateOnLoad [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.clearDeviceMetricsOverride [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.clearDeviceOrientationOverride [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.clearGeolocationOverride [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.deleteCookie [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.getCookies [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.removeScriptToEvaluateOnLoad [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.setDeviceMetricsOverride [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.setDeviceOrientationOverride [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.setGeolocationOverride [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Page.setTouchEmulationEnabled [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Performance.setTimeDomain [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Runtime.setAsyncCallStackDepth [redirect:Debugger]
2021/05/18 14:11:21 SKIPPING(c param): Runtime.addBinding.executionContextId [deprecated]
2021/05/18 14:11:21 SKIPPING(domain ): Schema [deprecated]
2021/05/18 14:11:21 SKIPPING(type   ): Security.InsecureContentStatus [deprecated]
2021/05/18 14:11:21 SKIPPING(event  ): Security.certificateError [deprecated]
2021/05/18 14:11:21 SKIPPING(e param): Security.securityStateChanged.schemeIsCryptographic [deprecated]
2021/05/18 14:11:21 SKIPPING(e param): Security.securityStateChanged.insecureContentStatus [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Security.handleCertificateError [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Security.setOverrideCertificateErrors [deprecated]
2021/05/18 14:11:21 SKIPPING(e param): Target.detachedFromTarget.targetId [deprecated]
2021/05/18 14:11:21 SKIPPING(e param): Target.receivedMessageFromTarget.targetId [deprecated]
2021/05/18 14:11:21 SKIPPING(c return param): Target.closeTarget.success [deprecated]
2021/05/18 14:11:21 SKIPPING(c param): Target.detachFromTarget.targetId [deprecated]
2021/05/18 14:11:21 SKIPPING(command): Target.sendMessageToTarget [deprecated]
2021/05/18 14:11:21 SKIPPING(c param): Tracing.start.categories [deprecated]
2021/05/18 14:11:21 SKIPPING(c param): Tracing.start.options [deprecated]
2021/05/18 14:11:21 CLEANING: /home/ken/src/chromedp/cdproto
2021/05/18 14:11:21 WRITING: 123 files
2021/05/18 14:11:21 RUNNING: goimports
2021/05/18 14:11:22 RUNNING: easyjson
2021/05/18 14:11:30 RUNNING: gofmt
2021/05/18 14:11:30 done.
```

[devtools-protocol]: https://chromedevtools.github.io/devtools-protocol/
[chromedp]: https://github.com/chromedp
[cdproto]: https://github.com/chromedp/cdproto
[browser-protocol]: https://chromium.googlesource.com/chromium/src/+/master/third_party/blink/public/devtools_protocol/browser_protocol.pdl
[js-protocol]: https://chromium.googlesource.com/v8/v8/+/master/include/js_protocol.pdl
[chromium-src]: https://chromium.googlesource.com/chromium/src.git
[har-spec]: http://www.softwareishard.com/blog/har-12-spec/
[effective-go]: https://golang.org/doc/effective_go.html
[cdproto-godoc]: https://godoc.org/github.com/chromedp/cdproto
[quicktemplate]: https://github.com/valyala/quicktemplate
