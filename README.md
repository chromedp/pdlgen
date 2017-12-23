# About chromedp-gen

`chromedp-gen` is a standalone tool for the [chromedp](https://github.com/chromedp)
project that generates the types for the [Chrome Debugging Protocol Domain APIs](https://chromedevtools.github.io/devtools-protocol/)
as defined in Chrome's `protocol.json`.

## Updating protocol.json

Run [`update.sh`](update.sh) to retrieve and combine the latest
[`browser_protocol.json`](https://chromium.googlesource.com/chromium/src/+/master/third_party/WebKit/Source/core/inspector/browser_protocol.json) and
[`js_protocol.json`](https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.json)
from the [Chromium source tree](https://chromium.googlesource.com/) into [`protocol.json`](protocol.json).

## Installing dependencies

`chromedp-gen` uses the [qtc](https://github.com/valyala/quicktemplate),
[easyjson](https://github.com/mailru/easyjson), and
[goimports](https://golang.org/x/tools/cmd/goimports) tools for generating
templates for the various CDP domains, for generating fast JSON
marshaler/unmarshalers, and for code formatting (and fixing missing imports),
respectively.

`chromedp-gen` expects these tools to be somewhere on your `$PATH`. Please
ensure that `$GOPATH/bin` is on your `$PATH`, and then install these tools (and
their associated dependencies) in the usual Go way:

```sh
$ go get -u \
    github.com/valyala/quicktemplate/qtc \
    github.com/mailru/easyjson/easyjson \
    golang.org/x/tools/cmd/goimports
```

## Generating types for chromedp

Assuming the `qtc`, `easyjson`, and `goimports` commands are available on
`$PATH` (see above), simply run the [`build.sh`](build.sh) to generate the
domain protocol types for `chromedp` command.

## Reference Output

The output of running `update.sh` and `build.sh` is below:
```sh
# change to chromedp-gen path
$ cd $GOPATH/src/github.com/chromedp/chromedp-gen

# update protocol.json to master
$ ./update.sh
BROWSER_PROTO: https://chromium.googlesource.com/chromium/src/+/master/third_party/WebKit/Source/core/inspector/browser_protocol.json?format=TEXT
JS_PROTO: https://chromium.googlesource.com/v8/v8/+/master/src/inspector/js_protocol.json?format=TEXT<Paste>

# update protocol.json to browser version 65.0.3299.5 and js version
$ ./update.sh 65.0.3299.5

$ ./build.sh
```
