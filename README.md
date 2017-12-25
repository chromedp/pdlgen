# About chromedp-gen

`chromedp-gen` is a standalone tool, built for the [chromedp][1] project, that
generates the commands, events, and types for the [Chrome Debugging Protocol
domains][2].

`chromedp-gen` works by applying [Go code templates](/templates) to the CDP
domains defined in the [`browser_protocol.json`][3] and [`js_protocol.json`][4]
files available in the Chromium source tree, generating (by default) the
[`github.com/chromedp/cdproto`][5] package and sub-packages.

Please note that any Issues or Pull Requests for the `cdproto` project should
instead be created on this project, and **NOT** on the `cdproto` project.

## Installing

`chromedp-gen` uses the [qtc][6], [easyjson][7], and [goimports][8]
tools, for generating the templated CDP-domain code, generating fast JSON
marshaler/unmarshalers, and fixing missing imports in the generated code,
respectively.

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

## Using chromedp-gen

`chromedp-gen` has sensible default options, and should be usable out-of-the-box:

```sh
# standard generation
$ chromedp-gen
2017/12/25 12:14:01 BROWSER: master
2017/12/25 12:14:01 JS:      master
2017/12/25 12:14:01 RETRIEVING: https://chromium.googlesource.com/chromium/src/+/master/third_party/WebKit/Source/core/inspector/browser_protocol.json?format=TEXT
2017/12/25 12:14:01 WROTE: /home/ken/src/go/pkg/chromedp-gen/browser/master/browser_protocol.json
2017/12/25 12:14:01 RETRIEVING: https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.json?format=TEXT
2017/12/25 12:14:01 WROTE: /home/ken/src/go/pkg/chromedp-gen/js/master/js_protocol.json
2017/12/25 12:14:01 SKIPPING(domain ): Console [deprecated]
2017/12/25 12:14:01 SKIPPING(command): DOM.hideHighlight [redirect:Overlay]
2017/12/25 12:14:01 SKIPPING(command): DOM.highlightNode [redirect:Overlay]
2017/12/25 12:14:01 SKIPPING(command): DOM.highlightRect [redirect:Overlay]
2017/12/25 12:14:01 SKIPPING(command): Emulation.setVisibleSize [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Network.canClearBrowserCache [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Network.canClearBrowserCookies [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Network.canEmulateNetworkConditions [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.addScriptToEvaluateOnLoad [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.clearDeviceMetricsOverride [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.clearDeviceOrientationOverride [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.clearGeolocationOverride [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.deleteCookie [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.getCookies [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.removeScriptToEvaluateOnLoad [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.setDeviceMetricsOverride [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.setDeviceOrientationOverride [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.setGeolocationOverride [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Page.setTouchEmulationEnabled [deprecated]
2017/12/25 12:14:01 SKIPPING(domain ): Schema [deprecated]
2017/12/25 12:14:01 SKIPPING(event  ): Security.certificateError [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Security.handleCertificateError [deprecated]
2017/12/25 12:14:01 SKIPPING(command): Security.setOverrideCertificateErrors [deprecated]
2017/12/25 12:14:01 SKIPPING(e param): Target.detachedFromTarget.targetId [deprecated]
2017/12/25 12:14:01 SKIPPING(e param): Target.receivedMessageFromTarget.targetId [deprecated]
2017/12/25 12:14:01 SKIPPING(c param): Target.detachFromTarget.targetId [deprecated]
2017/12/25 12:14:01 SKIPPING(c param): Target.sendMessageToTarget.targetId [deprecated]
2017/12/25 12:14:01 SKIPPING(c param): Tracing.start.categories [deprecated]
2017/12/25 12:14:01 SKIPPING(c param): Tracing.start.options [deprecated]
2017/12/25 12:14:01 CLEANING: /home/ken/src/go/src/github.com/chromedp/cdproto
2017/12/25 12:14:01 WRITING: 101 files
2017/12/25 12:14:01 WRITING: protocol.json
2017/12/25 12:14:01 RUNNING: goimports
2017/12/25 12:14:03 RUNNING: easyjson (stubs)
2017/12/25 12:14:03 RUNNING: easyjson
2017/12/25 12:14:09 RUNNING: gofmt
2017/12/25 12:14:09 done.
```

### Protocol Retrieval, Caching, and Options

`chromedp-gen` can be passed a single, combined protocol file via the `-proto`
command-line option for generating the commands, events, and types for the
Chromium Debugging Protocol domains. If the `-proto` option is not specified
(the default behavior), then the `browser_protocol.json` and `js_protocol.json`
protocol definition files will be retrieved from the [Chromium source tree][9]
and cached locally.

The revisions of `browser_protocol.json` and `js_protocol.json` that are
retrieved/cached can be controlled using the `-browser` and `-js` command-line
options, respectively, and can be any Git ref, branch, or tag in the Chromium
source tree. Both default to `master`.

Both `browser_protocol.json` and `js_protocol.json` will be updated
periodically after the cached files have "expired", based on the `-ttl` option.
A `-ttl=0` forces retrieving and caching the files immediately. By default, the
`-ttl` option has a value of 24 hours.

Additionally, a meta-protocol definition file containing the virtual `HAR`
domain is generated [from the HAR spec][10] and cached (similarly to the above)
as `har.json`. However, since the HAR definition is frozen, the retrieval and
caching is controlled separately by the command-line option `-ttlHar`. A
`-ttlHar=0` indicates never to regenerate the `har.json` (the default value).

The `browser_protocol.json`, `js_protocol.json`, and `har.json` files are
cached in the `$GOPATH/pkg/chromedp-gen` directory by default, and can be
changed by specifying the `-cache` option.

#### Command-line Options

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

[1]: https://github.com/chromedp
[2]: https://chromedevtools.github.io/devtools-protocol/
[3]: https://chromium.googlesource.com/chromium/src/+/master/third_party/WebKit/Source/core/inspector/browser_protocol.json
[4]: https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.json
[5]: https://github.com/chromedp/cdproto
[6]: https://github.com/valyala/quicktemplate
[7]: https://github.com/mailru/easyjson
[8]: https://golang.org/x/tools/cmd/goimports
[9]: https://chromium.googlesource.com/chromium/src.git
[10]: http://www.softwareishard.com/blog/har-12-spec/
