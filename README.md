# pdlgen

`pdlgen` generates Go code for the commands, events and types of the
[Chrome DevTools Protocol][devtools-protocol]. It is a core component of the
[`chromedp`][chromedp] project. The generator follows the needs of `chromedp`,
but its aim is to produce [type safe, fast, efficient, idiomatic Go code][cdproto]
that any Go program can use to drive Chrome.

Every issue and every pull request for the `cdproto` package belongs in this
repository, and none belongs in the `cdproto` repository, because `cdproto` is
generated output.

A workflow in this repository regenerates `cdproto` every day. When the
generated code changes, the workflow commits it and tags a release. See
[Releases](#releases).

## Installing

Install `pdlgen` in the usual Go way:

```sh
go install github.com/chromedp/pdlgen@latest
```

## Using

`pdlgen` generates the [`github.com/chromedp/cdproto`][cdproto-godoc]
package and a `github.com/chromedp/cdproto/<domain>` package for each domain.
The `--out` option names the directory of a checkout of the
[`cdproto`][cdproto] repository:

```sh
pdlgen --chromium=157.0.8084.3 --v8=15.7.23 --out=../cdproto
```

The generator retrieves the protocol files from the [Chromium source
tree][chromium-src], caches them, and replaces the contents of the output
directory. It keeps `LICENSE`, `README.md`, `CHANGELOG.md`, `go.mod`, `go.sum`
and any name that starts with a dot.

### Command-line options

```sh
pdlgen --help
```

The options are:

- `--chromium` is the Chromium version of `browser_protocol.pdl`. Any Git tag or
  branch of the Chromium source tree works. The default is the latest version.
- `--v8` is the V8 version of `js_protocol.pdl`. The default is the version in the
  `DEPS` file of the Chromium version. `--latest` uses the latest V8 version.
- `--pdl` reads one combined protocol file and retrieves nothing.
- `--out` is the output directory.
- `--cache` is the cache directory. The default is `pdlgen` in the user
  cache directory.
- `--ttl` is how long a cached file is used. The default is 24 hours, and `0`
  forces a new retrieval.
- `--go-pkg` is the Go package name of the output. The default is
  `github.com/chromedp/cdproto`.
- `--go-wl` lists the files that the cleaning step keeps.
- `--no-clean` keeps the files that the generator did not write.
- `--debug` writes the files without `goimports` and `gofmt`.

## How it works

The generator reads the protocol, skips what the protocol deprecates, and fills
standard Go [`text/template`][text-template] templates. Two rewrites change the
types. One removes name stuttering, so `css.CSSStyle` becomes `css.Style`. The
other gives each inline enum a named type, so the `type` parameter of
`Input.dispatchKeyEvent` has the type `input.DispatchKeyEventType`. Types that two domains need
move to the `cdp` package, to avoid import cycles. [`docs/GENERATOR.md`](docs/GENERATOR.md)
describes every step.

## Releases

`cdproto` is tagged `v0.<Chromium major>.<patch>`, so the first release for
Chromium 157 is `v0.157.0`, and the next is `v0.157.1`. The `Update` workflow
makes the tags, and each tag and `CHANGELOG.md` in `cdproto` record the Chromium
and V8 versions and the count of API changes. The module stays at major version
0, and any release can contain an incompatible change, because the protocol
removes and renames things. The package reports its versions with
`cdproto.ChromiumVersion()` and `cdproto.V8Version()`.
[`docs/RELEASES.md`](docs/RELEASES.md) has the rules.

## Documents

| Document | Contents |
| --- | --- |
| [`AGENTS.md`](AGENTS.md) | The rules for coding agents, which apply to a person too |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | How to change this repository |
| [`docs/GENERATOR.md`](docs/GENERATOR.md) | The pipeline, the templates and how to change them |
| [`docs/RELEASES.md`](docs/RELEASES.md) | The workflows, the tags and the changelog |
| [`docs/API.md`](docs/API.md) | A proposed typed API that uses generics and iterators |
| [`docs/PLAN.md`](docs/PLAN.md) | Purpose, architecture, testing and open questions |
| [`docs/BACKLOG.md`](docs/BACKLOG.md) | Work that is known and not done |
| [`docs/PROGRESS.md`](docs/PROGRESS.md) | Where the work stands |
| [`docs/decisions/`](docs/decisions/README.md) | Every decision, one file each, named by date |

[devtools-protocol]: https://chromedevtools.github.io/devtools-protocol/
[chromedp]: https://github.com/chromedp
[cdproto]: https://github.com/chromedp/cdproto
[browser-protocol]: https://chromium.googlesource.com/chromium/src/+/main/third_party/blink/public/devtools_protocol/browser_protocol.pdl
[js-protocol]: https://chromium.googlesource.com/v8/v8/+/main/include/js_protocol.pdl
[chromium-src]: https://chromium.googlesource.com/chromium/src.git
[har-spec]: http://www.softwareishard.com/blog/har-12-spec/
[cdproto-godoc]: https://godoc.org/github.com/chromedp/cdproto
[text-template]: https://pkg.go.dev/text/template
